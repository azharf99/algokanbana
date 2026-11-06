package store

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"algokanbana/internal/curriculum"
	"algokanbana/internal/timeutil"

	"github.com/google/uuid"
)

func (s *Store) GetRegularFeedbackWeek(ctx context.Context, tutorID, weekStart string) (map[string]any, error) {
	var startT time.Time
	var err error
	if strings.TrimSpace(weekStart) == "" {
		startT = timeutil.MondayOfWeekWIB(time.Now())
		weekStart = startT.Format("2006-01-02")
	} else {
		startT, err = timeutil.ParseDateWIB(weekStart)
		if err != nil {
			startT = timeutil.MondayOfWeekWIB(time.Now())
			weekStart = startT.Format("2006-01-02")
		} else {
			startT = timeutil.MondayOfWeekWIB(startT)
			weekStart = startT.Format("2006-01-02")
		}
	}
	endT := startT.AddDate(0, 0, 7)

	// Load existing reports for this tutor & week
	repRows, err := s.db.QueryContext(ctx, `
		SELECT report_id, tutor_id, week_start, lms_lesson_id, lms_student_id, student_name, parent_phone,
		       group_title, lesson_number, course_code, month_no, attendance, engagement, tasks,
		       custom1, custom2, stars, project_url, assembled_text, updated_at
		FROM regular_feedback_reports WHERE tutor_id = ? AND week_start = ?`, tutorID, weekStart)
	if err != nil {
		return nil, err
	}
	defer func() { _ = repRows.Close() }()

	reports := map[string]RegularFeedbackReport{}
	reportsList := []RegularFeedbackReport{}
	for repRows.Next() {
		var rp RegularFeedbackReport
		if err := repRows.Scan(&rp.ReportID, &rp.TutorID, &rp.WeekStart, &rp.LMSLessonID, &rp.LMSStudentID, &rp.StudentName, &rp.ParentPhone,
			&rp.GroupTitle, &rp.LessonNumber, &rp.CourseCode, &rp.MonthNo, &rp.Attendance, &rp.Engagement, &rp.Tasks,
			&rp.Custom1, &rp.Custom2, &rp.Stars, &rp.ProjectURL, &rp.AssembledText, &rp.UpdatedAt); err == nil {
			if c, ok := curriculum.FindCourse(rp.CourseCode); ok {
				rp.CourseName = c.Name
			}
			reports[rp.LMSLessonID+":"+rp.LMSStudentID] = rp
			reportsList = append(reportsList, rp)
		}
	}
	if err := repRows.Err(); err != nil {
		return nil, err
	}

	// Milestone lessons are every 4th lesson (course_lesson_number % 4 == 0) in the selected week
	lRows, err := s.db.QueryContext(ctx, `
		SELECT lesson_id, tutor_id, group_id, group_title, course_code, format, lesson_title,
		       lesson_number, course_lesson_number, starts_at, ends_at, duration_minutes,
		       meeting_url, prep_url, lms_url, is_completed
		FROM lessons
		WHERE tutor_id = ? AND starts_at >= ? AND starts_at < ? AND (course_lesson_number % 4) = 0
		ORDER BY starts_at ASC`, tutorID, timeutil.FormatWIB(startT), timeutil.FormatWIB(endT))
	if err != nil {
		return nil, err
	}
	defer func() { _ = lRows.Close() }()

	milestones := []Lesson{}
	for lRows.Next() {
		var l Lesson
		var done int
		if err := lRows.Scan(&l.LessonID, &l.TutorID, &l.GroupID, &l.GroupTitle, &l.CourseCode, &l.Format, &l.LessonTitle,
			&l.LessonNumber, &l.CourseLessonNumber, &l.StartsAt, &l.EndsAt, &l.DurationMinutes,
			&l.MeetingURL, &l.PrepURL, &l.LMSURL, &done); err == nil {
			l.IsCompleted = done == 1
			milestones = append(milestones, l)
		}
	}
	if err := lRows.Err(); err != nil {
		return nil, err
	}

	totalTargets := 0
	filledTargets := 0
	for i := range milestones {
		func() {
			stRows, err := s.db.QueryContext(ctx, `
				SELECT s.student_id, s.full_name, s.parent_phone
				FROM students s WHERE s.tutor_id = ? AND s.group_id = ? ORDER BY s.full_name ASC`,
				tutorID, milestones[i].GroupID)
			if err != nil {
				return
			}
			defer func() { _ = stRows.Close() }()
			list := []LessonStudent{}
			for stRows.Next() {
				var ls LessonStudent
				if err := stRows.Scan(&ls.StudentID, &ls.StudentName, &ls.ParentPhone); err == nil {
					ls.AttendanceStatus = "attended"
					totalTargets++
					if rp, ok := reports[milestones[i].LessonID+":"+ls.StudentID]; ok {
						ls.ReportFilled = true
						ls.ReportID = rp.ReportID
						filledTargets++
					}
					list = append(list, ls)
				}
			}
			_ = stRows.Err()
			milestones[i].Students = list
		}()
	}

	deadline := time.Date(endT.Year(), endT.Month(), endT.Day()-1, 23, 59, 0, 0, timeutil.JakartaLoc)
	return map[string]any{
		"weekStart":     weekStart,
		"prevWeek":      startT.AddDate(0, 0, -7).Format("2006-01-02"),
		"nextWeek":      startT.AddDate(0, 0, 7).Format("2006-01-02"),
		"deadlineAt":    timeutil.FormatWIB(deadline),
		"totalTargets":  totalTargets,
		"filledTargets": filledTargets,
		"milestones":    milestones,
		"reports":       reportsList,
		"courses":       curriculum.Courses,
	}, nil
}

