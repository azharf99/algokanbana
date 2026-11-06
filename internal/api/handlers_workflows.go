package api

import (
	"fmt"
	"html"
	"net/http"
	"runtime"
	"strings"
	"time"

	"algokanbana/internal/curriculum"
	"algokanbana/internal/store"

	"github.com/go-chi/chi/v5"
)

func (s *Server) handleListTasks(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	tasks, err := s.store.ListTasks(r.Context(), claims.TutorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tasks": tasks})
}

func (s *Server) handleCreateTask(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var body struct {
		Title       string `json:"title"`
		Description string `json:"description"`
		DueAt       string `json:"dueAt"`
		Priority    string `json:"priority"`
		ThreadID    string `json:"threadId"`
	}
	if err := decodeJSON(r, &body); err != nil || len(strings.TrimSpace(body.Title)) < 3 {
		writeError(w, http.StatusBadRequest, "judul tugas minimal 3 karakter")
		return
	}
	tkType := "personal"
	if body.ThreadID != "" {
		tkType = "chat_task"
	}
	tk, err := s.store.CreateTask(r.Context(), store.Task{
		TutorID:     claims.TutorID,
		Title:       strings.TrimSpace(body.Title),
		Description: strings.TrimSpace(body.Description),
		TaskType:    tkType,
		Priority:    body.Priority,
		DueAt:       body.DueAt,
		ThreadID:    body.ThreadID,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": tk})
}

func (s *Server) handleBulkUpdateTasks(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var body struct {
		TaskIDs []string `json:"taskIds"`
		Action  string   `json:"action"` // "complete" | "escalate" | "cancel" | "reopen"
		Reason  string   `json:"reason"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "format request tidak valid")
		return
	}
	tasks, err := s.store.BulkUpdateTasks(r.Context(), claims.TutorID, body.TaskIDs, body.Action, body.Reason)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.Action == "escalate" {
		s.sse.Broadcast("task_escalated", map[string]any{"count": len(body.TaskIDs)})
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "tasks": tasks})
}

func (s *Server) handleMoveTask(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	taskID := chi.URLParam(r, "id")
	var body struct {
		TargetStatus string `json:"targetStatus"` // "assigned" | "overdue" | "escalated" | "completed" | "cancelled"
		Reason       string `json:"reason"`
	}
	if err := decodeJSON(r, &body); err != nil || body.TargetStatus == "" {
		writeError(w, http.StatusBadRequest, "targetStatus wajib diisi")
		return
	}
	tk, err := s.store.MoveTask(r.Context(), claims.TutorID, taskID, body.TargetStatus, body.Reason)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	if body.TargetStatus == "escalated" {
		s.sse.Broadcast("task_escalated", map[string]any{"taskId": taskID})
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "task": tk})
}

func (s *Server) handleGetUrgentSubs(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	eligible, _ := s.store.ListEligibleUrgentSubLessons(r.Context(), claims.TutorID)
	requests, _ := s.store.ListUrgentSubRequests(r.Context(), claims.TutorID)
	writeJSON(w, http.StatusOK, map[string]any{
		"eligibleLessons": eligible,
		"requests":        requests,
	})
}

func (s *Server) handleCreateUrgentSub(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var body struct {
		LMSLessonID string   `json:"lmsLessonId"`
		CourseTitle string   `json:"courseTitle"`
		Reason      string   `json:"reason"`
		ProofURLs   []string `json:"proofUrls"`
	}
	if err := decodeJSON(r, &body); err != nil || body.LMSLessonID == "" || len(strings.TrimSpace(body.Reason)) < 5 {
		writeError(w, http.StatusBadRequest, "pilih sesi kelas dan isi alasan minimal 5 karakter")
		return
	}
	req, err := s.store.CreateUrgentSubRequest(r.Context(), store.UrgentSubRequest{
		TutorID:     claims.TutorID,
		TutorName:   claims.FullName,
		LMSLessonID: body.LMSLessonID,
		CourseTitle: body.CourseTitle,
		Reason:      strings.TrimSpace(body.Reason),
		ProofURLs:   body.ProofURLs,
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.sse.Broadcast("urgent_sub_created", req)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "request": req})
}

func (s *Server) handleWithdrawUrgentSub(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.UpdateUrgentSubStatus(r.Context(), id, "withdrawn"); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleGetRegularFeedback(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	week := r.URL.Query().Get("week")
	data, err := s.store.GetRegularFeedbackWeek(r.Context(), claims.TutorID, week)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handlePreviewRegularFeedback(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Locale      string `json:"locale"`
		StudentName string `json:"studentName"`
		CourseCode  string `json:"courseCode"`
		MonthNo     int    `json:"monthNo"`
		Attendance  string `json:"attendance"`
		Engagement  string `json:"engagement"`
		Tasks       string `json:"tasks"`
		Custom1     string `json:"custom1"`
		Custom2     string `json:"custom2"`
		ProjectURL  string `json:"projectUrl"`
		Stars       int    `json:"stars"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "format preview tidak valid")
		return
	}
	text := curriculum.AssembleRegularFeedback(
		body.Locale, body.StudentName, body.CourseCode, body.MonthNo,
		body.Attendance, body.Engagement, body.Tasks,
		body.Custom1, body.Custom2, body.ProjectURL, body.Stars,
	)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "assembledText": text})
}

