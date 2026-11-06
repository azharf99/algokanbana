package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"algokanbana/internal/auth"
	"algokanbana/internal/config"
	"algokanbana/internal/curriculum"
	"algokanbana/internal/lms"
	"algokanbana/internal/store"
	"algokanbana/internal/timeutil"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (s *Server) handleLogin(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Login     string `json:"login"`
		Password  string `json:"password"`
		VerifyLMS bool   `json:"verifyLms"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "format login JSON tidak valid")
		return
	}
	login := strings.TrimSpace(body.Login)
	if login == "" {
		writeError(w, http.StatusBadRequest, "username LMS tidak boleh kosong")
		return
	}
	if strings.TrimSpace(body.Password) == "" {
		writeError(w, http.StatusBadRequest, "password tidak boleh kosong")
		return
	}

	authSource := "local_fast"
	var lmsProfile *lms.UserProfile

	// In production mode or when explicit verification is requested, always verify credentials with upstream LMS!
	if s.cfg.AppMode == config.ModeProduction || body.VerifyLMS {
		prof, err := s.lms.VerifyCredentials(r.Context(), login, body.Password)
		if err != nil {
			writeError(w, http.StatusUnauthorized, "verifikasi LMS gagal: "+err.Error())
			return
		}
		lmsProfile = &prof
		authSource = "lms_verified"
	}

	tutor, err := s.store.GetTutorByLogin(r.Context(), login)
	if err != nil {
		// If verified with LMS, auto-provision tutor profile into SQLite!
		if lmsProfile != nil {
			nowStr := timeutil.FormatWIB(time.Now())
			fullName := strings.TrimSpace(lmsProfile.FullName)
			if fullName == "" {
				fullName = lmsProfile.Login
			}
			parts := strings.SplitN(fullName, " ", 2)
			first := parts[0]
			last := ""
			if len(parts) > 1 {
				last = parts[1]
			}
			newTutor := store.Tutor{
				TutorID:        uuid.NewString(),
				UserID:         uuid.NewString(),
				LMSUserID:      lmsProfile.LMSUserID,
				Login:          lmsProfile.Login,
				FirstName:      first,
				LastName:       last,
				FullName:       fullName,
				Email:          lmsProfile.EmailMask,
				Status:         "active",
				EmploymentType: "part_time",
				CountryCode:    "ID",
				CourseTypes:    []string{"coding"},
				LessonFormats:  []string{"group", "private"},
				DensityPref:    "default",
				LocalePref:     "id",
				AvailableRoles: []string{"tutor"},
				UpdatedAt:      nowStr,
			}
			if err := s.store.UpsertTutor(r.Context(), newTutor); err != nil {
				writeError(w, http.StatusInternalServerError, "gagal menyimpan profil tutor: "+err.Error())
				return
			}
			tutor = newTutor
		} else {
			writeError(w, http.StatusUnauthorized, "akun tidak ditemukan di database")
			return
		}
	} else if lmsProfile != nil {
		if tutor.LMSUserID == 0 && lmsProfile.LMSUserID != 0 {
			tutor.LMSUserID = lmsProfile.LMSUserID
			_ = s.store.UpsertTutor(r.Context(), tutor)
		}
	}

	// When LMS profile is verified, automatically sync real tutor groups into SQLite
	if lmsProfile != nil {
		go func(tutorID string) {
			rawGroups, err := s.lms.FetchTutorGroups(context.Background())
			if err == nil && len(rawGroups) > 0 {
				var groupsToSync []store.Group
				for _, rg := range rawGroups {
					fmtType := "group"
					if rg.Type == "intensive" {
						fmtType = "private"
					}
					isArch := rg.Status != "active"
					cCode, cName := parseCourseFromTitle(rg.Title)
					groupsToSync = append(groupsToSync, store.Group{
						GroupID:         fmt.Sprintf("%d", rg.ID),
						TutorID:         tutorID,
						Title:           rg.Title,
						CourseCode:      cCode,
						CourseName:      cName,
						Format:          fmtType,
						ScheduleSummary: formatScheduleFromTitle(rg.Title),
						StartDate:       rg.StartTime,
						StudentsCount:   1,
						LMSURL:          fmt.Sprintf("https://lms.algonova.id/group/view/%d", rg.ID),
						IsArchived:      isArch,
					})
				}
				_ = s.store.SyncTutorGroups(context.Background(), tutorID, groupsToSync)
			}
		}(tutor.TutorID)
	}

	claims := auth.SessionClaims{
		UserID:         tutor.UserID,
		TutorID:        tutor.TutorID,
		LMSUserID:      tutor.LMSUserID,
		Login:          tutor.Login,
		FullName:       tutor.FullName,
		ActiveRole:     "tutor",
		AvailableRoles: tutor.AvailableRoles,
	}
	if err := s.auth.SetCookie(w, claims, r.TLS != nil); err != nil {
		writeError(w, http.StatusInternalServerError, "gagal membuat sesi")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":         true,
		"authSource": authSource,
		"session":    claims,
		"tutor":      tutor,
	})
}

func parseCourseFromTitle(title string) (string, string) {
	upper := strings.ToUpper(title)
	for _, c := range curriculum.Courses {
		if strings.Contains(upper, " "+c.Code+" ") || strings.Contains(upper, "("+c.Code) || strings.Contains(upper, " "+c.Code) {
			return c.Code, c.Name
		}
	}
	if strings.Contains(upper, "PYTHON PRO") || strings.Contains(upper, "PP2") {
		return "PP2", "Python Pro (Year 2 - AI & Backend)"
	}
	if strings.Contains(upper, "ROBLOX") || strings.Contains(upper, "RB") {
		return "RB2.0", "Roblox Studio 2.0 (Lua Scripting)"
	}
	if strings.Contains(upper, "UNITY") || strings.Contains(upper, "GD") {
		return "GD", "Unity 3D Game Development (C#)"
	}
	if strings.Contains(upper, "FRONTEND") || strings.Contains(upper, "FE") {
		return "FE", "Frontend Development (HTML/CSS/JS/React)"
	}
	if strings.Contains(upper, "PYTHON") || strings.Contains(upper, "PS") {
		return "PS", "Python Start (Year 1)"
	}
	return "INT-PRV", "Intensive Private Coding Track (1-on-1)"
}

func formatScheduleFromTitle(title string) string {
	if idx := strings.Index(title, "("); idx != -1 {
		endIdx := strings.Index(title[idx:], ")")
		if endIdx != -1 {
			return strings.TrimSpace(title[idx+1 : idx+endIdx])
		}
	}
	return "Sesuai Jadwal LMS"
}

func (s *Server) handleLogout(w http.ResponseWriter, r *http.Request) {
	s.auth.ClearCookie(w)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleMe(w http.ResponseWriter, r *http.Request) {
	claims, err := s.auth.FromRequest(r)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false, "mode": s.cfg.AppMode})
		return
	}
	tutor, err := s.store.GetTutorByID(r.Context(), claims.TutorID)
	if err != nil {
		writeJSON(w, http.StatusOK, map[string]any{"authenticated": false, "mode": s.cfg.AppMode})
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"authenticated": true,
		"mode":          s.cfg.AppMode,
		"session":       claims,
		"tutor":         tutor,
	})
}

func (s *Server) handleSwitchRole(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var body struct {
		Role string `json:"role"`
	}
	if err := decodeJSON(r, &body); err != nil || (body.Role != "tutor" && body.Role != "team_lead") {
		writeError(w, http.StatusBadRequest, "role harus 'tutor' atau 'team_lead'")
		return
	}
	claims.ActiveRole = body.Role
	if err := s.auth.SetCookie(w, claims, r.TLS != nil); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "session": claims})
}

func (s *Server) handleBootstrap(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	tutor, err := s.store.GetTutorByID(r.Context(), claims.TutorID)
	if err != nil {
		writeError(w, http.StatusNotFound, "tutor tidak ditemukan")
		return
	}
	upcoming, _ := s.store.ListUpcomingLessons(r.Context(), claims.TutorID, 25)
	savedViews, _ := s.store.ListSavedViews(r.Context(), claims.TutorID)
	events, _ := s.store.ListPortalEvents(r.Context())
	tutorUnread, tlUnread, assessmentOpen := s.store.UnreadCounts(r.Context(), claims.TutorID)

	writeJSON(w, http.StatusOK, map[string]any{
		"ok":              true,
		"session":         claims,
		"tutor":           tutor,
		"upcomingLessons": upcoming,
		"savedViews":      savedViews,
		"portalEvents":    events,
		"coursesCount":    len(curriculum.Courses),
		"sync":            s.syncEng.GetStatus(),
		"badges": map[string]any{
			"tutorUnread":    tutorUnread,
			"tlUnread":       tlUnread,
			"assessmentOpen": assessmentOpen,
		},
		"kpiThresholds": map[string][2]int{
			"attendance": {80, 88},
			"discipline": {45, 50},
			"quality":    {68, 75},
		},
	})
}

func (s *Server) handleTriggerSync(w http.ResponseWriter, r *http.Request) {
	st := s.syncEng.TriggerSync(r.Context())
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "sync": st})
}

func (s *Server) handleSearch(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if len(q) < 1 {
		writeJSON(w, http.StatusOK, map[string]any{"students": []any{}, "groups": []any{}, "tasks": []any{}})
		return
	}
	res, err := s.store.SearchPalette(r.Context(), claims.TutorID, q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleCreateSavedView(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var body struct {
		PageKey     string `json:"pageKey"`
		Name        string `json:"name"`
		QueryString string `json:"queryString"`
	}
	if err := decodeJSON(r, &body); err != nil || strings.TrimSpace(body.Name) == "" {
		writeError(w, http.StatusBadRequest, "nama view wajib diisi")
		return
	}
	v, err := s.store.CreateSavedView(r.Context(), claims.TutorID, body.PageKey, body.Name, body.QueryString)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "view": v})
}

func (s *Server) handleDeleteSavedView(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	id := chi.URLParam(r, "id")
	if err := s.store.DeleteSavedView(r.Context(), claims.TutorID, id); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleUploadFiles(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	if err := r.ParseMultipartForm(10 << 20); err != nil {
		writeError(w, http.StatusBadRequest, "ukuran file maksimal 10 MB")
		return
	}
	kind := r.FormValue("kind") // "proof" | "avatar" | "cv" | "attachment"
	files := r.MultipartForm.File["files"]
	if len(files) == 0 && r.MultipartForm.File["file"] != nil {
		files = r.MultipartForm.File["file"]
	}
	if len(files) == 0 {
		writeError(w, http.StatusBadRequest, "tidak ada file yang diunggah")
		return
	}

	urls := make([]string, 0, len(files))
	for _, fh := range files {
		ext := strings.ToLower(filepath.Ext(fh.Filename))
		switch ext {
		case ".png", ".jpg", ".jpeg", ".webp", ".pdf":
		default:
			writeError(w, http.StatusBadRequest, "format file harus PNG, JPG, WEBP, atau PDF")
			return
		}
		src, err := fh.Open()
		if err != nil {
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		fname := fmt.Sprintf("%d_%s%s", time.Now().UnixMilli(), uuid.NewString()[:8], ext)
		dstPath := filepath.Join(s.cfg.UploadDir, fname)
		dst, err := os.Create(dstPath)
		if err != nil {
			_ = src.Close()
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		_, _ = io.Copy(dst, io.LimitReader(src, 10<<20))
		_ = src.Close()
		_ = dst.Close()
		urls = append(urls, "/uploads/"+fname)
	}

	if (kind == "avatar" || kind == "cv") && len(urls) > 0 {
		_ = s.store.SetTutorAssetURL(r.Context(), claims.TutorID, kind, urls[0])
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "urls": urls, "url": urls[0]})
}

func (s *Server) handleDashboardGroups(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	data, err := s.store.GetDashboardGroups(r.Context(), claims.TutorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleGroupDetail(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	groupID := chi.URLParam(r, "id")
	data, err := s.store.GetGroupDetail(r.Context(), claims.TutorID, groupID)
	if err != nil {
		writeError(w, http.StatusNotFound, "grup tidak ditemukan")
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleDashboardKPI(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	period := r.URL.Query().Get("period")
	if period == "" {
		period = "this_month"
	}
	data, err := s.store.GetMyKPI(r.Context(), claims.TutorID, period)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleListTLReports(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	list, err := s.store.ListTLReports(r.Context(), claims.TutorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"reports": list})
}

func (s *Server) handleCreateTLReport(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var body struct {
		ReportType string `json:"reportType"`
		GroupID    string `json:"groupId"`
		StudentID  string `json:"studentId"`
		Urgency    string `json:"urgency"`
		Summary    string `json:"summary"`
		Detail     string `json:"detail"`
		Message    string `json:"message"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "format laporan tidak valid")
		return
	}
	msg := strings.TrimSpace(body.Message)
	if msg == "" {
		msg = strings.TrimSpace(body.Summary)
		if strings.TrimSpace(body.Detail) != "" {
			msg += "\n\n" + strings.TrimSpace(body.Detail)
		}
	}
	if len(msg) < 10 {
		writeError(w, http.StatusBadRequest, "isi laporan ke Team Lead minimal 10 karakter")
		return
	}
	if body.ReportType == "" {
		body.ReportType = "student_issues"
	}
	if body.Urgency == "" {
		body.Urgency = "medium"
	}
	rep, err := s.store.CreateTLReport(r.Context(), store.TLReport{
		TutorID:    claims.TutorID,
		TutorName:  claims.FullName,
		ReportType: body.ReportType,
		GroupID:    strings.TrimSpace(body.GroupID), // Accepts numeric LMS IDs cleanly (Fixes Bug #1!)
		StudentID:  strings.TrimSpace(body.StudentID),
		Urgency:    body.Urgency,
		Message:    msg,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.sse.Broadcast("tl_report_created", rep)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "report": rep})
}

func (s *Server) handleListStudents(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	list, err := s.store.ListStudents(r.Context(), claims.TutorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"students": list})
}

func (s *Server) handleStudentDetail(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	id := chi.URLParam(r, "id")
	data, err := s.store.GetStudentDetail(r.Context(), claims.TutorID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "siswa tidak ditemukan")
		return
	}
	writeJSON(w, http.StatusOK, data)
}