func (s *Store) SaveRegularFeedbackReport(ctx context.Context, locale string, rp RegularFeedbackReport) (RegularFeedbackReport, error) {
	if rp.Stars < 1 || rp.Stars > 5 {
		rp.Stars = 5
	}
	if rp.MonthNo < 1 {
		rp.MonthNo = 1
	}
	if rp.StudentName == "" || rp.ParentPhone == "" {
		_ = s.db.QueryRowContext(ctx, `SELECT full_name, parent_phone, group_title FROM students WHERE student_id = ?`, rp.LMSStudentID).Scan(&rp.StudentName, &rp.ParentPhone, &rp.GroupTitle)
	}
	if rp.LessonNumber == 0 {
		_ = s.db.QueryRowContext(ctx, `SELECT course_lesson_number FROM lessons WHERE lesson_id = ?`, rp.LMSLessonID).Scan(&rp.LessonNumber)
	}
	if strings.TrimSpace(rp.AssembledText) == "" {
		rp.AssembledText = curriculum.AssembleRegularFeedback(
			locale, rp.StudentName, rp.CourseCode, rp.MonthNo,
			rp.Attendance, rp.Engagement, rp.Tasks,
			rp.Custom1, rp.Custom2, rp.ProjectURL, rp.Stars,
		)
	}
	if c, ok := curriculum.FindCourse(rp.CourseCode); ok {
		rp.CourseName = c.Name
	}

	var existingID string
	_ = s.db.QueryRowContext(ctx, `SELECT report_id FROM regular_feedback_reports WHERE tutor_id = ? AND lms_lesson_id = ? AND lms_student_id = ?`,
		rp.TutorID, rp.LMSLessonID, rp.LMSStudentID).Scan(&existingID)
	if existingID != "" {
		rp.ReportID = existingID
	} else {
		rp.ReportID = uuid.NewString()
	}
	rp.UpdatedAt = timeutil.FormatWIB(time.Now())

	_, err := s.db.ExecContext(ctx, `
		INSERT INTO regular_feedback_reports (
			report_id, tutor_id, week_start, lms_lesson_id, lms_student_id, student_name, parent_phone,
			group_title, lesson_number, course_code, month_no, attendance, engagement, tasks,
			custom1, custom2, stars, project_url, assembled_text, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(tutor_id, lms_lesson_id, lms_student_id) DO UPDATE SET
			course_code=excluded.course_code,
			month_no=excluded.month_no,
			attendance=excluded.attendance,
			engagement=excluded.engagement,
			tasks=excluded.tasks,
			custom1=excluded.custom1,
			custom2=excluded.custom2,
			stars=excluded.stars,
			project_url=excluded.project_url,
			assembled_text=excluded.assembled_text,
			updated_at=excluded.updated_at`,
		rp.ReportID, rp.TutorID, rp.WeekStart, rp.LMSLessonID, rp.LMSStudentID, rp.StudentName, rp.ParentPhone,
		rp.GroupTitle, rp.LessonNumber, rp.CourseCode, rp.MonthNo, rp.Attendance, rp.Engagement, rp.Tasks,
		rp.Custom1, rp.Custom2, rp.Stars, rp.ProjectURL, rp.AssembledText, rp.UpdatedAt)
	return rp, err
}

func (s *Store) GetRegularFeedbackByID(ctx context.Context, tutorID, reportID string) (RegularFeedbackReport, error) {
	var rp RegularFeedbackReport
	err := s.db.QueryRowContext(ctx, `
		SELECT report_id, tutor_id, week_start, lms_lesson_id, lms_student_id, student_name, parent_phone,
		       group_title, lesson_number, course_code, month_no, attendance, engagement, tasks,
		       custom1, custom2, stars, project_url, assembled_text, updated_at
		FROM regular_feedback_reports WHERE tutor_id = ? AND report_id = ?`, tutorID, reportID).Scan(
		&rp.ReportID, &rp.TutorID, &rp.WeekStart, &rp.LMSLessonID, &rp.LMSStudentID, &rp.StudentName, &rp.ParentPhone,
		&rp.GroupTitle, &rp.LessonNumber, &rp.CourseCode, &rp.MonthNo, &rp.Attendance, &rp.Engagement, &rp.Tasks,
		&rp.Custom1, &rp.Custom2, &rp.Stars, &rp.ProjectURL, &rp.AssembledText, &rp.UpdatedAt)
	if err == nil {
		if c, ok := curriculum.FindCourse(rp.CourseCode); ok {
			rp.CourseName = c.Name
		}
	}
	return rp, err
}

