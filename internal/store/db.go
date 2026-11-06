package store

import (
	"context"
	"database/sql"
	"embed"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"algokanbana/internal/timeutil"

	"github.com/google/uuid"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type Store struct {
	db *sql.DB
}

func Open(dbPath string) (*Store, error) {
	if dbPath != ":memory:" {
		if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
			return nil, fmt.Errorf("create db dir: %w", err)
		}
	}

	dsn := dbPath
	if dbPath != ":memory:" {
		dsn = fmt.Sprintf("file:%s?_pragma=busy_timeout(5000)&_pragma=journal_mode(WAL)&_pragma=synchronous(NORMAL)&_pragma=foreign_keys(ON)&_pragma=temp_store(MEMORY)&_pragma=cache_size(-16000)", filepath.ToSlash(dbPath))
	}

	db, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if dbPath == ":memory:" {
		db.SetMaxOpenConns(1)
	} else {
		db.SetMaxOpenConns(4)
		db.SetMaxIdleConns(4)
	}

	s := &Store{db: db}
	if err := s.migrate(); err != nil {
		_ = db.Close()
		return nil, err
	}
	return s, nil
}

func (s *Store) Close() error {
	return s.db.Close()
}

func (s *Store) DB() *sql.DB {
	return s.db
}

func (s *Store) migrate() error {
	sqlBytes, err := migrationFS.ReadFile("migrations/001_init.sql")
	if err != nil {
		return fmt.Errorf("read migration: %w", err)
	}
	if _, err := s.db.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("exec migration: %w", err)
	}
	return nil
}

func (s *Store) IsEmpty(ctx context.Context) (bool, error) {
	var cnt int
	err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM tutors`).Scan(&cnt)
	return cnt == 0, err
}

func parseJSONSlice(raw string) []string {
	var out []string
	if strings.TrimSpace(raw) != "" {
		_ = json.Unmarshal([]byte(raw), &out)
	}
	if out == nil {
		out = []string{}
	}
	return out
}

func toJSON(v any) string {
	b, err := json.Marshal(v)
	if err != nil {
		return "[]"
	}
	return string(b)
}

func scanTutor(row interface{ Scan(dest ...any) error }) (Tutor, error) {
	var t Tutor
	var specJSON, subjJSON, coursesJSON, formatsJSON, rolesJSON string
	var tgEnabled int
	err := row.Scan(
		&t.TutorID, &t.UserID, &t.LMSUserID, &t.Login,
		&t.FirstName, &t.LastName, &t.Email, &t.Phone, &t.Bio,
		&t.PhotoURL, &t.CVURL, &specJSON, &subjJSON,
		&t.EmploymentType, &t.Status, &t.CountryCode,
		&coursesJSON, &formatsJSON, &t.TLID, &t.TLName,
		&t.TelegramHandle, &tgEnabled, &t.TelegramLinkCode,
		&t.DensityPref, &t.LocalePref, &rolesJSON, &t.UpdatedAt,
	)
	if err != nil {
		return Tutor{}, err
	}
	t.FullName = strings.TrimSpace(t.FirstName + " " + t.LastName)
	t.SpecializationTags = parseJSONSlice(specJSON)
	t.SubjectTags = parseJSONSlice(subjJSON)
	t.CourseTypes = parseJSONSlice(coursesJSON)
	t.LessonFormats = parseJSONSlice(formatsJSON)
	t.AvailableRoles = parseJSONSlice(rolesJSON)
	t.TelegramEnabled = tgEnabled == 1
	t.ProfileIncomplete = strings.TrimSpace(t.Bio) == "" || strings.TrimSpace(t.Phone) == "" || len(t.SpecializationTags) == 0
	return t, nil
}

const tutorColumns = `tutor_id, user_id, lms_user_id, login, first_name, last_name, email, phone, bio,
photo_url, cv_url, specialization_tags_json, subject_tags_json, employment_type, status, country_code,
course_types_json, lesson_formats_json, tl_id, tl_name, telegram_handle, telegram_enabled, telegram_link_code,
density_pref, locale_pref, available_roles_json, updated_at`

func (s *Store) GetTutorByLogin(ctx context.Context, login string) (Tutor, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+tutorColumns+` FROM tutors WHERE LOWER(login) = LOWER(?)`, strings.TrimSpace(login))
	return scanTutor(row)
}

func (s *Store) GetTutorByID(ctx context.Context, tutorID string) (Tutor, error) {
	row := s.db.QueryRowContext(ctx, `SELECT `+tutorColumns+` FROM tutors WHERE tutor_id = ?`, tutorID)
	return scanTutor(row)
}

func (s *Store) UpsertTutor(ctx context.Context, t Tutor) error {
	tgInt := 0
	if t.TelegramEnabled {
		tgInt = 1
	}
	now := timeutil.FormatWIB(time.Now())
	if t.UpdatedAt == "" {
		t.UpdatedAt = now
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tutors (
			tutor_id, user_id, lms_user_id, login, first_name, last_name, email, phone, bio,
			photo_url, cv_url, specialization_tags_json, subject_tags_json, employment_type, status,
			country_code, course_types_json, lesson_formats_json, tl_id, tl_name,
			telegram_handle, telegram_enabled, telegram_link_code, density_pref, locale_pref,
			available_roles_json, updated_at
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(login) DO UPDATE SET
			lms_user_id = excluded.lms_user_id,
			first_name = CASE WHEN tutors.first_name = '' THEN excluded.first_name ELSE tutors.first_name END,
			email = CASE WHEN excluded.email != '' THEN excluded.email ELSE tutors.email END,
			status = excluded.status,
			updated_at = excluded.updated_at`,
		t.TutorID, t.UserID, t.LMSUserID, t.Login, t.FirstName, t.LastName, t.Email, t.Phone, t.Bio,
		t.PhotoURL, t.CVURL, toJSON(t.SpecializationTags), toJSON(t.SubjectTags), t.EmploymentType, t.Status,
		t.CountryCode, toJSON(t.CourseTypes), toJSON(t.LessonFormats), t.TLID, t.TLName,
		t.TelegramHandle, tgInt, t.TelegramLinkCode, t.DensityPref, t.LocalePref,
		toJSON(t.AvailableRoles), t.UpdatedAt,
	)
	return err
}