func (s *Server) handleSaveRegularFeedback(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var body store.RegularFeedbackReport
	if err := decodeJSON(r, &body); err != nil || body.LMSLessonID == "" || body.LMSStudentID == "" {
		writeError(w, http.StatusBadRequest, "sesi pertemuan dan siswa wajib dipilih")
		return
	}
	body.TutorID = claims.TutorID
	locale := r.URL.Query().Get("locale")
	if locale == "" {
		locale = "id"
	}
	saved, err := s.store.SaveRegularFeedbackReport(r.Context(), locale, body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "report": saved})
}

func (s *Server) handlePrintRegularFeedback(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	id := chi.URLParam(r, "id")
	rp, err := s.store.GetRegularFeedbackByID(r.Context(), claims.TutorID, id)
	if err != nil {
		http.Error(w, "Laporan tidak ditemukan", http.StatusNotFound)
		return
	}

	stars := strings.Repeat("★", rp.Stars) + strings.Repeat("☆", 5-rp.Stars)
	bodyHTML := strings.ReplaceAll(html.EscapeString(rp.AssembledText), "\n", "<br>")
	doc := fmt.Sprintf(`<!DOCTYPE html>
<html lang="id">
<head>
<meta charset="utf-8">
<title>Regular Feedback — %s</title>
<style>
  @page { size: A4; margin: 18mm; }
  body { font-family: 'Segoe UI', system-ui, sans-serif; color: #0f172a; background: #f8fafc; margin: 0; padding: 24px; }
  .sheet { max-width: 720px; margin: 0 auto; background: #ffffff; border: 1px solid #e2e8f0; border-radius: 16px; padding: 36px; box-shadow: 0 10px 30px rgba(15,23,42,0.06); }
  .top { display: flex; justify-content: space-between; align-items: center; border-bottom: 2px solid #6366f1; padding-bottom: 18px; margin-bottom: 24px; }
  .brand { font-size: 22px; font-weight: 800; color: #4f46e5; letter-spacing: -0.02em; }
  .badge { background: #eef2ff; color: #4338ca; padding: 6px 12px; border-radius: 999px; font-size: 12px; font-weight: 700; }
  .meta { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; background: #f8fafc; padding: 16px; border-radius: 12px; margin-bottom: 24px; font-size: 13px; }
  .content { line-height: 1.7; font-size: 14.5px; color: #1e293b; }
  .stars { margin-top: 20px; font-size: 22px; color: #f59e0b; }
  .footer { margin-top: 32px; padding-top: 16px; border-top: 1px solid #e2e8f0; display: flex; justify-content: space-between; font-size: 12px; color: #64748b; }
  .toolbar { max-width: 720px; margin: 0 auto 16px; display: flex; justify-content: flex-end; gap: 10px; }
  button { background: #4f46e5; color: #fff; border: 0; padding: 10px 18px; border-radius: 8px; font-weight: 600; cursor: pointer; }
  @media print { body { background: #fff; padding: 0; } .sheet { box-shadow: none; border: none; padding: 0; } .toolbar { display: none; } }
</style>
</head>
<body>
  <div class="toolbar"><button onclick="window.print()">Cetak / Simpan PDF (A4)</button></div>
  <div class="sheet">
    <div class="top">
      <div>
        <div class="brand">ALGONOVA INDONESIA</div>
        <div style="font-size:13px;color:#64748b;margin-top:4px;">Laporan Perkembangan Belajar Siswa (Regular Feedback)</div>
      </div>
      <div class="badge">Pertemuan #%d • Bulan ke-%d</div>
    </div>
    <div class="meta">
      <div><strong>Nama Siswa:</strong> %s</div>
      <div><strong>Kursus:</strong> %s (%s)</div>
      <div><strong>Grup Kelas:</strong> %s</div>
      <div><strong>Tutor Pengajar:</strong> %s</div>
    </div>
    <div class="content">%s</div>
    <div class="stars">%s</div>
    <div class="footer">
      <span>Dicetak melalui AlgoKanbana High-Performance Tutor Portal</span>
      <span>Diperbarui: %s</span>
    </div>
  </div>
</body>
</html>`,
		html.EscapeString(rp.StudentName), rp.LessonNumber, rp.MonthNo,
		html.EscapeString(rp.StudentName), html.EscapeString(rp.CourseName), html.EscapeString(rp.CourseCode),
		html.EscapeString(rp.GroupTitle), html.EscapeString(claims.FullName),
		bodyHTML, stars, html.EscapeString(rp.UpdatedAt[:10]))

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write([]byte(doc))
}

