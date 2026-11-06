package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"algokanbana/internal/config"
	"algokanbana/internal/seed"
	"algokanbana/internal/store"
	"algokanbana/internal/timeutil"
)

func setupTestServer(t *testing.T) (*httptest.Server, *http.Cookie) {
	t.Helper()
	tmpDir := t.TempDir()
	cfg := config.Config{
		AppMode:              config.ModeHybridSafe,
		Port:                 "0",
		DBPath:               filepath.Join(tmpDir, "test.db"),
		UploadDir:            filepath.Join(tmpDir, "uploads"),
		SessionSecret:        "test-secret-2026",
		SafeReadonlyUpstream: true,
		LMSBaseURL:           "https://lms.algonova.id",
	}

	st, err := store.Open(cfg.DBPath)
	if err != nil {
		t.Fatalf("open store: %v", err)
	}
	t.Cleanup(func() { _ = st.Close() })

	if err := seed.EnsureSeeded(context.Background(), st); err != nil {
		t.Fatalf("seed: %v", err)
	}

	srv := NewServer(cfg, st, nil)
	ts := httptest.NewServer(srv.Handler())
	t.Cleanup(ts.Close)

	// Login as azharfa
	loginPayload := `{"login":"azharfa","password":"test-password"}`
	resp, err := http.Post(ts.URL+"/api/v1/auth/login", "application/json", strings.NewReader(loginPayload))
	if err != nil {
		t.Fatalf("login post: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on login, got %d", resp.StatusCode)
	}
	var sessionCookie *http.Cookie
	for _, c := range resp.Cookies() {
		if c.Name == "ak_session" {
			sessionCookie = c
		}
	}
	if sessionCookie == nil {
		t.Fatalf("expected ak_session cookie to be set")
	}
	return ts, sessionCookie
}

func doAuthedJSON(t *testing.T, ts *httptest.Server, cookie *http.Cookie, method, path string, body any) (*http.Response, map[string]any) {
	t.Helper()
	var rdr *bytes.Reader
	if body != nil {
		b, _ := json.Marshal(body)
		rdr = bytes.NewReader(b)
	} else {
		rdr = bytes.NewReader(nil)
	}
	req, err := http.NewRequest(method, ts.URL+path, rdr)
	if err != nil {
		t.Fatalf("new req: %v", err)
	}
	req.AddCookie(cookie)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("%s %s failed: %v", method, path, err)
	}
	defer resp.Body.Close()
	var parsed map[string]any
	_ = json.NewDecoder(resp.Body).Decode(&parsed)
	return resp, parsed
}