func (s *Store) SyncTutorGroups(ctx context.Context, tutorID string, groups []Group) error {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback() }()

	stmt, err := tx.PrepareContext(ctx, `
		INSERT INTO groups (
			group_id, tutor_id, title, course_code, course_name, format,
			schedule_summary, start_date, students_count, avg_attendance_pct,
			avg_progress_pct, lms_url, is_archived
		) VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)
		ON CONFLICT(group_id) DO UPDATE SET
			tutor_id = excluded.tutor_id,
			title = excluded.title,
			course_code = excluded.course_code,
			course_name = excluded.course_name,
			format = excluded.format,
			schedule_summary = excluded.schedule_summary,
			start_date = excluded.start_date,
			lms_url = excluded.lms_url,
			is_archived = excluded.is_archived`)
	if err != nil {
		return err
	}
	defer func() { _ = stmt.Close() }()

	for _, g := range groups {
		isArch := 0
		if g.IsArchived {
			isArch = 1
		}
		if _, err := stmt.ExecContext(ctx,
			g.GroupID, tutorID, g.Title, g.CourseCode, g.CourseName, g.Format,
			g.ScheduleSummary, g.StartDate, g.StudentsCount, g.AvgAttendancePct,
			g.AvgProgressPct, g.LMSURL, isArch,
		); err != nil {
			return err
		}
	}
	return tx.Commit()
}

type UpdateProfileInput struct {
	FirstName          *string  `json:"firstName"`
	LastName           *string  `json:"lastName"`
	Phone              *string  `json:"phone"`
	Bio                *string  `json:"bio"`
	SpecializationTags []string `json:"specializationTags"`
	SubjectTags        []string `json:"subjectTags"`
	DensityPref        *string  `json:"densityPreference"`
	LocalePref         *string  `json:"localePreference"`
	TelegramEnabled    *bool    `json:"telegramEnabled"`
}

func (s *Store) UpdateTutorProfile(ctx context.Context, tutorID string, in UpdateProfileInput) (Tutor, error) {
	cur, err := s.GetTutorByID(ctx, tutorID)
	if err != nil {
		return Tutor{}, err
	}
	if in.FirstName != nil && strings.TrimSpace(*in.FirstName) != "" {
		cur.FirstName = strings.TrimSpace(*in.FirstName)
	}
	if in.LastName != nil && strings.TrimSpace(*in.LastName) != "" {
		cur.LastName = strings.TrimSpace(*in.LastName)
	}
	if in.Phone != nil {
		cur.Phone = strings.TrimSpace(*in.Phone)
	}
	if in.Bio != nil {
		cur.Bio = strings.TrimSpace(*in.Bio)
	}
	if in.SpecializationTags != nil {
		cur.SpecializationTags = in.SpecializationTags
	}
	if in.SubjectTags != nil {
		cur.SubjectTags = in.SubjectTags
	}
	if in.DensityPref != nil && (*in.DensityPref == "compact" || *in.DensityPref == "default") {
		cur.DensityPref = *in.DensityPref
	}
	if in.LocalePref != nil && (*in.LocalePref == "id" || *in.LocalePref == "en" || *in.LocalePref == "ru") {
		cur.LocalePref = *in.LocalePref
	}
	if in.TelegramEnabled != nil {
		cur.TelegramEnabled = *in.TelegramEnabled
	}

	tgInt := 0
	if cur.TelegramEnabled {
		tgInt = 1
	}
	now := timeutil.FormatWIB(time.Now())
	_, err = s.db.ExecContext(ctx, `
		UPDATE tutors SET
			first_name = ?, last_name = ?, phone = ?, bio = ?,
			specialization_tags_json = ?, subject_tags_json = ?,
			density_pref = ?, locale_pref = ?, telegram_enabled = ?, updated_at = ?
		WHERE tutor_id = ?`,
		cur.FirstName, cur.LastName, cur.Phone, cur.Bio,
		toJSON(cur.SpecializationTags), toJSON(cur.SubjectTags),
		cur.DensityPref, cur.LocalePref, tgInt, now, tutorID,
	)
	if err != nil {
		return Tutor{}, err
	}
	return s.GetTutorByID(ctx, tutorID)
}

func (s *Store) SetTutorAssetURL(ctx context.Context, tutorID, kind, assetURL string) error {
	col := "photo_url"
	if kind == "cv" {
		col = "cv_url"
	}
	_, err := s.db.ExecContext(ctx, `UPDATE tutors SET `+col+` = ?, updated_at = ? WHERE tutor_id = ?`, assetURL, timeutil.FormatWIB(time.Now()), tutorID)
	return err
}