func (s *Store) ListMTFeedback(ctx context.Context, tutorID string) ([]MTFeedbackEntry, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT entry_id, tutor_id, week_start, lms_lesson_id, lms_student_id, student_name, lesson_number,
		       lesson_topic, recording_url, understanding, independence, accuracy, engagement,
		       what_worked, development_area, cause, next_steps, tutor_comment, is_on_time, source, created_at
		FROM mt_feedback_entries WHERE tutor_id = ? ORDER BY created_at DESC`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []MTFeedbackEntry{}
	for rows.Next() {
		var e MTFeedbackEntry
		var onTime int
		if err := rows.Scan(&e.EntryID, &e.TutorID, &e.WeekStart, &e.LMSLessonID, &e.LMSStudentID, &e.StudentName, &e.LessonNumber,
			&e.LessonTopic, &e.RecordingURL, &e.Understanding, &e.Independence, &e.Accuracy, &e.Engagement,
			&e.WhatWorked, &e.DevelopmentArea, &e.Cause, &e.NextSteps, &e.TutorComment, &onTime, &e.Source, &e.CreatedAt); err == nil {
			e.IsOnTime = onTime == 1
			out = append(out, e)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) CreateMTFeedback(ctx context.Context, e MTFeedbackEntry) (MTFeedbackEntry, error) {
	e.EntryID = uuid.NewString()
	e.IsOnTime = true
	e.Source = "portal"
	e.CreatedAt = timeutil.FormatWIB(time.Now())
	if e.StudentName == "" {
		_ = s.db.QueryRowContext(ctx, `SELECT full_name FROM students WHERE student_id = ?`, e.LMSStudentID).Scan(&e.StudentName)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO mt_feedback_entries (
			entry_id, tutor_id, week_start, lms_lesson_id, lms_student_id, student_name, lesson_number,
			lesson_topic, recording_url, understanding, independence, accuracy, engagement,
			what_worked, development_area, cause, next_steps, tutor_comment, is_on_time, source, created_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		e.EntryID, e.TutorID, e.WeekStart, e.LMSLessonID, e.LMSStudentID, e.StudentName, e.LessonNumber,
		e.LessonTopic, e.RecordingURL, e.Understanding, e.Independence, e.Accuracy, e.Engagement,
		e.WhatWorked, e.DevelopmentArea, e.Cause, e.NextSteps, e.TutorComment, 1, e.Source, e.CreatedAt)
	return e, err
}

func (s *Store) GetScheduleBundle(ctx context.Context, tutorID, fromDate, toDate string) (map[string]any, error) {
	now := time.Now()
	weekMon := timeutil.MondayOfWeekWIB(now)
	fromT := weekMon.AddDate(0, 0, -7)
	toT := weekMon.AddDate(0, 0, 35)
	if t, err := timeutil.ParseDateWIB(fromDate); err == nil {
		fromT = t
	}
	if t, err := timeutil.ParseDateWIB(toDate); err == nil {
		toT = t.AddDate(0, 0, 1)
	}

	lRows, err := s.db.QueryContext(ctx, `
		SELECT lesson_id, tutor_id, group_id, group_title, course_code, format, lesson_title,
		       lesson_number, course_lesson_number, starts_at, ends_at, duration_minutes,
		       meeting_url, prep_url, lms_url, is_completed
		FROM lessons
		WHERE tutor_id = ? AND starts_at >= ? AND starts_at <= ?
		ORDER BY starts_at ASC`, tutorID, timeutil.FormatWIB(fromT), timeutil.FormatWIB(toT))
	if err != nil {
		return nil, err
	}
	defer func() { _ = lRows.Close() }()
	lessons := []Lesson{}
	totalMinutes4W := 0
	w4End := weekMon.AddDate(0, 0, 28)
	for lRows.Next() {
		var l Lesson
		var done int
		if err := lRows.Scan(&l.LessonID, &l.TutorID, &l.GroupID, &l.GroupTitle, &l.CourseCode, &l.Format, &l.LessonTitle,
			&l.LessonNumber, &l.CourseLessonNumber, &l.StartsAt, &l.EndsAt, &l.DurationMinutes,
			&l.MeetingURL, &l.PrepURL, &l.LMSURL, &done); err == nil {
			l.IsCompleted = done == 1
			lessons = append(lessons, l)
			if st, err := time.Parse(time.RFC3339, l.StartsAt); err == nil && !st.Before(weekMon) && st.Before(w4End) {
				totalMinutes4W += l.DurationMinutes
			}
		}
	}
	if err := lRows.Err(); err != nil {
		return nil, err
	}

	aRows, err := s.db.QueryContext(ctx, `SELECT weekday, start_min, status, sheet_cell FROM availability_slots WHERE tutor_id = ? ORDER BY weekday ASC, start_min ASC`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = aRows.Close() }()
	slots := []AvailabilitySlot{}
	for aRows.Next() {
		var sl AvailabilitySlot
		if err := aRows.Scan(&sl.Weekday, &sl.StartMin, &sl.Status, &sl.SheetCell); err == nil {
			slots = append(slots, sl)
		}
	}
	if err := aRows.Err(); err != nil {
		return nil, err
	}

	reschedules, _ := s.ListRescheduleRequests(ctx, tutorID)
	events, _ := s.ListPortalEvents(ctx)

	capacityHours := 160
	busyHours := totalMinutes4W / 60
	freeHours := max(capacityHours-busyHours, 0)
	loadPct := 0
	if capacityHours > 0 {
		loadPct = (busyHours * 100) / capacityHours
	}

	return map[string]any{
		"weekStart": weekMon.Format("2006-01-02"),
		"workload": map[string]any{
			"busyHours":     busyHours,
			"freeHours":     freeHours,
			"loadPct":       loadPct,
			"freePct":       100 - loadPct,
			"capacityHours": capacityHours,
		},
		"lessons":            lessons,
		"availabilitySlots":  slots,
		"rescheduleRequests": reschedules,
		"events":             events,
	}, nil
}

type SlotCoord struct {
	Weekday  int `json:"weekday"`
	StartMin int `json:"startMin"`
}

func (s *Store) UpdateAvailabilitySlots(ctx context.Context, tutorID string, coords []SlotCoord, openStatus string) ([]map[string]any, error) {
	target := "not_available"
	if openStatus == "open" || openStatus == "available" {
		target = "available"
	}
	nowStr := timeutil.FormatWIB(time.Now())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	results := make([]map[string]any, 0, len(coords))
	for _, c := range coords {
		var curStatus string
		err := tx.QueryRowContext(ctx, `SELECT status FROM availability_slots WHERE tutor_id = ? AND weekday = ? AND start_min = ?`, tutorID, c.Weekday, c.StartMin).Scan(&curStatus)
		if err != nil {
			_, _ = tx.ExecContext(ctx, `INSERT INTO availability_slots (tutor_id, weekday, start_min, status, sheet_cell, updated_at) VALUES (?,?,?,?,?,?)`,
				tutorID, c.Weekday, c.StartMin, target, "AS", nowStr)
			results = append(results, map[string]any{"weekday": c.Weekday, "startMin": c.StartMin, "outcome": "written", "status": target})
			continue
		}
		if curStatus == "booked" || curStatus == "booked_for_group" {
			results = append(results, map[string]any{"weekday": c.Weekday, "startMin": c.StartMin, "outcome": "booked", "status": curStatus})
			continue
		}
		_, _ = tx.ExecContext(ctx, `UPDATE availability_slots SET status = ?, updated_at = ? WHERE tutor_id = ? AND weekday = ? AND start_min = ?`,
			target, nowStr, tutorID, c.Weekday, c.StartMin)
		results = append(results, map[string]any{"weekday": c.Weekday, "startMin": c.StartMin, "outcome": "written", "status": target})
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return results, nil
}

func (s *Store) ListRescheduleRequests(ctx context.Context, tutorID string) ([]RescheduleRequest, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT request_id, tutor_id, tutor_name, lms_lesson_id, lms_group_id, group_name, group_format,
		       lesson_number, req_type, original_starts_at, new_starts_at, note, proof_urls_json,
		       status, tl_note, created_at, reviewed_at
		FROM reschedule_requests WHERE tutor_id = ? ORDER BY created_at DESC`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []RescheduleRequest{}
	for rows.Next() {
		var r RescheduleRequest
		var proofs string
		if err := rows.Scan(&r.RequestID, &r.TutorID, &r.TutorName, &r.LMSLessonID, &r.LMSGroupID, &r.GroupName, &r.GroupFormat,
			&r.LessonNumber, &r.ReqType, &r.OriginalStartsAt, &r.NewStartsAt, &r.Note, &proofs,
			&r.Status, &r.TLNote, &r.CreatedAt, &r.ReviewedAt); err == nil {
			r.ProofURLs = parseJSONSlice(proofs)
			out = append(out, r)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) CreateRescheduleRequest(ctx context.Context, req RescheduleRequest, now time.Time) (RescheduleRequest, error) {
	if req.ReqType != "otr" && req.ReqType != "permanent" {
		req.ReqType = "otr"
	}

	var origStartsStr string
	var durMin int
	err := s.db.QueryRowContext(ctx, `SELECT group_id, group_title, format, lesson_number, starts_at, duration_minutes FROM lessons WHERE lesson_id = ?`, req.LMSLessonID).Scan(
		&req.LMSGroupID, &req.GroupName, &req.GroupFormat, &req.LessonNumber, &origStartsStr, &durMin)
	if err == nil {
		req.OriginalStartsAt = origStartsStr
	} else if req.OriginalStartsAt == "" {
		return RescheduleRequest{}, errors.New("sesi kelas tidak ditemukan")
	}
	if durMin <= 0 {
		durMin = 90
	}

	origT, err := time.Parse(time.RFC3339, req.OriginalStartsAt)
	if err == nil && !timeutil.CanRescheduleLesson(origT, now) {
		return RescheduleRequest{}, errors.New("batas waktu reschedule pekan lalu (Senin 23:59 WIB) telah lewat")
	}

	newT, err := timeutil.ParseLMSMoscowTime(req.NewStartsAt)
	if err != nil {
		return RescheduleRequest{}, fmt.Errorf("format waktu baru tidak valid: %w", err)
	}
	req.NewStartsAt = timeutil.FormatWIB(newT)
	newEndT := newT.Add(time.Duration(durMin) * time.Minute)

	if req.ReqType == "permanent" && origT.Before(now) {
		return RescheduleRequest{}, errors.New("perubahan jadwal permanen hanya dapat diajukan untuk sesi kelas mendatang")
	}

	// Check pending duplicate
	var pendingCnt int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM reschedule_requests WHERE tutor_id = ? AND lms_lesson_id = ? AND status = 'pending'`, req.TutorID, req.LMSLessonID).Scan(&pendingCnt)
	if pendingCnt > 0 {
		return RescheduleRequest{}, errors.New("sudah ada permintaan reschedule aktif untuk sesi pelajaran ini")
	}

	// Check overlap conflict against existing lessons
	var conflictGroup string
	_ = s.db.QueryRowContext(ctx, `
		SELECT group_title FROM lessons
		WHERE tutor_id = ? AND lesson_id != ? AND starts_at < ? AND ends_at > ? LIMIT 1`,
		req.TutorID, req.LMSLessonID, timeutil.FormatWIB(newEndT), req.NewStartsAt).Scan(&conflictGroup)
	if conflictGroup != "" {
		return RescheduleRequest{}, fmt.Errorf("bentrok dengan jadwal kelas %s pada jam tersebut", conflictGroup)
	}

	req.RequestID = uuid.NewString()
	req.Status = "pending"
	req.CreatedAt = timeutil.FormatWIB(now)
	if req.ProofURLs == nil {
		req.ProofURLs = []string{}
	}

	_, err = s.db.ExecContext(ctx, `
		INSERT INTO reschedule_requests (
			request_id, tutor_id, tutor_name, lms_lesson_id, lms_group_id, group_name, group_format,
			lesson_number, req_type, original_starts_at, new_starts_at, note, proof_urls_json,
			status, tl_note, created_at, reviewed_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		req.RequestID, req.TutorID, req.TutorName, req.LMSLessonID, req.LMSGroupID, req.GroupName, req.GroupFormat,
		req.LessonNumber, req.ReqType, req.OriginalStartsAt, req.NewStartsAt, req.Note, toJSON(req.ProofURLs),
		req.Status, "", req.CreatedAt, "")
	return req, err
}

func (s *Store) UpdateRescheduleStatus(ctx context.Context, requestID, status, tlNote string) error {
	nowStr := timeutil.FormatWIB(time.Now())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	var lessonID, newStarts string
	if err := tx.QueryRowContext(ctx, `SELECT lms_lesson_id, new_starts_at FROM reschedule_requests WHERE request_id = ?`, requestID).Scan(&lessonID, &newStarts); err != nil {
		return err
	}
	if _, err := tx.ExecContext(ctx, `UPDATE reschedule_requests SET status = ?, tl_note = ?, reviewed_at = ? WHERE request_id = ?`, status, strings.TrimSpace(tlNote), nowStr, requestID); err != nil {
		return err
	}
	if status == "approved" {
		var dur int
		_ = tx.QueryRowContext(ctx, `SELECT duration_minutes FROM lessons WHERE lesson_id = ?`, lessonID).Scan(&dur)
		if dur <= 0 {
			dur = 90
		}
		if st, err := time.Parse(time.RFC3339, newStarts); err == nil {
			endStr := timeutil.FormatWIB(st.Add(time.Duration(dur) * time.Minute))
			_, _ = tx.ExecContext(ctx, `UPDATE lessons SET starts_at = ?, ends_at = ? WHERE lesson_id = ?`, newStarts, endStr, lessonID)
		}
	}
	return tx.Commit()
}

func (s *Store) CreateWorkloadRequest(ctx context.Context, w WorkloadRequest) (WorkloadRequest, error) {
	w.RequestID = uuid.NewString()
	w.Status = "pending"
	w.CreatedAt = timeutil.FormatWIB(time.Now())
	_, err := s.db.ExecContext(ctx, `INSERT INTO workload_requests (request_id, tutor_id, tutor_name, req_type, reason, status, created_at) VALUES (?,?,?,?,?,?,?)`,
		w.RequestID, w.TutorID, w.TutorName, w.ReqType, w.Reason, w.Status, w.CreatedAt)
	return w, err
}

func (s *Store) GetAssessmentBundle(ctx context.Context, tutorID string) (map[string]any, error) {
	var cycle AssessmentCycle
	var qJSON string
	var isOpen int
	err := s.db.QueryRowContext(ctx, `SELECT cycle_code, title, channel, deadline_at, is_open, questions_json FROM assessment_cycles ORDER BY is_open DESC LIMIT 1`).Scan(
		&cycle.CycleCode, &cycle.Title, &cycle.Channel, &cycle.DeadlineAt, &isOpen, &qJSON)
	var activeCycle *AssessmentCycle
	if err == nil {
		cycle.IsOpen = isOpen == 1
		_ = json.Unmarshal([]byte(qJSON), &cycle.Questions)
		var sub AssessmentSubmission
		var progJSON, ansJSON string
		if errSub := s.db.QueryRowContext(ctx, `SELECT submission_id, cycle_code, programs_json, answers_json, score_pct, submitted_at FROM assessment_submissions WHERE tutor_id = ? AND cycle_code = ?`, tutorID, cycle.CycleCode).Scan(
			&sub.SubmissionID, &sub.CycleCode, &progJSON, &ansJSON, &sub.ScorePct, &sub.SubmittedAt); errSub == nil {
			sub.Programs = parseJSONSlice(progJSON)
			_ = json.Unmarshal([]byte(ansJSON), &sub.Answers)
			cycle.Submission = &sub
		}
		activeCycle = &cycle
	}

	hRows, err := s.db.QueryContext(ctx, `SELECT id, cycle_code, cycle_label, month_key, program, score_pct, max_pct, status, graded_at FROM assessment_history WHERE tutor_id = ? ORDER BY id DESC`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = hRows.Close() }()
	history := []AssessmentHistoryItem{}
	for hRows.Next() {
		var h AssessmentHistoryItem
		if err := hRows.Scan(&h.ID, &h.CycleCode, &h.CycleLabel, &h.MonthKey, &h.Program, &h.ScorePct, &h.MaxPct, &h.Status, &h.GradedAt); err == nil {
			history = append(history, h)
		}
	}
	if err := hRows.Err(); err != nil {
		return nil, err
	}

	return map[string]any{
		"activeCycle": activeCycle,
		"history":     history,
	}, nil
}

func (s *Store) SubmitAssessment(ctx context.Context, tutorID, cycleCode string, programs []string, answers map[string]any) (AssessmentSubmission, error) {
	var exists int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM assessment_submissions WHERE tutor_id = ? AND cycle_code = ?`, tutorID, cycleCode).Scan(&exists)
	if exists > 0 {
		return AssessmentSubmission{}, errors.New("assessment untuk siklus ini sudah dikirim sebelumnya")
	}
	if len(answers) == 0 {
		return AssessmentSubmission{}, errors.New("jawaban assessment tidak boleh kosong")
	}
	if len(programs) == 0 {
		programs = []string{"coding"}
	}
	sub := AssessmentSubmission{
		SubmissionID: uuid.NewString(),
		CycleCode:    cycleCode,
		Programs:     programs,
		Answers:      answers,
		ScorePct:     100,
		SubmittedAt:  timeutil.FormatWIB(time.Now()),
	}
	ansBytes, _ := json.Marshal(answers)
	_, err := s.db.ExecContext(ctx, `INSERT INTO assessment_submissions (submission_id, tutor_id, cycle_code, programs_json, answers_json, score_pct, submitted_at) VALUES (?,?,?,?,?,?,?)`,
		sub.SubmissionID, tutorID, sub.CycleCode, toJSON(sub.Programs), string(ansBytes), sub.ScorePct, sub.SubmittedAt)
	if err != nil {
		return AssessmentSubmission{}, err
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO assessment_history (tutor_id, cycle_code, cycle_label, month_key, program, score_pct, max_pct, status, graded_at) VALUES (?,?,?,?,?,?,?,?,?)`,
		tutorID, cycleCode, "Assessment "+cycleCode, time.Now().Format("2006-01"), "Coding & Design", 100, 100, "passed", sub.SubmittedAt)
	return sub, nil
}

func (s *Store) ListKBArticles(ctx context.Context, category, query string) ([]KBArticle, error) {
	q := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	cat := strings.TrimSpace(category)
	rows, err := s.db.QueryContext(ctx, `
		SELECT article_id, category, locale, title, summary, content_md, helpful_count
		FROM kb_articles
		WHERE (? = '' OR category = ?) AND (LOWER(title) LIKE ? OR LOWER(summary) LIKE ? OR LOWER(content_md) LIKE ?)
		ORDER BY category ASC, title ASC`, cat, cat, q, q, q)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []KBArticle{}
	for rows.Next() {
		var a KBArticle
		if err := rows.Scan(&a.ArticleID, &a.Category, &a.Locale, &a.Title, &a.Summary, &a.ContentMD, &a.HelpfulCount); err == nil {
			out = append(out, a)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListThreads(ctx context.Context, tutorID string) ([]Thread, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT t.thread_id, t.tutor_id, t.subject, t.request_type, t.target_team, t.priority, t.origin,
		       t.related_group_ref, t.related_student_ref, t.unread_tutor, t.unread_tl, t.status, t.updated_at, t.created_at,
		       COALESCE((SELECT m.body FROM messages m WHERE m.thread_id = t.thread_id ORDER BY m.created_at DESC LIMIT 1), '')
		FROM threads t WHERE t.tutor_id = ? ORDER BY t.updated_at DESC`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Thread{}
	for rows.Next() {
		var th Thread
		if err := rows.Scan(&th.ThreadID, &th.TutorID, &th.Subject, &th.RequestType, &th.TargetTeam, &th.Priority, &th.Origin,
			&th.RelatedGroupRef, &th.RelatedStudentRef, &th.UnreadTutor, &th.UnreadTL, &th.Status, &th.UpdatedAt, &th.CreatedAt, &th.LastMessage); err == nil {
			out = append(out, th)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) GetThread(ctx context.Context, threadID string) (Thread, error) {
	var th Thread
	err := s.db.QueryRowContext(ctx, `
		SELECT thread_id, tutor_id, subject, request_type, target_team, priority, origin,
		       related_group_ref, related_student_ref, unread_tutor, unread_tl, status, updated_at, created_at
		FROM threads WHERE thread_id = ?`, threadID).Scan(
		&th.ThreadID, &th.TutorID, &th.Subject, &th.RequestType, &th.TargetTeam, &th.Priority, &th.Origin,
		&th.RelatedGroupRef, &th.RelatedStudentRef, &th.UnreadTutor, &th.UnreadTL, &th.Status, &th.UpdatedAt, &th.CreatedAt)
	if err != nil {
		return Thread{}, err
	}

	mRows, err := s.db.QueryContext(ctx, `
		SELECT message_id, thread_id, sender_role, sender_name, kind, body, attachment_url, created_at
		FROM messages WHERE thread_id = ? ORDER BY created_at ASC`, threadID)
	if err == nil {
		defer func() { _ = mRows.Close() }()
		msgs := []Message{}
		for mRows.Next() {
			var m Message
			if err := mRows.Scan(&m.MessageID, &m.ThreadID, &m.SenderRole, &m.SenderName, &m.Kind, &m.Body, &m.AttachmentURL, &m.CreatedAt); err == nil {
				msgs = append(msgs, m)
			}
		}
		_ = mRows.Err()
		th.Messages = msgs
	}
	return th, nil
}

func routeTargetTeam(requestType string) string {
	switch requestType {
	case "customer_support", "student_transfer", "parent_issue":
		return "cs"
	case "icm":
		return "ism"
	case "technical_issue":
		return "tech"
	default:
		return "tl"
	}
}

// CreateSupportRequestThread fixes NanoKanbana Bug #2 by accepting human names, numeric LMS IDs, or UUIDs.
func (s *Store) CreateSupportRequestThread(ctx context.Context, tutorID, senderName, requestType, priority, relatedStudentRef, relatedGroupRef, description, attachmentURL, origin string) (Thread, error) {
	nowStr := timeutil.FormatWIB(time.Now())
	if origin == "" {
		origin = "direct"
	}
	if priority == "" {
		priority = "medium"
	}
	th := Thread{
		ThreadID:          uuid.NewString(),
		TutorID:           tutorID,
		Subject:           fmt.Sprintf("[%s] %s", strings.ToUpper(strings.ReplaceAll(requestType, "_", " ")), truncateStr(description, 56)),
		RequestType:       requestType,
		TargetTeam:        routeTargetTeam(requestType),
		Priority:          priority,
		Origin:            origin,
		RelatedGroupRef:   strings.TrimSpace(relatedGroupRef),
		RelatedStudentRef: strings.TrimSpace(relatedStudentRef),
		UnreadTutor:       0,
		UnreadTL:          1,
		Status:            "open",
		LastMessage:       description,
		UpdatedAt:         nowStr,
		CreatedAt:         nowStr,
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO threads (thread_id, tutor_id, subject, request_type, target_team, priority, origin, related_group_ref, related_student_ref, unread_tutor, unread_tl, status, updated_at, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		th.ThreadID, th.TutorID, th.Subject, th.RequestType, th.TargetTeam, th.Priority, th.Origin, th.RelatedGroupRef, th.RelatedStudentRef, 0, 1, "open", nowStr, nowStr)
	if err != nil {
		return Thread{}, err
	}
	msg := Message{
		MessageID:     uuid.NewString(),
		ThreadID:      th.ThreadID,
		SenderRole:    "tutor",
		SenderName:    senderName,
		Kind:          "request",
		Body:          strings.TrimSpace(description),
		AttachmentURL: strings.TrimSpace(attachmentURL),
		CreatedAt:     nowStr,
	}
	_, _ = s.db.ExecContext(ctx, `INSERT INTO messages (message_id, thread_id, sender_role, sender_name, kind, body, attachment_url, created_at) VALUES (?,?,?,?,?,?,?,?)`,
		msg.MessageID, msg.ThreadID, msg.SenderRole, msg.SenderName, msg.Kind, msg.Body, msg.AttachmentURL, msg.CreatedAt)
	th.Messages = []Message{msg}
	return th, nil
}

func (s *Store) ReplyThread(ctx context.Context, threadID, senderRole, senderName, kind, body, attachmentURL string) (Message, error) {
	nowStr := timeutil.FormatWIB(time.Now())
	if kind == "" {
		kind = "text"
	}
	m := Message{
		MessageID:     uuid.NewString(),
		ThreadID:      threadID,
		SenderRole:    senderRole,
		SenderName:    senderName,
		Kind:          kind,
		Body:          strings.TrimSpace(body),
		AttachmentURL: strings.TrimSpace(attachmentURL),
		CreatedAt:     nowStr,
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO messages (message_id, thread_id, sender_role, sender_name, kind, body, attachment_url, created_at) VALUES (?,?,?,?,?,?,?,?)`,
		m.MessageID, m.ThreadID, m.SenderRole, m.SenderName, m.Kind, m.Body, m.AttachmentURL, m.CreatedAt)
	if err != nil {
		return Message{}, err
	}
	if senderRole == "team_lead" {
		_, _ = s.db.ExecContext(ctx, `UPDATE threads SET updated_at = ?, unread_tutor = unread_tutor + 1, unread_tl = 0 WHERE thread_id = ?`, nowStr, threadID)
	} else {
		_, _ = s.db.ExecContext(ctx, `UPDATE threads SET updated_at = ?, unread_tl = unread_tl + 1, unread_tutor = 0 WHERE thread_id = ?`, nowStr, threadID)
	}
	return m, nil
}

func (s *Store) MarkThreadRead(ctx context.Context, threadID, role string) error {
	col := "unread_tutor"
	if role == "team_lead" {
		col = "unread_tl"
	}
	_, err := s.db.ExecContext(ctx, `UPDATE threads SET `+col+` = 0 WHERE thread_id = ?`, threadID)
	return err
}

func (s *Store) EscalateThread(ctx context.Context, threadID, senderName, severity, targetTeam, reason string) error {
	nowStr := timeutil.FormatWIB(time.Now())
	_, err := s.db.ExecContext(ctx, `UPDATE threads SET priority = ?, target_team = ?, origin = 'escalated', updated_at = ?, unread_tl = unread_tl + 1 WHERE thread_id = ?`,
		severity, targetTeam, nowStr, threadID)
	if err != nil {
		return err
	}
	_, err = s.ReplyThread(ctx, threadID, "tutor", senderName, "escalation", fmt.Sprintf("[ESKALASI -> %s | %s] %s", strings.ToUpper(targetTeam), strings.ToUpper(severity), reason), "")
	return err
}

// GetTLDemoQueue returns live incoming items for the Team Lead Demo view (for pitching to Algonova leadership).
func (s *Store) GetTLDemoQueue(ctx context.Context, tutorID string) (map[string]any, error) {
	reports, _ := s.ListTLReports(ctx, tutorID)
	reschedules, _ := s.ListRescheduleRequests(ctx, tutorID)
	urgentSubs, _ := s.ListUrgentSubRequests(ctx, tutorID)
	threads, _ := s.ListThreads(ctx, tutorID)

	escalatedTasks := []Task{}
	allTasks, _ := s.ListTasks(ctx, tutorID)
	for _, tk := range allTasks {
		if tk.Status == "escalated" {
			escalatedTasks = append(escalatedTasks, tk)
		}
	}
	tlName := "Team Lead Algonova"
	var customTL string
	if err := s.db.QueryRowContext(ctx, `SELECT tl_name FROM tutors WHERE tutor_id = ?`, tutorID).Scan(&customTL); err == nil && customTL != "" {
		tlName = customTL
	}
	return map[string]any{
		"teamLeadName":       tlName,
		"tlReports":          reports,
		"rescheduleRequests": reschedules,
		"urgentSubRequests":  urgentSubs,
		"escalatedTasks":     escalatedTasks,
		"threads":            threads,
	}, nil
}

func (s *Store) GetRoutinesSummary(ctx context.Context, tutorID string) (RoutinesSummary, error) {
	var sum RoutinesSummary

	// 1. Feedback Routine
	weekStart := timeutil.MondayOfWeekWIB(time.Now()).Format("2006-01-02")
	fbWeek, _ := s.GetRegularFeedbackWeek(ctx, tutorID, weekStart)
	if fbWeek != nil {
		total, _ := fbWeek["totalTargets"].(int)
		filled, _ := fbWeek["filledTargets"].(int)
		pct := 0
		if total > 0 {
			pct = (filled * 100) / total
		}
		sum.FeedbackRoutine.Filled = filled
		sum.FeedbackRoutine.Total = total
		sum.FeedbackRoutine.Pct = pct
		if total > 0 {
			sum.FeedbackRoutine.Label = fmt.Sprintf("%d dari %d • %d%%", filled, total, pct)
		} else {
			sum.FeedbackRoutine.Label = "Belum ada target feedback pekan ini"
		}
	}

	// 2. Graduation Routine
	var gradCount int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM students WHERE tutor_id = ? AND lessons_remaining <= 3`, tutorID).Scan(&gradCount)
	sum.GraduationRoutine.Pending = gradCount
	sum.GraduationRoutine.Label = fmt.Sprintf("%d siswa perlu catatan", gradCount)

	// 3. Attendance Routine
	var avgAtt sql.NullInt64
	var lessonsAtt, lessonsTot sql.NullInt64
	_ = s.db.QueryRowContext(ctx, `SELECT AVG(attendance_pct), SUM(lessons_attended), SUM(lessons_total) FROM students WHERE tutor_id = ?`, tutorID).Scan(&avgAtt, &lessonsAtt, &lessonsTot)
	if avgAtt.Valid {
		sum.AttendanceRoutine.Pct = int(avgAtt.Int64)
	}
	if lessonsAtt.Valid {
		sum.AttendanceRoutine.Done = int(lessonsAtt.Int64)
	}
	if lessonsTot.Valid {
		sum.AttendanceRoutine.Total = int(lessonsTot.Int64)
	}
	if sum.AttendanceRoutine.Total > 0 {
		sum.AttendanceRoutine.Label = fmt.Sprintf("%d%% • %d/%d hadir", sum.AttendanceRoutine.Pct, sum.AttendanceRoutine.Done, sum.AttendanceRoutine.Total)
	} else {
		sum.AttendanceRoutine.Label = "Belum ada data kehadiran siswa"
	}

	// 4. Assessment Routine
	var cycleCode string
	var isOpen int
	_ = s.db.QueryRowContext(ctx, `SELECT cycle_code, is_open FROM assessment_cycles ORDER BY is_open DESC LIMIT 1`).Scan(&cycleCode, &isOpen)
	var subCount int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM assessment_submissions WHERE tutor_id = ? AND cycle_code = ?`, tutorID, cycleCode).Scan(&subCount)
	sum.AssessmentRoutine.CycleCode = cycleCode
	if subCount > 0 {
		sum.AssessmentRoutine.Status = "completed"
		sum.AssessmentRoutine.Label = "100% Selesai"
	} else {
		sum.AssessmentRoutine.Status = "pending"
		sum.AssessmentRoutine.Label = "Belum dimulai"
	}

	// 5. Survey Routine
	var survPending int
	var nearestDeadline string
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1), COALESCE(MIN(deadline_at), '') FROM survey_targets WHERE tutor_id = ? AND status = 'pending'`, tutorID).Scan(&survPending, &nearestDeadline)
	sum.SurveyRoutine.PendingCount = survPending
	sum.SurveyRoutine.DeadlineAt = nearestDeadline
	if survPending > 0 {
		sum.SurveyRoutine.Label = fmt.Sprintf("%d survei perlu diisi", survPending)
	} else {
		sum.SurveyRoutine.Label = "Semua survei tuntas"
	}

	return sum, nil
}

func (s *Store) ListSurveyTargets(ctx context.Context, tutorID string) ([]SurveyTarget, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT survey_id, tutor_id, student_id, student_name, group_title, course_name, lessons_remaining, deadline_at, status, created_at
		FROM survey_targets WHERE tutor_id = ? ORDER BY CASE status WHEN 'pending' THEN 0 ELSE 1 END, deadline_at ASC`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	out := []SurveyTarget{}
	for rows.Next() {
		var st SurveyTarget
		if err := rows.Scan(&st.SurveyID, &st.TutorID, &st.StudentID, &st.StudentName, &st.GroupTitle, &st.CourseName, &st.LessonsRemaining, &st.DeadlineAt, &st.Status, &st.CreatedAt); err != nil {
			return nil, err
		}
		var sub SurveySubmission
		if errSub := s.db.QueryRowContext(ctx, `
			SELECT submission_id, survey_id, tutor_id, student_id, nomination_badge, observation_notes,
			       comprehension_score, independence_score, focus_score, stability_score, motivation_score,
			       primary_motivator, parent_feedback_notes, submitted_at
			FROM survey_submissions WHERE survey_id = ?`, st.SurveyID).Scan(
			&sub.SubmissionID, &sub.SurveyID, &sub.TutorID, &sub.StudentID, &sub.NominationBadge, &sub.ObservationNotes,
			&sub.ComprehensionScore, &sub.IndependenceScore, &sub.FocusScore, &sub.StabilityScore, &sub.MotivationScore,
			&sub.PrimaryMotivator, &sub.ParentFeedbackNotes, &sub.SubmittedAt); errSub == nil {
			st.Submission = &sub
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) GetSurveyTarget(ctx context.Context, tutorID, surveyID string) (SurveyTarget, error) {
	var st SurveyTarget
	err := s.db.QueryRowContext(ctx, `
		SELECT survey_id, tutor_id, student_id, student_name, group_title, course_name, lessons_remaining, deadline_at, status, created_at
		FROM survey_targets WHERE tutor_id = ? AND survey_id = ?`, tutorID, surveyID).Scan(
		&st.SurveyID, &st.TutorID, &st.StudentID, &st.StudentName, &st.GroupTitle, &st.CourseName, &st.LessonsRemaining, &st.DeadlineAt, &st.Status, &st.CreatedAt)
	if err != nil {
		return SurveyTarget{}, err
	}
	var sub SurveySubmission
	if errSub := s.db.QueryRowContext(ctx, `
		SELECT submission_id, survey_id, tutor_id, student_id, nomination_badge, observation_notes,
		       comprehension_score, independence_score, focus_score, stability_score, motivation_score,
		       primary_motivator, parent_feedback_notes, submitted_at
		FROM survey_submissions WHERE survey_id = ?`, st.SurveyID).Scan(
		&sub.SubmissionID, &sub.SurveyID, &sub.TutorID, &sub.StudentID, &sub.NominationBadge, &sub.ObservationNotes,
		&sub.ComprehensionScore, &sub.IndependenceScore, &sub.FocusScore, &sub.StabilityScore, &sub.MotivationScore,
		&sub.PrimaryMotivator, &sub.ParentFeedbackNotes, &sub.SubmittedAt); errSub == nil {
		st.Submission = &sub
	}
	return st, nil
}

func (s *Store) SubmitSurvey(ctx context.Context, tutorID string, sub SurveySubmission) (SurveySubmission, error) {
	var st SurveyTarget
	err := s.db.QueryRowContext(ctx, `SELECT survey_id, student_id FROM survey_targets WHERE tutor_id = ? AND survey_id = ?`, tutorID, sub.SurveyID).Scan(&st.SurveyID, &st.StudentID)
	if err != nil {
		return SurveySubmission{}, errors.New("target survei tidak ditemukan")
	}

	sub.SubmissionID = uuid.NewString()
	sub.TutorID = tutorID
	sub.StudentID = st.StudentID
	sub.SubmittedAt = timeutil.FormatWIB(time.Now())

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return SurveySubmission{}, err
	}
	defer func() { _ = tx.Rollback() }()

	_, err = tx.ExecContext(ctx, `
		INSERT INTO survey_submissions (
			submission_id, survey_id, tutor_id, student_id, nomination_badge, observation_notes,
			comprehension_score, independence_score, focus_score, stability_score, motivation_score,
			primary_motivator, parent_feedback_notes, submitted_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(survey_id) DO UPDATE SET
			nomination_badge=excluded.nomination_badge,
			observation_notes=excluded.observation_notes,
			comprehension_score=excluded.comprehension_score,
			independence_score=excluded.independence_score,
			focus_score=excluded.focus_score,
			stability_score=excluded.stability_score,
			motivation_score=excluded.motivation_score,
			primary_motivator=excluded.primary_motivator,
			parent_feedback_notes=excluded.parent_feedback_notes,
			submitted_at=excluded.submitted_at`,
		sub.SubmissionID, sub.SurveyID, sub.TutorID, sub.StudentID, sub.NominationBadge, sub.ObservationNotes,
		sub.ComprehensionScore, sub.IndependenceScore, sub.FocusScore, sub.StabilityScore, sub.MotivationScore,
		sub.PrimaryMotivator, sub.ParentFeedbackNotes, sub.SubmittedAt)
	if err != nil {
		return SurveySubmission{}, err
	}

	_, err = tx.ExecContext(ctx, `UPDATE survey_targets SET status = 'completed' WHERE survey_id = ?`, sub.SurveyID)
	if err != nil {
		return SurveySubmission{}, err
	}

	return sub, tx.Commit()
}

func truncateStr(s string, maxLen int) string {
	s = strings.TrimSpace(s)
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen] + "..."
}
