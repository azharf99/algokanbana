package api

import (
	"context"
	"fmt"
	"io"
	"log"
	"math"
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

	// When LMS profile is verified, automatically sync real tutor data into SQLite
	if lmsProfile != nil {
		s.performLMSSync(r.Context(), tutor.TutorID, tutor.Login, true)
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
	isSecure := s.auth.ResolveSecure(r, s.cfg.CookieSecure)
	if err := s.auth.SetCookie(w, claims, isSecure); err != nil {
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
	isSecure := s.auth.ResolveSecure(r, s.cfg.CookieSecure)
	if err := s.auth.SetCookie(w, claims, isSecure); err != nil {
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
	claims := getClaims(r)
	s.performLMSSync(r.Context(), claims.TutorID, claims.Login, false)
	writeJSON(w, http.StatusOK, map[string]any{
		"ok":      true,
		"sync":    s.syncEng.GetStatus(),
		"message": "Sinkronisasi LMS dimulai di latar belakang",
	})
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

func (s *Server) performLMSSync(ctx context.Context, tutorID, login string, syncPhase1Sync bool) {
	if syncPhase1Sync {
		p1Ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
		defer cancel()
		groups, err := s.syncLMSPhase1(p1Ctx, tutorID, login)
		if err == nil && len(groups) > 0 {
			go func() {
				p2Ctx, p2Cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer p2Cancel()
				s.syncLMSPhase2(p2Ctx, tutorID, login, groups)
			}()
		}
	} else {
		go func() {
			p1Ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
			defer cancel()
			groups, err := s.syncLMSPhase1(p1Ctx, tutorID, login)
			if err == nil && len(groups) > 0 {
				p2Ctx, p2Cancel := context.WithTimeout(context.Background(), 5*time.Minute)
				defer p2Cancel()
				s.syncLMSPhase2(p2Ctx, tutorID, login, groups)
			}
		}()
	}
}

func (s *Server) syncLMSPhase1(ctx context.Context, tutorID, login string) ([]store.Group, error) {
	log.Printf("[LMS-SYNC] Starting Phase 1 sync for tutor %s (%s)...", tutorID, login)

	dashGroups, errDash := s.lms.FetchTeacherDashboardGroups(ctx)
	if errDash != nil {
		log.Printf("[LMS-SYNC WARN] FetchTeacherDashboardGroups: %v", errDash)
	}
	rawGroups, errGroups := s.lms.FetchTutorGroups(ctx)
	if errGroups != nil {
		log.Printf("[LMS-SYNC WARN] FetchTutorGroups: %v", errGroups)
	}

	if errDash != nil && errGroups != nil {
		log.Printf("[LMS-SYNC ERR] Both LMS group endpoints failed for tutor %s", tutorID)
		return nil, fmt.Errorf("both LMS group endpoints failed")
	}

	dashMap := make(map[int64]lms.RawTeacherDashboardGroup, len(dashGroups))
	for _, dg := range dashGroups {
		dashMap[dg.ID] = dg
	}

	groupMap := make(map[string]store.Group)
	var upcomingLessons []store.Lesson

	for _, rg := range rawGroups {
		fmtType := "group"
		if rg.Type == "intensive" {
			fmtType = "private"
		}
		isArch := rg.Status != "active"
		cCode, cName := parseCourseFromTitle(rg.Title)
		gid := fmt.Sprintf("%d", rg.ID)
		stCount := 1
		avgAtt := 0
		avgProg := 0

		if dg, ok := dashMap[rg.ID]; ok {
			stCount = dg.ActiveStudentsCount
			if len(dg.LessonStats) > 0 {
				var totalAttRatio, totalProg float64
				var statCount int
				for _, st := range dg.LessonStats {
					if st.ActiveStudentsCount > 0 {
						totalAttRatio += float64(st.AttendedStudentsCount) / float64(st.ActiveStudentsCount)
					}
					totalProg += st.RegularTaskPercent
					statCount++
				}
				if statCount > 0 {
					avgAtt = int(math.Round((totalAttRatio / float64(statCount)) * 100))
					avgProg = int(math.Round(totalProg / float64(statCount)))
				}
			}
			if dg.NextGroupLesson != nil && dg.NextGroupLesson.StartTime != "" {
				stTime, err := timeutil.ParseLMSMoscowTime(dg.NextGroupLesson.StartTime)
				if err == nil {
					dur := 60
					if fmtType == "private" {
						dur = 50
					}
					stStr := timeutil.FormatWIB(stTime)
					endStr := timeutil.FormatWIB(stTime.Add(time.Duration(dur) * time.Minute))
					upcomingLessons = append(upcomingLessons, store.Lesson{
						LessonID:           fmt.Sprintf("dash_%d_%d", dg.ID, dg.NextGroupLesson.LessonID),
						TutorID:            tutorID,
						GroupID:            gid,
						GroupTitle:         rg.Title,
						CourseCode:         cCode,
						Format:             fmtType,
						LessonTitle:        dg.NextGroupLesson.LessonTitle,
						LessonNumber:       dg.NextGroupLesson.LessonPosition,
						CourseLessonNumber: dg.NextGroupLesson.LessonPosition,
						StartsAt:           stStr,
						EndsAt:             endStr,
						DurationMinutes:    dur,
						MeetingURL:         "",
						PrepURL:            "",
						LMSURL:             fmt.Sprintf("https://lms.algonova.id/group/view/%d", rg.ID),
						IsCompleted:        false,
					})
				}
			}
		}

		groupMap[gid] = store.Group{
			GroupID:          gid,
			TutorID:          tutorID,
			Title:            rg.Title,
			CourseCode:       cCode,
			CourseName:       cName,
			Format:           fmtType,
			ScheduleSummary:  formatScheduleFromTitle(rg.Title),
			StartDate:        rg.StartTime,
			StudentsCount:    stCount,
			AvgAttendancePct: avgAtt,
			AvgProgressPct:   avgProg,
			LMSURL:           fmt.Sprintf("https://lms.algonova.id/group/view/%d", rg.ID),
			IsArchived:       isArch,
		}
	}

	for _, dg := range dashGroups {
		gid := fmt.Sprintf("%d", dg.ID)
		if _, exists := groupMap[gid]; !exists {
			cCode, cName := parseCourseFromTitle(dg.GroupTitle)
			fmtType := "group"
			if strings.Contains(strings.ToLower(dg.GroupTitle), "intensive") || strings.Contains(strings.ToLower(dg.GroupTitle), "private") {
				fmtType = "private"
			}
			groupMap[gid] = store.Group{
				GroupID:          gid,
				TutorID:          tutorID,
				Title:            dg.GroupTitle,
				CourseCode:       cCode,
				CourseName:       cName,
				Format:           fmtType,
				ScheduleSummary:  formatScheduleFromTitle(dg.GroupTitle),
				StartDate:        "",
				StudentsCount:    dg.ActiveStudentsCount,
				AvgAttendancePct: 0,
				AvgProgressPct:   0,
				LMSURL:           fmt.Sprintf("https://lms.algonova.id/group/view/%d", dg.ID),
				IsArchived:       false,
			}
		}
	}

	var groupsToSync []store.Group
	for _, g := range groupMap {
		groupsToSync = append(groupsToSync, g)
	}

	if err := s.store.SyncTutorGroups(ctx, tutorID, groupsToSync); err != nil {
		log.Printf("[LMS-SYNC ERR] SyncTutorGroups failed: %v", err)
	} else {
		log.Printf("[LMS-SYNC] Phase 1: Synced %d groups into SQLite", len(groupsToSync))
	}

	if len(upcomingLessons) > 0 {
		if err := s.store.SyncGroupLessons(ctx, tutorID, upcomingLessons); err != nil {
			log.Printf("[LMS-SYNC ERR] Sync upcoming lessons failed: %v", err)
		} else {
			log.Printf("[LMS-SYNC] Phase 1: Synced %d upcoming lessons into SQLite", len(upcomingLessons))
		}
	}

	if err := s.store.SyncWeeklyRoutineTasks(ctx, tutorID); err != nil {
		log.Printf("[LMS-SYNC ERR] SyncWeeklyRoutineTasks failed: %v", err)
	}

	return groupsToSync, nil
}

func (s *Server) syncLMSPhase2(ctx context.Context, tutorID, login string, groups []store.Group) {
	start := time.Now()
	log.Printf("[LMS-SYNC] Starting Phase 2 sync (students, lessons, slots) for tutor %s...", tutorID)

	var totalStudents, totalLessons int
	for _, g := range groups {
		if g.IsArchived {
			continue
		}
		var gidInt int64
		_, err := fmt.Sscanf(g.GroupID, "%d", &gidInt)
		if err != nil || gidInt <= 0 {
			continue
		}

		time.Sleep(50 * time.Millisecond)

		// 1. Fetch Students
		rawStudents, err := s.lms.FetchGroupStudents(ctx, gidInt)
		if err != nil {
			log.Printf("[LMS-SYNC WARN] FetchGroupStudents for group %s failed: %v", g.GroupID, err)
		} else if len(rawStudents) > 0 {
			var studentsToSync []store.Student
			for _, rs := range rawStudents {
				rem := 0
				if rs.Wallet != nil {
					rem = rs.Wallet.Credits
				}
				if rem <= 0 {
					for _, ag := range rs.ActiveGroups {
						if ag.ID == gidInt && ag.GroupWallet != nil {
							rem = ag.GroupWallet.Credits
							break
						}
					}
				}
				if rem < 0 {
					rem = 0
				}

				pName := strings.TrimSpace(rs.ParentName)
				if pName == "" {
					pName = "Orang Tua " + rs.FullName
				}

				studentsToSync = append(studentsToSync, store.Student{
					StudentID:             fmt.Sprintf("%d", rs.ID),
					TutorID:               tutorID,
					FullName:              rs.FullName,
					GroupID:               g.GroupID,
					GroupTitle:            g.Title,
					GroupFormat:           g.Format,
					CourseCode:            g.CourseCode,
					CourseName:            g.CourseName,
					AttendancePct:         g.AvgAttendancePct,
					ProgressPct:           g.AvgProgressPct,
					LessonsAttended:       0,
					LessonsTotal:          32,
					LessonsRemaining:      rem,
					TotalLessonsPurchased: rem + 8,
					LastLessonAt:          "",
					ParentName:            pName,
					ParentPhone:           rs.Phone,
					LMSURL:                fmt.Sprintf("https://lms.algonova.id/group/student/view/%d", rs.ID),
					DiagnosticPct:         85,
					DiagnosticGoal:        "Pengembangan Coding",
				})
			}
			if err := s.store.SyncGroupStudents(ctx, tutorID, studentsToSync); err != nil {
				log.Printf("[LMS-SYNC ERR] SyncGroupStudents group %s failed: %v", g.GroupID, err)
			} else {
				totalStudents += len(studentsToSync)
			}
		}

		time.Sleep(50 * time.Millisecond)

		// 2. Fetch Lessons
		rawLessons, err := s.lms.FetchGroupLessons(ctx, gidInt)
		if err != nil {
			log.Printf("[LMS-SYNC WARN] FetchGroupLessons for group %s failed: %v", g.GroupID, err)
		} else if len(rawLessons) > 0 {
			var lessonsToSync []store.Lesson
			dur := 60
			if g.Format == "private" {
				dur = 50
			}
			for _, rl := range rawLessons {
				stTime, err := timeutil.ParseLMSMoscowTime(rl.StartTime)
				var stStr, endStr string
				if err == nil {
					stStr = timeutil.FormatWIB(stTime)
					endStr = timeutil.FormatWIB(stTime.Add(time.Duration(dur) * time.Minute))
				} else {
					stStr = rl.StartTime
					endStr = rl.StartTime
				}
				isComp := rl.Status == 2 || rl.FinishedAt != ""
				lessonsToSync = append(lessonsToSync, store.Lesson{
					LessonID:           fmt.Sprintf("%d", rl.ID),
					TutorID:            tutorID,
					GroupID:            g.GroupID,
					GroupTitle:         g.Title,
					CourseCode:         g.CourseCode,
					Format:             g.Format,
					LessonTitle:        rl.Title,
					LessonNumber:       rl.Number,
					CourseLessonNumber: rl.Number,
					StartsAt:           stStr,
					EndsAt:             endStr,
					DurationMinutes:    dur,
					MeetingURL:         "",
					PrepURL:            "",
					LMSURL:             fmt.Sprintf("https://lms.algonova.id/group/lesson/view/%d", rl.ID),
					IsCompleted:        isComp,
				})
			}
			if err := s.store.SyncGroupLessons(ctx, tutorID, lessonsToSync); err != nil {
				log.Printf("[LMS-SYNC ERR] SyncGroupLessons group %s failed: %v", g.GroupID, err)
			} else {
				totalLessons += len(lessonsToSync)
			}
		}
	}

	durTotal := time.Since(start)
	syncSnap := s.syncEng.RecordSyncSuccess(durTotal)
	log.Printf("[LMS-SYNC] Full sync completed for tutor %s: %d groups, %d students, %d lessons in %v",
		tutorID, len(groups), totalStudents, totalLessons, durTotal)

	s.sse.Broadcast("sync_completed", map[string]any{
		"ok":             true,
		"tutorId":        tutorID,
		"groupsCount":    len(groups),
		"syncedStudents": totalStudents,
		"syncedLessons":  totalLessons,
		"sync":           syncSnap,
	})
}