func (s *Store) UpdateTelegramAction(ctx context.Context, tutorID, action string) (map[string]any, error) {
	now := time.Now()
	switch action {
	case "generate_code":
		code := "NK-" + strings.ToUpper(uuid.NewString()[:8])
		_, err := s.db.ExecContext(ctx, `UPDATE tutors SET telegram_link_code = ?, updated_at = ? WHERE tutor_id = ?`, code, timeutil.FormatWIB(now), tutorID)
		return map[string]any{
			"deepLink":  "https://t.me/AlgonovaTutorBot?start=" + code,
			"code":      code,
			"expiresAt": timeutil.FormatWIB(now.Add(15 * time.Minute)),
		}, err
	case "unlink":
		_, err := s.db.ExecContext(ctx, `UPDATE tutors SET telegram_handle = '', telegram_enabled = 0, telegram_link_code = '', updated_at = ? WHERE tutor_id = ?`, timeutil.FormatWIB(now), tutorID)
		return map[string]any{"unlinked": true}, err
	case "relink_demo":
		_, err := s.db.ExecContext(ctx, `UPDATE tutors SET telegram_handle = '@azhar_faturohman', telegram_enabled = 1, telegram_link_code = '', updated_at = ? WHERE tutor_id = ?`, timeutil.FormatWIB(now), tutorID)
		return map[string]any{"linked": "@azhar_faturohman"}, err
	default:
		return nil, errors.New("unknown telegram action")
	}
}