func (s *Server) handleListMTFeedback(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	list, err := s.store.ListMTFeedback(r.Context(), claims.TutorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"entries": list})
}

func (s *Server) handleCreateMTFeedback(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var body store.MTFeedbackEntry
	if err := decodeJSON(r, &body); err != nil || strings.TrimSpace(body.LessonTopic) == "" {
		writeError(w, http.StatusBadRequest, "topik pelajaran wajib diisi")
		return
	}
	body.TutorID = claims.TutorID
	saved, err := s.store.CreateMTFeedback(r.Context(), body)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "entry": saved})
}

func (s *Server) handleGetSchedule(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	from := r.URL.Query().Get("from")
	to := r.URL.Query().Get("to")
	data, err := s.store.GetScheduleBundle(r.Context(), claims.TutorID, from, to)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleUpdateAvailability(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var body struct {
		Slots  []store.SlotCoord `json:"slots"`
		Status string            `json:"status"` // "open" | "closed"
	}
	if err := decodeJSON(r, &body); err != nil || len(body.Slots) == 0 {
		writeError(w, http.StatusBadRequest, "pilih minimal 1 slot jadwal")
		return
	}
	results, err := s.store.UpdateAvailabilitySlots(r.Context(), claims.TutorID, body.Slots, body.Status)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "results": results})
}

func (s *Server) handleCreateReschedule(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var body struct {
		LMSLessonID      string   `json:"lmsLessonId"`
		ReqType          string   `json:"type"` // "otr" | "permanent"
		OriginalStartsAt string   `json:"originalStartsAt"`
		NewStartsAt      string   `json:"newStartsAt"`
		Note             string   `json:"note"`
		ProofURLs        []string `json:"proofUrls"`
	}
	if err := decodeJSON(r, &body); err != nil || body.LMSLessonID == "" || body.NewStartsAt == "" {
		writeError(w, http.StatusBadRequest, "pilih sesi kelas dan tentukan tanggal/jam baru")
		return
	}
	created, err := s.store.CreateRescheduleRequest(r.Context(), store.RescheduleRequest{
		TutorID:          claims.TutorID,
		TutorName:        claims.FullName,
		LMSLessonID:      body.LMSLessonID,
		ReqType:          body.ReqType,
		OriginalStartsAt: body.OriginalStartsAt,
		NewStartsAt:      body.NewStartsAt,
		Note:             strings.TrimSpace(body.Note),
		ProofURLs:        body.ProofURLs,
	}, time.Now())
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.sse.Broadcast("reschedule_created", created)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "request": created})
}