func TestEndToEndTutorAndTeamLeadWorkflows(t *testing.T) {
	ts, cookie := setupTestServer(t)

	// 1. Bootstrap & latency header check
	resp, boot := doAuthedJSON(t, ts, cookie, http.MethodGet, "/api/v1/bootstrap", nil)
	if resp.StatusCode != http.StatusOK || boot["ok"] != true {
		t.Fatalf("bootstrap failed: %v", boot)
	}
	if resp.Header.Get("X-Response-Time-Ms") == "" {
		t.Fatalf("expected X-Response-Time-Ms header")
	}

	// 2. Dashboard 27 active groups + Bug #1 fix (numeric LMS group ID in TL report)
	_, dash := doAuthedJSON(t, ts, cookie, http.MethodGet, "/api/v1/dashboard/groups", nil)
	summary := dash["summary"].(map[string]any)
	if int(summary["activeGroupsCount"].(float64)) != 27 {
		t.Fatalf("expected 27 active groups, got %v", summary["activeGroupsCount"])
	}

	repResp, repData := doAuthedJSON(t, ts, cookie, http.MethodPost, "/api/v1/reports/tl", map[string]any{
		"reportType": "attendance_drop",
		"groupId":    "1710124", // Numeric LMS ID that crashed Zod UUID validation in old NanoKanbana!
		"urgency":    "high",
		"summary":    "Siswa private meminta penyesuaian jam minggu depan",
		"detail":     "Orang tua sudah menginfokan lewat WhatsApp dan siap mengganti sesi di hari Kamis.",
	})
	if repResp.StatusCode != http.StatusOK || repData["ok"] != true {
		t.Fatalf("expected numeric LMS groupId report to succeed, got %d %v", repResp.StatusCode, repData)
	}

	// 3. Regular Feedback save + printable A4 view
	_, fbWeek := doAuthedJSON(t, ts, cookie, http.MethodGet, "/api/v1/feedback/regular", nil)
	milestones := fbWeek["milestones"].([]any)
	if len(milestones) == 0 {
		t.Fatalf("expected milestone lessons in current week")
	}
	firstLesson := milestones[0].(map[string]any)
	firstStudents := firstLesson["students"].([]any)
	firstStu := firstStudents[0].(map[string]any)

	saveResp, saveFB := doAuthedJSON(t, ts, cookie, http.MethodPut, "/api/v1/feedback/regular", map[string]any{
		"weekStart":    fbWeek["weekStart"],
		"lmsLessonId":  firstLesson["lessonId"],
		"lmsStudentId": firstStu["studentId"],
		"courseCode":   firstLesson["courseCode"],
		"monthNo":      2,
		"attendance":   "full",
		"engagement":   "very_active",
		"tasks":        "all_done",
		"custom1":      "Mampu menyelesaikan tantangan fungsi & algoritma dengan mandiri.",
		"stars":        5,
		"projectUrl":   "https://scratch.mit.edu/projects/123456",
	})
	if saveResp.StatusCode != http.StatusOK || saveFB["ok"] != true {
		t.Fatalf("save regular feedback failed: %v", saveFB)
	}
	reportObj := saveFB["report"].(map[string]any)
	reportID := reportObj["reportId"].(string)

	printReq, _ := http.NewRequest(http.MethodGet, ts.URL+"/api/v1/feedback/regular/"+reportID+"/print", nil)
	printReq.AddCookie(cookie)
	printResp, err := http.DefaultClient.Do(printReq)
	if err != nil || printResp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 on printable A4 feedback HTML, got %v %v", printResp, err)
	}
	_ = printResp.Body.Close()

	// 4. Schedule + Private Student Reschedule + Team Lead Approval
	_, sched := doAuthedJSON(t, ts, cookie, http.MethodGet, "/api/v1/schedule", nil)
	lessons := sched["lessons"].([]any)
	var futureLessonID string
	for _, item := range lessons {
		lm := item.(map[string]any)
		st, _ := time.Parse(time.RFC3339, lm["startsAt"].(string))
		if st.After(time.Now().Add(24 * time.Hour)) {
			futureLessonID = lm["lessonId"].(string)
			break
		}
	}
	if futureLessonID == "" {
		t.Fatalf("expected at least one future lesson")
	}

	targetDate := time.Now().In(timeutil.JakartaLoc).AddDate(0, 0, 8)
	// Ensure weekday Mon-Thu 10:00 WIB where no class is scheduled
	for targetDate.Weekday() == time.Saturday || targetDate.Weekday() == time.Sunday {
		targetDate = targetDate.AddDate(0, 0, 1)
	}
	newSlot := timeutil.FormatWIB(time.Date(targetDate.Year(), targetDate.Month(), targetDate.Day(), 10, 0, 0, 0, timeutil.JakartaLoc))
	reschResp, reschData := doAuthedJSON(t, ts, cookie, http.MethodPost, "/api/v1/reschedules", map[string]any{
		"lmsLessonId": futureLessonID,
		"type":        "otr",
		"newStartsAt": newSlot,
		"note":        "Murid private ada ujian sekolah, sepakat pindah ke jam 10:00 WIB.",
	})
	if reschResp.StatusCode != http.StatusOK || reschData["ok"] != true {
		t.Fatalf("reschedule failed: %d %v", reschResp.StatusCode, reschData)
	}
	reqID := reschData["request"].(map[string]any)["requestId"].(string)

	// Team Lead approves the reschedule
	tlResp, tlData := doAuthedJSON(t, ts, cookie, http.MethodPost, "/api/v1/tl/reschedules/"+reqID+"/review", map[string]any{
		"status": "approved",
		"tlNote": "Disetujui, terima kasih konfirmasi cepatnya!",
	})
	if tlResp.StatusCode != http.StatusOK || tlData["ok"] != true {
		t.Fatalf("TL review reschedule failed: %v", tlData)
	}

	// 5. Bug #2 Fix: Support request accepts student name & numeric group ID
	supResp, supData := doAuthedJSON(t, ts, cookie, http.MethodPost, "/api/v1/requests", map[string]any{
		"requestType":       "parent_issue",
		"priority":          "medium",
		"relatedStudentRef": "Nadia Putri",
		"relatedGroupRef":   "1710125",
		"description":       "Orang tua menanyakan invoice paket lanjutan Roblox Studio 2.0.",
	})
	if supResp.StatusCode != http.StatusOK || supData["ok"] != true {
		t.Fatalf("support request with human student name failed: %v", supData)
	}

	// 6. Benchmark endpoint check
	bResp, bData := doAuthedJSON(t, ts, cookie, http.MethodGet, "/api/v1/benchmark", nil)
	if bResp.StatusCode != http.StatusOK || len(bData["comparisons"].([]any)) != 7 {
		t.Fatalf("benchmark endpoint failed: %v", bData)
	}

	// 7. Drag-and-Drop Task Move Check
	_, tasksRes := doAuthedJSON(t, ts, cookie, http.MethodGet, "/api/v1/tasks", nil)
	allTasks := tasksRes["tasks"].([]any)
	if len(allTasks) == 0 {
		t.Fatalf("expected seeded tasks")
	}
	firstTask := allTasks[0].(map[string]any)
	tid := firstTask["taskId"].(string)

	moveResp, moveData := doAuthedJSON(t, ts, cookie, http.MethodPatch, "/api/v1/tasks/"+tid+"/move", map[string]any{
		"targetStatus": "completed",
		"reason":       "Diselesaikan via drag and drop",
	})
	if moveResp.StatusCode != http.StatusOK || moveData["ok"] != true {
		t.Fatalf("move task failed: %v", moveData)
	}

	// 8. Weekly Routines Summary
	rResp, rData := doAuthedJSON(t, ts, cookie, http.MethodGet, "/api/v1/routines/summary", nil)
	if rResp.StatusCode != http.StatusOK || rData["feedbackRoutine"] == nil {
		t.Fatalf("routines summary failed: %v", rData)
	}

	// 9. Surveys Module & 9-Step Submission
	sResp, sData := doAuthedJSON(t, ts, cookie, http.MethodGet, "/api/v1/surveys", nil)
	surveys := sData["surveys"].([]any)
	if sResp.StatusCode != http.StatusOK || len(surveys) == 0 {
		t.Fatalf("expected seeded surveys, got %v", sData)
	}
	firstSurvey := surveys[0].(map[string]any)
	sid := firstSurvey["surveyId"].(string)

	subResp, subData := doAuthedJSON(t, ts, cookie, http.MethodPost, "/api/v1/surveys/"+sid+"/submit", map[string]any{
		"nominationBadge":     "Paling Konsisten",
		"observationNotes":    "Siswa menunjukkan antusiasme tinggi dalam proyek akhir Python.",
		"comprehensionScore":  5,
		"independenceScore":   5,
		"focusScore":          5,
		"stabilityScore":      5,
		"motivationScore":     5,
		"primaryMotivator":    "Eksplorasi pembuatan game mandiri",
		"parentFeedbackNotes": "Orang tua sangat mendukung dan siap perpanjang ke modul lanjutan.",
	})
	if subResp.StatusCode != http.StatusOK || subData["ok"] != true {
		t.Fatalf("survey submission failed: %v", subData)
	}
}