func (s *Store) ListUpcomingLessons(ctx context.Context, tutorID string, limit int) ([]Lesson, error) {
	nowStr := timeutil.FormatWIB(time.Now().Add(-2 * time.Hour))
	rows, err := s.db.QueryContext(ctx, `
		SELECT lesson_id, tutor_id, group_id, group_title, course_code, format, lesson_title,
		       lesson_number, course_lesson_number, starts_at, ends_at, duration_minutes,
		       meeting_url, prep_url, lms_url, is_completed
		FROM lessons
		WHERE tutor_id = ? AND starts_at >= ?
		ORDER BY starts_at ASC
		LIMIT ?`, tutorID, nowStr, limit)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Lesson{}
	for rows.Next() {
		var l Lesson
		var done int
		if err := rows.Scan(
			&l.LessonID, &l.TutorID, &l.GroupID, &l.GroupTitle, &l.CourseCode, &l.Format, &l.LessonTitle,
			&l.LessonNumber, &l.CourseLessonNumber, &l.StartsAt, &l.EndsAt, &l.DurationMinutes,
			&l.MeetingURL, &l.PrepURL, &l.LMSURL, &done,
		); err != nil {
			return nil, err
		}
		l.IsCompleted = done == 1
		out = append(out, l)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListPortalEvents(ctx context.Context) ([]PortalEvent, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT event_id, title, description, starts_at, ends_at, meeting_url, created_by FROM portal_events ORDER BY starts_at ASC`)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []PortalEvent{}
	for rows.Next() {
		var e PortalEvent
		if err := rows.Scan(&e.EventID, &e.Title, &e.Description, &e.StartsAt, &e.EndsAt, &e.MeetingURL, &e.CreatedBy); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListSavedViews(ctx context.Context, tutorID string) ([]SavedView, error) {
	rows, err := s.db.QueryContext(ctx, `SELECT view_id, page_key, name, query_string, created_at FROM saved_views WHERE tutor_id = ? ORDER BY created_at DESC`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []SavedView{}
	for rows.Next() {
		var v SavedView
		if err := rows.Scan(&v.ViewID, &v.PageKey, &v.Name, &v.QueryString, &v.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, v)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) CreateSavedView(ctx context.Context, tutorID, pageKey, name, queryString string) (SavedView, error) {
	v := SavedView{
		ViewID:      uuid.NewString(),
		PageKey:     strings.TrimSpace(pageKey),
		Name:        strings.TrimSpace(name),
		QueryString: strings.TrimSpace(queryString),
		CreatedAt:   timeutil.FormatWIB(time.Now()),
	}
	_, err := s.db.ExecContext(ctx, `INSERT INTO saved_views (view_id, tutor_id, page_key, name, query_string, created_at) VALUES (?,?,?,?,?,?)`,
		v.ViewID, tutorID, v.PageKey, v.Name, v.QueryString, v.CreatedAt)
	return v, err
}

func (s *Store) DeleteSavedView(ctx context.Context, tutorID, viewID string) error {
	_, err := s.db.ExecContext(ctx, `DELETE FROM saved_views WHERE tutor_id = ? AND view_id = ?`, tutorID, viewID)
	return err
}

func (s *Store) UnreadCounts(ctx context.Context, tutorID string) (tutorUnread int, tlUnread int, openAssessment bool) {
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(SUM(unread_tutor),0), COALESCE(SUM(unread_tl),0) FROM threads WHERE tutor_id = ?`, tutorID).Scan(&tutorUnread, &tlUnread)
	var cnt int
	_ = s.db.QueryRowContext(ctx, `
		SELECT COUNT(1) FROM assessment_cycles c
		WHERE c.is_open = 1 AND NOT EXISTS (
			SELECT 1 FROM assessment_submissions sub WHERE sub.cycle_code = c.cycle_code AND sub.tutor_id = ?
		)`, tutorID).Scan(&cnt)
	return tutorUnread, tlUnread, cnt > 0
}

func (s *Store) SearchPalette(ctx context.Context, tutorID, query string) (map[string]any, error) {
	q := "%" + strings.ToLower(strings.TrimSpace(query)) + "%"
	students := []Student{}
	sRows, err := s.db.QueryContext(ctx, `
		SELECT student_id, tutor_id, full_name, group_id, group_title, group_format, course_code, course_name,
		       attendance_pct, progress_pct, lessons_attended, lessons_total, lessons_remaining, total_lessons_purchased,
		       last_lesson_at, parent_name, parent_phone, lms_url, diagnostic_pct, diagnostic_goal
		FROM students WHERE tutor_id = ? AND (LOWER(full_name) LIKE ? OR LOWER(group_title) LIKE ? OR student_id LIKE ?) LIMIT 8`,
		tutorID, q, q, q)
	if err == nil {
		defer func() { _ = sRows.Close() }()
		for sRows.Next() {
			var st Student
			if err := sRows.Scan(&st.StudentID, &st.TutorID, &st.FullName, &st.GroupID, &st.GroupTitle, &st.GroupFormat, &st.CourseCode, &st.CourseName,
				&st.AttendancePct, &st.ProgressPct, &st.LessonsAttended, &st.LessonsTotal, &st.LessonsRemaining, &st.TotalLessonsPurchased,
				&st.LastLessonAt, &st.ParentName, &st.ParentPhone, &st.LMSURL, &st.DiagnosticPct, &st.DiagnosticGoal); err == nil {
				students = append(students, st)
			}
		}
		_ = sRows.Err()
	}

	groups := []Group{}
	gRows, err := s.db.QueryContext(ctx, `
		SELECT group_id, tutor_id, title, course_code, course_name, format, schedule_summary, start_date,
		       students_count, avg_attendance_pct, avg_progress_pct, lms_url, is_archived
		FROM groups WHERE tutor_id = ? AND (LOWER(title) LIKE ? OR group_id LIKE ? OR LOWER(course_name) LIKE ?) LIMIT 8`,
		tutorID, q, q, q)
	if err == nil {
		defer func() { _ = gRows.Close() }()
		for gRows.Next() {
			var g Group
			var arch int
			if err := gRows.Scan(&g.GroupID, &g.TutorID, &g.Title, &g.CourseCode, &g.CourseName, &g.Format, &g.ScheduleSummary, &g.StartDate,
				&g.StudentsCount, &g.AvgAttendancePct, &g.AvgProgressPct, &g.LMSURL, &arch); err == nil {
				g.IsArchived = arch == 1
				groups = append(groups, g)
			}
		}
		_ = gRows.Err()
	}

	tasks := []Task{}
	tRows, err := s.db.QueryContext(ctx, `
		SELECT task_id, tutor_id, title, description, task_type, status, priority, due_at, assigned_at, completed_at, reason_note, metric_code, week_start, related_group_id, related_group_title, related_student_id, related_student_name, thread_id
		FROM tasks WHERE tutor_id = ? AND (LOWER(title) LIKE ? OR LOWER(related_student_name) LIKE ?) LIMIT 6`,
		tutorID, q, q)
	if err == nil {
		defer func() { _ = tRows.Close() }()
		for tRows.Next() {
			var tk Task
			if err := tRows.Scan(&tk.TaskID, &tk.TutorID, &tk.Title, &tk.Description, &tk.TaskType, &tk.Status, &tk.Priority, &tk.DueAt, &tk.AssignedAt, &tk.CompletedAt, &tk.ReasonNote, &tk.MetricCode, &tk.WeekStart, &tk.RelatedGroupID, &tk.RelatedGroupTitle, &tk.RelatedStudentID, &tk.RelatedStudentName, &tk.ThreadID); err == nil {
				tasks = append(tasks, tk)
			}
		}
		_ = tRows.Err()
	}

	return map[string]any{
		"students": students,
		"groups":   groups,
		"tasks":    tasks,
	}, nil
}

func (s *Store) GetDashboardGroups(ctx context.Context, tutorID string) (map[string]any, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT group_id, tutor_id, title, course_code, course_name, format, schedule_summary, start_date,
		       students_count, avg_attendance_pct, avg_progress_pct, lms_url, is_archived
		FROM groups WHERE tutor_id = ? ORDER BY is_archived ASC, title ASC`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()

	active := []Group{}
	archived := []Group{}
	totalStudents := 0
	attSum := 0
	for rows.Next() {
		var g Group
		var arch int
		if err := rows.Scan(&g.GroupID, &g.TutorID, &g.Title, &g.CourseCode, &g.CourseName, &g.Format, &g.ScheduleSummary, &g.StartDate,
			&g.StudentsCount, &g.AvgAttendancePct, &g.AvgProgressPct, &g.LMSURL, &arch); err != nil {
			return nil, err
		}
		g.IsArchived = arch == 1 || g.StudentsCount == 0
		if g.IsArchived {
			archived = append(archived, g)
		} else {
			active = append(active, g)
			totalStudents += g.StudentsCount
			attSum += g.AvgAttendancePct
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	avgAtt := 0
	if len(active) > 0 {
		avgAtt = attSum / len(active)
	}
	return map[string]any{
		"summary": map[string]any{
			"activeGroupsCount": len(active),
			"archivedCount":     len(archived),
			"totalStudents":     totalStudents,
			"avgAttendancePct":  avgAtt,
		},
		"groups":         active,
		"archivedGroups": archived,
	}, nil
}

func (s *Store) GetGroupDetail(ctx context.Context, tutorID, groupID string) (map[string]any, error) {
	var g Group
	var arch int
	err := s.db.QueryRowContext(ctx, `
		SELECT group_id, tutor_id, title, course_code, course_name, format, schedule_summary, start_date,
		       students_count, avg_attendance_pct, avg_progress_pct, lms_url, is_archived
		FROM groups WHERE tutor_id = ? AND group_id = ?`, tutorID, groupID).Scan(
		&g.GroupID, &g.TutorID, &g.Title, &g.CourseCode, &g.CourseName, &g.Format, &g.ScheduleSummary, &g.StartDate,
		&g.StudentsCount, &g.AvgAttendancePct, &g.AvgProgressPct, &g.LMSURL, &arch)
	if err != nil {
		return nil, err
	}
	g.IsArchived = arch == 1

	sRows, err := s.db.QueryContext(ctx, `
		SELECT student_id, tutor_id, full_name, group_id, group_title, group_format, course_code, course_name,
		       attendance_pct, progress_pct, lessons_attended, lessons_total, lessons_remaining, total_lessons_purchased,
		       last_lesson_at, parent_name, parent_phone, lms_url, diagnostic_pct, diagnostic_goal
		FROM students WHERE tutor_id = ? AND group_id = ? ORDER BY full_name ASC`, tutorID, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = sRows.Close() }()
	students := []Student{}
	for sRows.Next() {
		var st Student
		if err := sRows.Scan(&st.StudentID, &st.TutorID, &st.FullName, &st.GroupID, &st.GroupTitle, &st.GroupFormat, &st.CourseCode, &st.CourseName,
			&st.AttendancePct, &st.ProgressPct, &st.LessonsAttended, &st.LessonsTotal, &st.LessonsRemaining, &st.TotalLessonsPurchased,
			&st.LastLessonAt, &st.ParentName, &st.ParentPhone, &st.LMSURL, &st.DiagnosticPct, &st.DiagnosticGoal); err != nil {
			return nil, err
		}
		students = append(students, st)
	}
	if err := sRows.Err(); err != nil {
		return nil, err
	}

	lRows, err := s.db.QueryContext(ctx, `
		SELECT lesson_id, tutor_id, group_id, group_title, course_code, format, lesson_title,
		       lesson_number, course_lesson_number, starts_at, ends_at, duration_minutes,
		       meeting_url, prep_url, lms_url, is_completed
		FROM lessons WHERE tutor_id = ? AND group_id = ? ORDER BY starts_at DESC LIMIT 8`, tutorID, groupID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = lRows.Close() }()
	lessons := []Lesson{}
	for lRows.Next() {
		var l Lesson
		var done int
		if err := lRows.Scan(&l.LessonID, &l.TutorID, &l.GroupID, &l.GroupTitle, &l.CourseCode, &l.Format, &l.LessonTitle,
			&l.LessonNumber, &l.CourseLessonNumber, &l.StartsAt, &l.EndsAt, &l.DurationMinutes,
			&l.MeetingURL, &l.PrepURL, &l.LMSURL, &done); err == nil {
			l.IsCompleted = done == 1
			lessons = append(lessons, l)
		}
	}
	if err := lRows.Err(); err != nil {
		return nil, err
	}

	return map[string]any{
		"group":         g,
		"students":      students,
		"recentLessons": lessons,
	}, nil
}

func (s *Store) GetMyKPI(ctx context.Context, tutorID, period string) (map[string]any, error) {
	var avgAtt int
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(AVG(avg_attendance_pct), 86) FROM groups WHERE tutor_id = ? AND is_archived = 0`, tutorID).Scan(&avgAtt)

	var totalTasks, completedTasks int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1), COALESCE(SUM(CASE WHEN status='completed' THEN 1 ELSE 0 END), 0) FROM tasks WHERE tutor_id = ?`, tutorID).Scan(&totalTasks, &completedTasks)

	var avgAssess int
	_ = s.db.QueryRowContext(ctx, `SELECT COALESCE(AVG(score_pct), 98) FROM assessment_history WHERE tutor_id = ?`, tutorID).Scan(&avgAssess)

	var fbCount int
	_ = s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM regular_feedback_reports WHERE tutor_id = ?`, tutorID).Scan(&fbCount)
	feedbackScore := min(50, 44+fbCount*2)
	assessmentScore := avgAssess / 2 // out of 50
	disciplineTotal := (feedbackScore + assessmentScore)

	kpiTasks, _ := s.ListTasks(ctx, tutorID)
	weeks := []map[string]any{
		{"week": 1, "label": "Minggu 1", "weekStart": "2026-10-05", "attendancePct": max(82, avgAtt-2), "tasksDone": 4, "tasksTotal": 4},
		{"week": 2, "label": "Minggu 2", "weekStart": "2026-10-12", "attendancePct": avgAtt + 1, "tasksDone": 5, "tasksTotal": 5},
		{"week": 3, "label": "Minggu 3", "weekStart": "2026-10-19", "attendancePct": avgAtt + 2, "tasksDone": completedTasks, "tasksTotal": max(totalTasks, 1)},
		{"week": 4, "label": "Minggu 4 (Berjalan)", "weekStart": "2026-10-26", "attendancePct": avgAtt, "tasksDone": completedTasks, "tasksTotal": max(totalTasks, 1)},
	}

	return map[string]any{
		"period":          period,
		"attendanceKpi":   avgAtt,
		"disciplineScore": disciplineTotal,
		"feedbackScore":   feedbackScore,
		"assessmentScore": assessmentScore,
		"qualityScore":    94,
		"thresholds": map[string][2]int{
			"attendance": {80, 88},
			"discipline": {45, 50},
			"quality":    {68, 75},
		},
		"weeks":    weeks,
		"kpiTasks": kpiTasks,
	}, nil
}

