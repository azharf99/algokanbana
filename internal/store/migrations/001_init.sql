CREATE TABLE IF NOT EXISTS tutors (
    tutor_id TEXT PRIMARY KEY,
    user_id TEXT NOT NULL UNIQUE,
    lms_user_id INTEGER NOT NULL,
    login TEXT NOT NULL UNIQUE,
    first_name TEXT NOT NULL,
    last_name TEXT NOT NULL,
    email TEXT NOT NULL,
    phone TEXT NOT NULL DEFAULT '',
    bio TEXT NOT NULL DEFAULT '',
    photo_url TEXT NOT NULL DEFAULT '',
    cv_url TEXT NOT NULL DEFAULT '',
    specialization_tags_json TEXT NOT NULL DEFAULT '[]',
    subject_tags_json TEXT NOT NULL DEFAULT '[]',
    employment_type TEXT NOT NULL DEFAULT 'part_time',
    status TEXT NOT NULL DEFAULT 'active',
    country_code TEXT NOT NULL DEFAULT 'ID',
    course_types_json TEXT NOT NULL DEFAULT '["coding"]',
    lesson_formats_json TEXT NOT NULL DEFAULT '["group","private"]',
    tl_id TEXT NOT NULL DEFAULT '',
    tl_name TEXT NOT NULL DEFAULT '',
    telegram_handle TEXT NOT NULL DEFAULT '',
    telegram_enabled INTEGER NOT NULL DEFAULT 1,
    telegram_link_code TEXT NOT NULL DEFAULT '',
    density_pref TEXT NOT NULL DEFAULT 'default',
    locale_pref TEXT NOT NULL DEFAULT 'id',
    available_roles_json TEXT NOT NULL DEFAULT '["tutor","team_lead"]',
    updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS groups (
    group_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    title TEXT NOT NULL,
    course_code TEXT NOT NULL,
    course_name TEXT NOT NULL,
    format TEXT NOT NULL, -- 'group' | 'private'
    schedule_summary TEXT NOT NULL,
    start_date TEXT NOT NULL,
    students_count INTEGER NOT NULL DEFAULT 0,
    avg_attendance_pct INTEGER NOT NULL DEFAULT 0,
    avg_progress_pct INTEGER NOT NULL DEFAULT 0,
    lms_url TEXT NOT NULL,
    is_archived INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_groups_tutor ON groups(tutor_id, is_archived);

CREATE TABLE IF NOT EXISTS students (
    student_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    full_name TEXT NOT NULL,
    group_id TEXT NOT NULL,
    group_title TEXT NOT NULL,
    group_format TEXT NOT NULL DEFAULT 'group',
    course_code TEXT NOT NULL,
    course_name TEXT NOT NULL,
    attendance_pct INTEGER NOT NULL DEFAULT 0,
    progress_pct INTEGER NOT NULL DEFAULT 0,
    lessons_attended INTEGER NOT NULL DEFAULT 0,
    lessons_total INTEGER NOT NULL DEFAULT 0,
    lessons_remaining INTEGER NOT NULL DEFAULT 0,
    total_lessons_purchased INTEGER NOT NULL DEFAULT 0,
    last_lesson_at TEXT NOT NULL DEFAULT '',
    parent_name TEXT NOT NULL DEFAULT '',
    parent_phone TEXT NOT NULL DEFAULT '',
    lms_url TEXT NOT NULL DEFAULT '',
    diagnostic_pct INTEGER NOT NULL DEFAULT 85,
    diagnostic_goal TEXT NOT NULL DEFAULT 'Penguasaan Logika & Proyek Mandiri'
);
CREATE INDEX IF NOT EXISTS idx_students_tutor_group ON students(tutor_id, group_id);

CREATE TABLE IF NOT EXISTS lessons (
    lesson_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    group_id TEXT NOT NULL,
    group_title TEXT NOT NULL,
    course_code TEXT NOT NULL,
    format TEXT NOT NULL, -- 'group' | 'private'
    lesson_title TEXT NOT NULL,
    lesson_number INTEGER NOT NULL DEFAULT 1,
    course_lesson_number INTEGER NOT NULL DEFAULT 1,
    starts_at TEXT NOT NULL, -- RFC3339 WIB (+07:00) normalized from LMS +03:00
    ends_at TEXT NOT NULL,
    duration_minutes INTEGER NOT NULL DEFAULT 90,
    meeting_url TEXT NOT NULL DEFAULT '',
    prep_url TEXT NOT NULL DEFAULT '',
    lms_url TEXT NOT NULL DEFAULT '',
    is_completed INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_lessons_tutor_starts ON lessons(tutor_id, starts_at);

CREATE TABLE IF NOT EXISTS lesson_students (
    lesson_id TEXT NOT NULL,
    student_id TEXT NOT NULL,
    student_name TEXT NOT NULL,
    attendance_status TEXT NOT NULL DEFAULT 'attended', -- 'attended' | 'missed' | 'excused' | 'upcoming'
    PRIMARY KEY (lesson_id, student_id)
);

CREATE TABLE IF NOT EXISTS tasks (
    task_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    title TEXT NOT NULL,
    description TEXT NOT NULL DEFAULT '',
    task_type TEXT NOT NULL DEFAULT 'personal',
    status TEXT NOT NULL DEFAULT 'assigned', -- 'assigned' | 'completed' | 'overdue' | 'escalated' | 'cancelled'
    priority TEXT NOT NULL DEFAULT 'medium', -- 'low' | 'medium' | 'high'
    due_at TEXT NOT NULL DEFAULT '',
    assigned_at TEXT NOT NULL,
    completed_at TEXT NOT NULL DEFAULT '',
    reason_note TEXT NOT NULL DEFAULT '',
    metric_code TEXT NOT NULL DEFAULT '',
    week_start TEXT NOT NULL DEFAULT '',
    related_group_id TEXT NOT NULL DEFAULT '',
    related_group_title TEXT NOT NULL DEFAULT '',
    related_student_id TEXT NOT NULL DEFAULT '',
    related_student_name TEXT NOT NULL DEFAULT '',
    thread_id TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_tasks_tutor_status ON tasks(tutor_id, status);

CREATE TABLE IF NOT EXISTS urgent_sub_requests (
    request_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    tutor_name TEXT NOT NULL,
    lms_lesson_id TEXT NOT NULL,
    group_title TEXT NOT NULL,
    course_title TEXT NOT NULL,
    lesson_starts_at TEXT NOT NULL,
    reason TEXT NOT NULL,
    proof_urls_json TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending' | 'approved' | 'withdrawn'
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS regular_feedback_reports (
    report_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    week_start TEXT NOT NULL,
    lms_lesson_id TEXT NOT NULL,
    lms_student_id TEXT NOT NULL,
    student_name TEXT NOT NULL,
    parent_phone TEXT NOT NULL DEFAULT '',
    group_title TEXT NOT NULL DEFAULT '',
    lesson_number INTEGER NOT NULL DEFAULT 4,
    course_code TEXT NOT NULL,
    month_no INTEGER NOT NULL DEFAULT 1,
    attendance TEXT NOT NULL,
    engagement TEXT NOT NULL,
    tasks TEXT NOT NULL,
    custom1 TEXT NOT NULL DEFAULT '',
    custom2 TEXT NOT NULL DEFAULT '',
    stars INTEGER NOT NULL DEFAULT 5,
    project_url TEXT NOT NULL DEFAULT '',
    assembled_text TEXT NOT NULL,
    updated_at TEXT NOT NULL,
    UNIQUE(tutor_id, lms_lesson_id, lms_student_id)
);
CREATE INDEX IF NOT EXISTS idx_reg_feedback_week ON regular_feedback_reports(tutor_id, week_start);

CREATE TABLE IF NOT EXISTS mt_feedback_entries (
    entry_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    week_start TEXT NOT NULL,
    lms_lesson_id TEXT NOT NULL,
    lms_student_id TEXT NOT NULL,
    student_name TEXT NOT NULL,
    lesson_number INTEGER NOT NULL DEFAULT 1,
    lesson_topic TEXT NOT NULL,
    recording_url TEXT NOT NULL DEFAULT '',
    understanding INTEGER NOT NULL DEFAULT 5,
    independence INTEGER NOT NULL DEFAULT 5,
    accuracy INTEGER NOT NULL DEFAULT 5,
    engagement TEXT NOT NULL DEFAULT 'active',
    what_worked TEXT NOT NULL DEFAULT '',
    development_area TEXT NOT NULL DEFAULT '',
    cause TEXT NOT NULL DEFAULT '',
    next_steps TEXT NOT NULL DEFAULT '',
    tutor_comment TEXT NOT NULL DEFAULT '',
    is_on_time INTEGER NOT NULL DEFAULT 1,
    source TEXT NOT NULL DEFAULT 'portal',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS availability_slots (
    tutor_id TEXT NOT NULL,
    weekday INTEGER NOT NULL,   -- 0=Mon .. 6=Sun
    start_min INTEGER NOT NULL, -- minutes from 00:00 WIB (420..1350)
    status TEXT NOT NULL,       -- 'available' | 'avail_for_group' | 'not_available' | 'booked' | 'booked_for_group'
    sheet_cell TEXT NOT NULL DEFAULT '',
    updated_at TEXT NOT NULL,
    PRIMARY KEY (tutor_id, weekday, start_min)
);

CREATE TABLE IF NOT EXISTS reschedule_requests (
    request_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    tutor_name TEXT NOT NULL,
    lms_lesson_id TEXT NOT NULL,
    lms_group_id TEXT NOT NULL DEFAULT '',
    group_name TEXT NOT NULL DEFAULT '',
    group_format TEXT NOT NULL DEFAULT 'private',
    lesson_number INTEGER NOT NULL DEFAULT 1,
    req_type TEXT NOT NULL, -- 'otr' | 'permanent'
    original_starts_at TEXT NOT NULL,
    new_starts_at TEXT NOT NULL,
    note TEXT NOT NULL DEFAULT '',
    proof_urls_json TEXT NOT NULL DEFAULT '[]',
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending' | 'approved' | 'rejected' | 'withdrawn'
    tl_note TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL,
    reviewed_at TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_reschedule_tutor ON reschedule_requests(tutor_id, status);

CREATE TABLE IF NOT EXISTS workload_requests (
    request_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    tutor_name TEXT NOT NULL,
    req_type TEXT NOT NULL,
    reason TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS tl_reports (
    report_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    tutor_name TEXT NOT NULL,
    report_type TEXT NOT NULL,
    group_id TEXT NOT NULL DEFAULT '',
    group_title TEXT NOT NULL DEFAULT '',
    student_id TEXT NOT NULL DEFAULT '',
    student_name TEXT NOT NULL DEFAULT '',
    urgency TEXT NOT NULL DEFAULT 'medium',
    message TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open', -- 'open' | 'acknowledged' | 'resolved'
    tl_reply TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS assessment_cycles (
    cycle_code TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    channel TEXT NOT NULL DEFAULT 'nk',
    deadline_at TEXT NOT NULL,
    is_open INTEGER NOT NULL DEFAULT 1,
    questions_json TEXT NOT NULL DEFAULT '[]'
);

CREATE TABLE IF NOT EXISTS assessment_submissions (
    submission_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    cycle_code TEXT NOT NULL,
    programs_json TEXT NOT NULL DEFAULT '["coding"]',
    answers_json TEXT NOT NULL DEFAULT '{}',
    score_pct INTEGER NOT NULL DEFAULT 100,
    submitted_at TEXT NOT NULL,
    UNIQUE(tutor_id, cycle_code)
);

CREATE TABLE IF NOT EXISTS assessment_history (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    tutor_id TEXT NOT NULL,
    cycle_code TEXT NOT NULL,
    cycle_label TEXT NOT NULL,
    month_key TEXT NOT NULL,
    program TEXT NOT NULL DEFAULT 'Coding & Design',
    score_pct INTEGER NOT NULL,
    max_pct INTEGER NOT NULL DEFAULT 100,
    status TEXT NOT NULL DEFAULT 'passed',
    graded_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS kb_articles (
    article_id TEXT PRIMARY KEY,
    category TEXT NOT NULL,
    locale TEXT NOT NULL DEFAULT 'id',
    title TEXT NOT NULL,
    summary TEXT NOT NULL,
    content_md TEXT NOT NULL,
    helpful_count INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS threads (
    thread_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    subject TEXT NOT NULL,
    request_type TEXT NOT NULL DEFAULT 'general',
    target_team TEXT NOT NULL DEFAULT 'tl', -- 'tl' | 'cs' | 'ism' | 'hr' | 'tech'
    priority TEXT NOT NULL DEFAULT 'medium',
    origin TEXT NOT NULL DEFAULT 'direct', -- 'direct' | 'kb_unresolved' | 'escalated' | 'broadcast'
    related_group_ref TEXT NOT NULL DEFAULT '',
    related_student_ref TEXT NOT NULL DEFAULT '',
    unread_tutor INTEGER NOT NULL DEFAULT 0,
    unread_tl INTEGER NOT NULL DEFAULT 0,
    status TEXT NOT NULL DEFAULT 'open',
    updated_at TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS messages (
    message_id TEXT PRIMARY KEY,
    thread_id TEXT NOT NULL,
    sender_role TEXT NOT NULL, -- 'tutor' | 'team_lead' | 'system'
    sender_name TEXT NOT NULL,
    kind TEXT NOT NULL DEFAULT 'text', -- 'text' | 'request' | 'task' | 'broadcast' | 'escalation'
    body TEXT NOT NULL,
    attachment_url TEXT NOT NULL DEFAULT '',
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_messages_thread ON messages(thread_id, created_at);

CREATE TABLE IF NOT EXISTS saved_views (
    view_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    page_key TEXT NOT NULL,
    name TEXT NOT NULL,
    query_string TEXT NOT NULL,
    created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS portal_events (
    event_id TEXT PRIMARY KEY,
    title TEXT NOT NULL,
    description TEXT NOT NULL,
    starts_at TEXT NOT NULL,
    ends_at TEXT NOT NULL,
    meeting_url TEXT NOT NULL DEFAULT '',
    created_by TEXT NOT NULL DEFAULT 'Academic Team Algonova'
);

CREATE TABLE IF NOT EXISTS survey_targets (
    survey_id TEXT PRIMARY KEY,
    tutor_id TEXT NOT NULL,
    student_id TEXT NOT NULL,
    student_name TEXT NOT NULL,
    group_title TEXT NOT NULL,
    course_name TEXT NOT NULL,
    lessons_remaining INTEGER NOT NULL DEFAULT 4,
    deadline_at TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending', -- 'pending' | 'completed'
    created_at TEXT NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_surveys_tutor ON survey_targets(tutor_id, status);

CREATE TABLE IF NOT EXISTS survey_submissions (
    submission_id TEXT PRIMARY KEY,
    survey_id TEXT NOT NULL UNIQUE,
    tutor_id TEXT NOT NULL,
    student_id TEXT NOT NULL,
    nomination_badge TEXT NOT NULL,
    observation_notes TEXT NOT NULL,
    comprehension_score INTEGER NOT NULL DEFAULT 5,
    independence_score INTEGER NOT NULL DEFAULT 5,
    focus_score INTEGER NOT NULL DEFAULT 5,
    stability_score INTEGER NOT NULL DEFAULT 5,
    motivation_score INTEGER NOT NULL DEFAULT 5,
    primary_motivator TEXT NOT NULL DEFAULT '',
    parent_feedback_notes TEXT NOT NULL DEFAULT '',
    submitted_at TEXT NOT NULL
);