func (s *Server) handleWithdrawReschedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if err := s.store.UpdateRescheduleStatus(r.Context(), id, "withdrawn", "Ditarik oleh tutor"); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleCreateWorkloadRequest(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var body struct {
		ReqType string `json:"reqType"`
		Reason  string `json:"reason"`
	}
	if err := decodeJSON(r, &body); err != nil || len(strings.TrimSpace(body.Reason)) < 10 {
		writeError(w, http.StatusBadRequest, "alasan tambahan beban kerja minimal 10 karakter")
		return
	}
	wReq, err := s.store.CreateWorkloadRequest(r.Context(), store.WorkloadRequest{
		TutorID:   claims.TutorID,
		TutorName: claims.FullName,
		ReqType:   body.ReqType,
		Reason:    strings.TrimSpace(body.Reason),
	})
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	// Also open a communication thread to Team Lead matching NanoKanbana's submitRequest routing
	_, _ = s.store.CreateSupportRequestThread(r.Context(), claims.TutorID, claims.FullName, body.ReqType, "medium", "", "", body.Reason, "", "direct")
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "request": wReq})
}

func (s *Server) handleGetAssessments(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	data, err := s.store.GetAssessmentBundle(r.Context(), claims.TutorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, data)
}

func (s *Server) handleSubmitAssessment(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	cycle := chi.URLParam(r, "cycle")
	var body struct {
		Programs []string       `json:"programs"`
		Answers  map[string]any `json:"answers"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "format jawaban tidak valid")
		return
	}
	sub, err := s.store.SubmitAssessment(r.Context(), claims.TutorID, cycle, body.Programs, body.Answers)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "submission": sub})
}

func (s *Server) handleListKBArticles(w http.ResponseWriter, r *http.Request) {
	cat := r.URL.Query().Get("category")
	q := r.URL.Query().Get("q")
	list, err := s.store.ListKBArticles(r.Context(), cat, q)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"articles": list})
}

func (s *Server) handleListThreads(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	list, err := s.store.ListThreads(r.Context(), claims.TutorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"threads": list})
}

func (s *Server) handleGetThread(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	th, err := s.store.GetThread(r.Context(), id)
	if err != nil {
		writeError(w, http.StatusNotFound, "percakapan tidak ditemukan")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"thread": th})
}

func (s *Server) handleCreateSupportRequest(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var body struct {
		RequestType       string `json:"requestType"`
		Priority          string `json:"priority"`
		RelatedStudentRef string `json:"relatedStudentRef"`
		RelatedGroupRef   string `json:"relatedGroupRef"`
		Description       string `json:"description"`
		AttachmentURL     string `json:"attachmentUrl"`
		Origin            string `json:"origin"`
	}
	if err := decodeJSON(r, &body); err != nil || len(strings.TrimSpace(body.Description)) < 10 {
		writeError(w, http.StatusBadRequest, "deskripsi permintaan minimal 10 karakter")
		return
	}
	th, err := s.store.CreateSupportRequestThread(
		r.Context(), claims.TutorID, claims.FullName,
		body.RequestType, body.Priority, body.RelatedStudentRef, body.RelatedGroupRef,
		body.Description, body.AttachmentURL, body.Origin,
	)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.sse.Broadcast("thread_updated", th)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "thread": th})
}

func (s *Server) handleReplyThread(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	id := chi.URLParam(r, "id")
	var body struct {
		Body          string `json:"body"`
		AttachmentURL string `json:"attachmentUrl"`
	}
	if err := decodeJSON(r, &body); err != nil || strings.TrimSpace(body.Body) == "" {
		writeError(w, http.StatusBadRequest, "isi pesan tidak boleh kosong")
		return
	}
	senderName := claims.FullName
	if claims.ActiveRole == "team_lead" {
		senderName = "Team Lead Algonova"
	}
	msg, err := s.store.ReplyThread(r.Context(), id, claims.ActiveRole, senderName, "text", body.Body, body.AttachmentURL)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.sse.Broadcast("message_created", msg)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "message": msg})
}

func (s *Server) handleMarkThreadRead(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	id := chi.URLParam(r, "id")
	_ = s.store.MarkThreadRead(r.Context(), id, claims.ActiveRole)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleEscalateThread(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	id := chi.URLParam(r, "id")
	var body struct {
		Severity   string `json:"severity"`
		TargetTeam string `json:"targetTeam"`
		Reason     string `json:"reason"`
	}
	if err := decodeJSON(r, &body); err != nil || len(strings.TrimSpace(body.Reason)) < 10 {
		writeError(w, http.StatusBadRequest, "alasan eskalasi minimal 10 karakter")
		return
	}
	if err := s.store.EscalateThread(r.Context(), id, claims.FullName, body.Severity, body.TargetTeam, body.Reason); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.sse.Broadcast("thread_escalated", map[string]any{"threadId": id, "targetTeam": body.TargetTeam})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleGetProfile(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	tutor, err := s.store.GetTutorByID(r.Context(), claims.TutorID)
	if err != nil {
		writeError(w, http.StatusNotFound, "profil tidak ditemukan")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"tutor": tutor})
}

func (s *Server) handleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var in store.UpdateProfileInput
	if err := decodeJSON(r, &in); err != nil {
		writeError(w, http.StatusBadRequest, "input profil tidak valid")
		return
	}
	updated, err := s.store.UpdateTutorProfile(r.Context(), claims.TutorID, in)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "tutor": updated})
}

func (s *Server) handleTelegramAction(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	var body struct {
		Action string `json:"action"` // "generate_code" | "unlink" | "relink_demo"
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "aksi telegram tidak valid")
		return
	}
	res, err := s.store.UpdateTelegramAction(r.Context(), claims.TutorID, body.Action)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	tutor, _ := s.store.GetTutorByID(r.Context(), claims.TutorID)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "data": res, "tutor": tutor})
}

// Team Lead Demo & Approval Handlers (for end-to-end demo to Algonova leadership)
func (s *Server) handleTLDemoQueue(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	q, err := s.store.GetTLDemoQueue(r.Context(), claims.TutorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, q)
}

func (s *Server) handleTLReviewReschedule(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Status string `json:"status"` // "approved" | "rejected"
		TLNote string `json:"tlNote"`
	}
	if err := decodeJSON(r, &body); err != nil || (body.Status != "approved" && body.Status != "rejected") {
		writeError(w, http.StatusBadRequest, "status harus approved atau rejected")
		return
	}
	if err := s.store.UpdateRescheduleStatus(r.Context(), id, body.Status, body.TLNote); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.sse.Broadcast("reschedule_reviewed", map[string]any{"requestId": id, "status": body.Status})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleTLReviewReport(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Status  string `json:"status"` // "acknowledged" | "resolved"
		TLReply string `json:"tlReply"`
	}
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "format review tidak valid")
		return
	}
	if body.Status == "" {
		body.Status = "resolved"
	}
	if err := s.store.ReviewTLReport(r.Context(), id, body.Status, body.TLReply); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.sse.Broadcast("tl_report_reviewed", map[string]any{"reportId": id, "status": body.Status})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (s *Server) handleTLReviewUrgentSub(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Status string `json:"status"` // "approved"
	}
	_ = decodeJSON(r, &body)
	if body.Status == "" {
		body.Status = "approved"
	}
	if err := s.store.UpdateUrgentSubStatus(r.Context(), id, body.Status); err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	s.sse.Broadcast("urgent_sub_reviewed", map[string]any{"requestId": id, "status": body.Status})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

// handleBenchmark executes real server-side queries and returns live timing & RAM metrics
// side-by-side against the audited NanoKanbana latencies for leadership presentation.
func (s *Server) handleBenchmark(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	ctx := r.Context()

	measure := func(fn func()) float64 {
		t0 := time.Now()
		fn()
		ms := float64(time.Since(t0).Microseconds()) / 1000.0
		if ms < 0.05 {
			ms = 0.05
		}
		return ms
	}

	schedMs := measure(func() { _, _ = s.store.GetScheduleBundle(ctx, claims.TutorID, "", "") })
	dashMs := measure(func() { _, _ = s.store.GetDashboardGroups(ctx, claims.TutorID) })
	taskMs := measure(func() { _, _ = s.store.ListTasks(ctx, claims.TutorID) })
	fbMs := measure(func() { _, _ = s.store.GetRegularFeedbackWeek(ctx, claims.TutorID, "") })
	stuMs := measure(func() { _, _ = s.store.ListStudents(ctx, claims.TutorID) })
	commMs := measure(func() { _, _ = s.store.ListThreads(ctx, claims.TutorID) })
	assessMs := measure(func() { _, _ = s.store.GetAssessmentBundle(ctx, claims.TutorID) })

	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)

	rows := []map[string]any{
		{"module": "Jadwal & 217 Slot Ketersediaan", "oldMs": 12175.0, "newMs": schedMs, "speedup": int(12175.0 / schedMs)},
		{"module": "Dasbor (Grup & Kehadiran)", "oldMs": 11834.0, "newMs": dashMs, "speedup": int(11834.0 / dashMs)},
		{"module": "Bantuan & Komunikasi (Inbox)", "oldMs": 7505.0, "newMs": commMs, "speedup": int(7505.0 / commMs)},
		{"module": "Papan Tugas (Taskboard Kanban)", "oldMs": 4569.0, "newMs": taskMs, "speedup": int(4569.0 / taskMs)},
		{"module": "Tutor Assessment + Riwayat", "oldMs": 4204.0, "newMs": assessMs, "speedup": int(4204.0 / assessMs)},
		{"module": "Regular Feedback (32 Kursus)", "oldMs": 4011.0, "newMs": fbMs, "speedup": int(4011.0 / fbMs)},
		{"module": "Daftar Siswa & Filter Kuota", "oldMs": 3870.0, "newMs": stuMs, "speedup": int(3870.0 / stuMs)},
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"goVersion":     runtime.Version(),
		"goroutines":    runtime.NumGoroutine(),
		"heapAllocMB":   float64(mem.Alloc) / (1024 * 1024),
		"sysMemMB":      float64(mem.Sys) / (1024 * 1024),
		"comparisons":   rows,
		"bugsFixed":     5,
		"oldStack":      "Next.js Serverless (Seoul icn1) + PHP 8.3 Laravel (Railway) + Live Google Sheets",
		"newStack":      "Golang Single Static Binary + SQLite WAL + Embedded Svelte 5 (Docker FROM scratch)",
		"safeModeGuard": s.cfg.SafeReadonlyUpstream,
	})
}

func (s *Server) handleGetRoutinesSummary(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	summary, err := s.store.GetRoutinesSummary(r.Context(), claims.TutorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, summary)
}

func (s *Server) handleListSurveys(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	list, err := s.store.ListSurveyTargets(r.Context(), claims.TutorID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"surveys": list, "targets": list})
}

func (s *Server) handleGetSurvey(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	id := chi.URLParam(r, "id")
	target, err := s.store.GetSurveyTarget(r.Context(), claims.TutorID, id)
	if err != nil {
		writeError(w, http.StatusNotFound, "target survei tidak ditemukan")
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"survey": target})
}

func (s *Server) handleSubmitSurvey(w http.ResponseWriter, r *http.Request) {
	claims := getClaims(r)
	surveyID := chi.URLParam(r, "id")
	var body store.SurveySubmission
	if err := decodeJSON(r, &body); err != nil {
		writeError(w, http.StatusBadRequest, "format jawaban survei tidak valid")
		return
	}
	body.SurveyID = surveyID
	if body.NominationBadge == "" {
		body.NominationBadge = "Paling Konsisten"
	}
	saved, err := s.store.SubmitSurvey(r.Context(), claims.TutorID, body)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	s.sse.Broadcast("survey_submitted", map[string]any{"surveyId": surveyID})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "submission": saved})
}