func (s *Store) ListStudents(ctx context.Context, tutorID string) ([]Student, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT student_id, tutor_id, full_name, group_id, group_title, group_format, course_code, course_name,
		       attendance_pct, progress_pct, lessons_attended, lessons_total, lessons_remaining, total_lessons_purchased,
		       last_lesson_at, parent_name, parent_phone, lms_url, diagnostic_pct, diagnostic_goal
		FROM students WHERE tutor_id = ? ORDER BY full_name ASC`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Student{}
	for rows.Next() {
		var st Student
		if err := rows.Scan(&st.StudentID, &st.TutorID, &st.FullName, &st.GroupID, &st.GroupTitle, &st.GroupFormat, &st.CourseCode, &st.CourseName,
			&st.AttendancePct, &st.ProgressPct, &st.LessonsAttended, &st.LessonsTotal, &st.LessonsRemaining, &st.TotalLessonsPurchased,
			&st.LastLessonAt, &st.ParentName, &st.ParentPhone, &st.LMSURL, &st.DiagnosticPct, &st.DiagnosticGoal); err != nil {
			return nil, err
		}
		out = append(out, st)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) GetStudentDetail(ctx context.Context, tutorID, studentID string) (map[string]any, error) {
	var st Student
	err := s.db.QueryRowContext(ctx, `
		SELECT student_id, tutor_id, full_name, group_id, group_title, group_format, course_code, course_name,
		       attendance_pct, progress_pct, lessons_attended, lessons_total, lessons_remaining, total_lessons_purchased,
		       last_lesson_at, parent_name, parent_phone, lms_url, diagnostic_pct, diagnostic_goal
		FROM students WHERE tutor_id = ? AND student_id = ?`, tutorID, studentID).Scan(
		&st.StudentID, &st.TutorID, &st.FullName, &st.GroupID, &st.GroupTitle, &st.GroupFormat, &st.CourseCode, &st.CourseName,
		&st.AttendancePct, &st.ProgressPct, &st.LessonsAttended, &st.LessonsTotal, &st.LessonsRemaining, &st.TotalLessonsPurchased,
		&st.LastLessonAt, &st.ParentName, &st.ParentPhone, &st.LMSURL, &st.DiagnosticPct, &st.DiagnosticGoal)
	if err != nil {
		return nil, err
	}

	reports := []RegularFeedbackReport{}
	rRows, err := s.db.QueryContext(ctx, `
		SELECT report_id, tutor_id, week_start, lms_lesson_id, lms_student_id, student_name, parent_phone,
		       group_title, lesson_number, course_code, month_no, attendance, engagement, tasks,
		       custom1, custom2, stars, project_url, assembled_text, updated_at
		FROM regular_feedback_reports WHERE tutor_id = ? AND lms_student_id = ? ORDER BY updated_at DESC`, tutorID, studentID)
	if err == nil {
		defer func() { _ = rRows.Close() }()
		for rRows.Next() {
			var rp RegularFeedbackReport
			if err := rRows.Scan(&rp.ReportID, &rp.TutorID, &rp.WeekStart, &rp.LMSLessonID, &rp.LMSStudentID, &rp.StudentName, &rp.ParentPhone,
				&rp.GroupTitle, &rp.LessonNumber, &rp.CourseCode, &rp.MonthNo, &rp.Attendance, &rp.Engagement, &rp.Tasks,
				&rp.Custom1, &rp.Custom2, &rp.Stars, &rp.ProjectURL, &rp.AssembledText, &rp.UpdatedAt); err == nil {
				reports = append(reports, rp)
			}
		}
		_ = rRows.Err()
	}

	return map[string]any{
		"student":         st,
		"feedbackReports": reports,
	}, nil
}

// CreateTLReport fixes NanoKanbana Bug #1 by accepting numeric LMS group IDs OR UUIDs cleanly.
func (s *Store) CreateTLReport(ctx context.Context, rep TLReport) (TLReport, error) {
	rep.ReportID = uuid.NewString()
	rep.Status = "open"
	rep.CreatedAt = timeutil.FormatWIB(time.Now())
	if rep.GroupID != "" && rep.GroupTitle == "" {
		_ = s.db.QueryRowContext(ctx, `SELECT title FROM groups WHERE group_id = ?`, rep.GroupID).Scan(&rep.GroupTitle)
	}
	if rep.StudentID != "" && rep.StudentName == "" {
		_ = s.db.QueryRowContext(ctx, `SELECT full_name FROM students WHERE student_id = ?`, rep.StudentID).Scan(&rep.StudentName)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tl_reports (report_id, tutor_id, tutor_name, report_type, group_id, group_title, student_id, student_name, urgency, message, status, tl_reply, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		rep.ReportID, rep.TutorID, rep.TutorName, rep.ReportType, rep.GroupID, rep.GroupTitle, rep.StudentID, rep.StudentName, rep.Urgency, rep.Message, rep.Status, "", rep.CreatedAt)
	return rep, err
}

func (s *Store) ListTLReports(ctx context.Context, tutorID string) ([]TLReport, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT report_id, tutor_id, tutor_name, report_type, group_id, group_title, student_id, student_name, urgency, message, status, tl_reply, created_at
		FROM tl_reports WHERE tutor_id = ? ORDER BY created_at DESC`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []TLReport{}
	for rows.Next() {
		var r TLReport
		if err := rows.Scan(&r.ReportID, &r.TutorID, &r.TutorName, &r.ReportType, &r.GroupID, &r.GroupTitle, &r.StudentID, &r.StudentName, &r.Urgency, &r.Message, &r.Status, &r.TLReply, &r.CreatedAt); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ReviewTLReport(ctx context.Context, reportID, status, tlReply string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE tl_reports SET status = ?, tl_reply = ? WHERE report_id = ?`, status, strings.TrimSpace(tlReply), reportID)
	return err
}

func (s *Store) ListTasks(ctx context.Context, tutorID string) ([]Task, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT task_id, tutor_id, title, description, task_type, status, priority, due_at, assigned_at,
		       completed_at, reason_note, metric_code, week_start, related_group_id, related_group_title,
		       related_student_id, related_student_name, thread_id
		FROM tasks WHERE tutor_id = ? ORDER BY CASE status WHEN 'overdue' THEN 0 WHEN 'assigned' THEN 1 WHEN 'escalated' THEN 2 WHEN 'completed' THEN 3 ELSE 4 END, due_at ASC`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Task{}
	for rows.Next() {
		var tk Task
		if err := rows.Scan(&tk.TaskID, &tk.TutorID, &tk.Title, &tk.Description, &tk.TaskType, &tk.Status, &tk.Priority, &tk.DueAt, &tk.AssignedAt,
			&tk.CompletedAt, &tk.ReasonNote, &tk.MetricCode, &tk.WeekStart, &tk.RelatedGroupID, &tk.RelatedGroupTitle,
			&tk.RelatedStudentID, &tk.RelatedStudentName, &tk.ThreadID); err != nil {
			return nil, err
		}
		out = append(out, tk)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) CreateTask(ctx context.Context, tk Task) (Task, error) {
	if tk.TaskID == "" {
		tk.TaskID = uuid.NewString()
	}
	if tk.TaskType == "" {
		tk.TaskType = "personal"
	}
	if tk.Status == "" {
		tk.Status = "assigned"
	}
	if tk.Priority == "" {
		tk.Priority = "medium"
	}
	tk.AssignedAt = timeutil.FormatWIB(time.Now())
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO tasks (task_id, tutor_id, title, description, task_type, status, priority, due_at, assigned_at, completed_at, reason_note, metric_code, week_start, related_group_id, related_group_title, related_student_id, related_student_name, thread_id)
		VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`,
		tk.TaskID, tk.TutorID, tk.Title, tk.Description, tk.TaskType, tk.Status, tk.Priority, tk.DueAt, tk.AssignedAt, "", "", tk.MetricCode, tk.WeekStart, tk.RelatedGroupID, tk.RelatedGroupTitle, tk.RelatedStudentID, tk.RelatedStudentName, tk.ThreadID)
	return tk, err
}

func (s *Store) BulkUpdateTasks(ctx context.Context, tutorID string, taskIDs []string, action, reason string) ([]Task, error) {
	if len(taskIDs) == 0 {
		return nil, errors.New("taskIds tidak boleh kosong")
	}
	nowStr := timeutil.FormatWIB(time.Now())
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, err
	}
	defer func() { _ = tx.Rollback() }()

	for _, id := range taskIDs {
		var metricCode string
		err := tx.QueryRowContext(ctx, `SELECT metric_code FROM tasks WHERE tutor_id = ? AND task_id = ?`, tutorID, id).Scan(&metricCode)
		if err != nil {
			continue
		}
		switch action {
		case "complete":
			_, err = tx.ExecContext(ctx, `UPDATE tasks SET status = 'completed', completed_at = ?, reason_note = ? WHERE tutor_id = ? AND task_id = ?`, nowStr, reason, tutorID, id)
		case "escalate":
			if len(strings.TrimSpace(reason)) < 5 {
				return nil, errors.New("alasan eskalasi minimal 5 karakter")
			}
			_, err = tx.ExecContext(ctx, `UPDATE tasks SET status = 'escalated', reason_note = ? WHERE tutor_id = ? AND task_id = ?`, strings.TrimSpace(reason), tutorID, id)
		case "cancel":
			if metricCode != "" {
				return nil, errors.New("tugas KPI wajib tidak dapat dibatalkan (hanya bisa diselesaikan atau dieskalasi)")
			}
			if len(strings.TrimSpace(reason)) < 5 {
				return nil, errors.New("alasan pembatalan minimal 5 karakter")
			}
			_, err = tx.ExecContext(ctx, `UPDATE tasks SET status = 'cancelled', reason_note = ? WHERE tutor_id = ? AND task_id = ?`, strings.TrimSpace(reason), tutorID, id)
		case "reopen":
			_, err = tx.ExecContext(ctx, `UPDATE tasks SET status = 'assigned', completed_at = '' WHERE tutor_id = ? AND task_id = ?`, tutorID, id)
		default:
			return nil, fmt.Errorf("aksi tugas tidak dikenal: %s", action)
		}
		if err != nil {
			return nil, err
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s.ListTasks(ctx, tutorID)
}

func (s *Store) MoveTask(ctx context.Context, tutorID, taskID, targetStatus, reason string) (Task, error) {
	nowStr := timeutil.FormatWIB(time.Now())
	var metricCode string
	err := s.db.QueryRowContext(ctx, `SELECT metric_code FROM tasks WHERE tutor_id = ? AND task_id = ?`, tutorID, taskID).Scan(&metricCode)
	if err != nil {
		return Task{}, errors.New("tugas tidak ditemukan")
	}

	switch targetStatus {
	case "completed":
		_, err = s.db.ExecContext(ctx, `UPDATE tasks SET status = 'completed', completed_at = ?, reason_note = ? WHERE tutor_id = ? AND task_id = ?`, nowStr, reason, tutorID, taskID)
	case "escalated":
		if len(strings.TrimSpace(reason)) < 5 {
			return Task{}, errors.New("alasan eskalasi minimal 5 karakter")
		}
		_, err = s.db.ExecContext(ctx, `UPDATE tasks SET status = 'escalated', reason_note = ? WHERE tutor_id = ? AND task_id = ?`, strings.TrimSpace(reason), tutorID, taskID)
	case "cancelled":
		if metricCode != "" {
			return Task{}, errors.New("tugas KPI wajib tidak dapat dibatalkan (hanya bisa diselesaikan atau dieskalasi)")
		}
		if len(strings.TrimSpace(reason)) < 5 {
			return Task{}, errors.New("alasan pembatalan minimal 5 karakter")
		}
		_, err = s.db.ExecContext(ctx, `UPDATE tasks SET status = 'cancelled', reason_note = ? WHERE tutor_id = ? AND task_id = ?`, strings.TrimSpace(reason), tutorID, taskID)
	case "assigned":
		_, err = s.db.ExecContext(ctx, `UPDATE tasks SET status = 'assigned', completed_at = '' WHERE tutor_id = ? AND task_id = ?`, tutorID, taskID)
	case "overdue":
		_, err = s.db.ExecContext(ctx, `UPDATE tasks SET status = 'overdue' WHERE tutor_id = ? AND task_id = ?`, tutorID, taskID)
	default:
		return Task{}, fmt.Errorf("status target tidak valid: %s", targetStatus)
	}
	if err != nil {
		return Task{}, err
	}

	var tk Task
	err = s.db.QueryRowContext(ctx, `
		SELECT task_id, tutor_id, title, description, task_type, status, priority, due_at, assigned_at,
		       completed_at, reason_note, metric_code, week_start, related_group_id, related_group_title,
		       related_student_id, related_student_name, thread_id
		FROM tasks WHERE tutor_id = ? AND task_id = ?`, tutorID, taskID).Scan(
		&tk.TaskID, &tk.TutorID, &tk.Title, &tk.Description, &tk.TaskType, &tk.Status, &tk.Priority, &tk.DueAt, &tk.AssignedAt,
		&tk.CompletedAt, &tk.ReasonNote, &tk.MetricCode, &tk.WeekStart, &tk.RelatedGroupID, &tk.RelatedGroupTitle,
		&tk.RelatedStudentID, &tk.RelatedStudentName, &tk.ThreadID)
	return tk, err
}

func (s *Store) ListEligibleUrgentSubLessons(ctx context.Context, tutorID string) ([]Lesson, error) {
	fromStr := timeutil.FormatWIB(time.Now().Add(50 * time.Minute))
	toStr := timeutil.FormatWIB(time.Now().Add(72 * time.Hour))
	rows, err := s.db.QueryContext(ctx, `
		SELECT lesson_id, tutor_id, group_id, group_title, course_code, format, lesson_title,
		       lesson_number, course_lesson_number, starts_at, ends_at, duration_minutes,
		       meeting_url, prep_url, lms_url, is_completed
		FROM lessons
		WHERE tutor_id = ? AND starts_at >= ? AND starts_at <= ? AND is_completed = 0
		ORDER BY starts_at ASC`, tutorID, fromStr, toStr)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []Lesson{}
	for rows.Next() {
		var l Lesson
		var done int
		if err := rows.Scan(&l.LessonID, &l.TutorID, &l.GroupID, &l.GroupTitle, &l.CourseCode, &l.Format, &l.LessonTitle,
			&l.LessonNumber, &l.CourseLessonNumber, &l.StartsAt, &l.EndsAt, &l.DurationMinutes,
			&l.MeetingURL, &l.PrepURL, &l.LMSURL, &done); err == nil {
			out = append(out, l)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) ListUrgentSubRequests(ctx context.Context, tutorID string) ([]UrgentSubRequest, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT request_id, tutor_id, tutor_name, lms_lesson_id, group_title, course_title, lesson_starts_at, reason, proof_urls_json, status, created_at
		FROM urgent_sub_requests WHERE tutor_id = ? ORDER BY created_at DESC`, tutorID)
	if err != nil {
		return nil, err
	}
	defer func() { _ = rows.Close() }()
	out := []UrgentSubRequest{}
	for rows.Next() {
		var r UrgentSubRequest
		var proofs string
		if err := rows.Scan(&r.RequestID, &r.TutorID, &r.TutorName, &r.LMSLessonID, &r.GroupTitle, &r.CourseTitle, &r.LessonStartsAt, &r.Reason, &proofs, &r.Status, &r.CreatedAt); err == nil {
			r.ProofURLs = parseJSONSlice(proofs)
			out = append(out, r)
		}
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func (s *Store) CreateUrgentSubRequest(ctx context.Context, req UrgentSubRequest) (UrgentSubRequest, error) {
	req.RequestID = uuid.NewString()
	req.Status = "pending"
	req.CreatedAt = timeutil.FormatWIB(time.Now())
	if req.GroupTitle == "" || req.LessonStartsAt == "" {
		_ = s.db.QueryRowContext(ctx, `SELECT group_title, starts_at FROM lessons WHERE lesson_id = ?`, req.LMSLessonID).Scan(&req.GroupTitle, &req.LessonStartsAt)
	}
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO urgent_sub_requests (request_id, tutor_id, tutor_name, lms_lesson_id, group_title, course_title, lesson_starts_at, reason, proof_urls_json, status, created_at)
		VALUES (?,?,?,?,?,?,?,?,?,?,?)`,
		req.RequestID, req.TutorID, req.TutorName, req.LMSLessonID, req.GroupTitle, req.CourseTitle, req.LessonStartsAt, req.Reason, toJSON(req.ProofURLs), req.Status, req.CreatedAt)
	return req, err
}

func (s *Store) UpdateUrgentSubStatus(ctx context.Context, requestID, status string) error {
	_, err := s.db.ExecContext(ctx, `UPDATE urgent_sub_requests SET status = ? WHERE request_id = ?`, status, requestID)
	return err
}
