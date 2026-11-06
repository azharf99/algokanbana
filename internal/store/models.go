package store

type Tutor struct {
	TutorID            string   `json:"tutorId"`
	UserID             string   `json:"userId"`
	LMSUserID          int64    `json:"lmsUserId"`
	Login              string   `json:"login"`
	FirstName          string   `json:"firstName"`
	LastName           string   `json:"lastName"`
	FullName           string   `json:"fullName"`
	Email              string   `json:"email"`
	Phone              string   `json:"phone"`
	Bio                string   `json:"bio"`
	PhotoURL           string   `json:"photoUrl"`
	CVURL              string   `json:"cvUrl"`
	SpecializationTags []string `json:"specializationTags"`
	SubjectTags        []string `json:"subjectTags"`
	EmploymentType     string   `json:"employmentType"`
	Status             string   `json:"status"`
	CountryCode        string   `json:"countryCode"`
	CourseTypes        []string `json:"courseTypes"`
	LessonFormats      []string `json:"lessonFormats"`
	TLID               string   `json:"tlId"`
	TLName             string   `json:"tlName"`
	TelegramHandle     string   `json:"telegramHandle"`
	TelegramEnabled    bool     `json:"telegramEnabled"`
	TelegramLinkCode   string   `json:"telegramLinkCode,omitempty"`
	DensityPref        string   `json:"densityPreference"`
	LocalePref         string   `json:"localePreference"`
	AvailableRoles     []string `json:"availableRoles"`
	ProfileIncomplete  bool     `json:"profileIncomplete"`
	UpdatedAt          string   `json:"updatedAt"`
}

type Group struct {
	GroupID          string `json:"groupId"`
	TutorID          string `json:"tutorId"`
	Title            string `json:"title"`
	CourseCode       string `json:"courseCode"`
	CourseName       string `json:"courseName"`
	Format           string `json:"format"`
	ScheduleSummary  string `json:"scheduleSummary"`
	StartDate        string `json:"startDate"`
	StudentsCount    int    `json:"studentsCount"`
	AvgAttendancePct int    `json:"avgAttendancePct"`
	AvgProgressPct   int    `json:"avgProgressPct"`
	LMSURL           string `json:"lmsUrl"`
	IsArchived       bool   `json:"isArchived"`
}

type Student struct {
	StudentID             string `json:"studentId"`
	TutorID               string `json:"tutorId"`
	FullName              string `json:"fullName"`
	GroupID               string `json:"groupId"`
	GroupTitle            string `json:"groupTitle"`
	GroupFormat           string `json:"groupFormat"`
	CourseCode            string `json:"courseCode"`
	CourseName            string `json:"courseName"`
	AttendancePct         int    `json:"attendancePct"`
	ProgressPct           int    `json:"progressPct"`
	LessonsAttended       int    `json:"lessonsAttended"`
	LessonsTotal          int    `json:"lessonsTotal"`
	LessonsRemaining      int    `json:"lessonsRemaining"`
	TotalLessonsPurchased int    `json:"totalLessonsPurchased"`
	LastLessonAt          string `json:"lastLessonAt"`
	ParentName            string `json:"parentName"`
	ParentPhone           string `json:"parentPhone"`
	LMSURL                string `json:"lmsUrl"`
	DiagnosticPct         int    `json:"diagnosticPct"`
	DiagnosticGoal        string `json:"diagnosticGoal"`
}

type LessonStudent struct {
	StudentID        string `json:"studentId"`
	StudentName      string `json:"studentName"`
	AttendanceStatus string `json:"attendanceStatus"`
	ParentPhone      string `json:"parentPhone,omitempty"`
	ReportFilled     bool   `json:"reportFilled"`
	ReportID         string `json:"reportId,omitempty"`
}

type Lesson struct {
	LessonID           string          `json:"lessonId"`
	TutorID            string          `json:"tutorId"`
	GroupID            string          `json:"groupId"`
	GroupTitle         string          `json:"groupTitle"`
	CourseCode         string          `json:"courseCode"`
	Format             string          `json:"format"`
	LessonTitle        string          `json:"lessonTitle"`
	LessonNumber       int             `json:"lessonNumber"`
	CourseLessonNumber int             `json:"courseLessonNumber"`
	StartsAt           string          `json:"startsAt"`
	EndsAt             string          `json:"endsAt"`
	DurationMinutes    int             `json:"durationMinutes"`
	MeetingURL         string          `json:"meetingUrl"`
	PrepURL            string          `json:"prepUrl"`
	LMSURL             string          `json:"lmsUrl"`
	IsCompleted        bool            `json:"isCompleted"`
	Students           []LessonStudent `json:"students,omitempty"`
}

type Task struct {
	TaskID             string `json:"taskId"`
	TutorID            string `json:"tutorId"`
	Title              string `json:"title"`
	Description        string `json:"description"`
	TaskType           string `json:"taskType"`
	Status             string `json:"status"`
	Priority           string `json:"priority"`
	DueAt              string `json:"dueAt"`
	AssignedAt         string `json:"assignedAt"`
	CompletedAt        string `json:"completedAt,omitempty"`
	ReasonNote         string `json:"reasonNote,omitempty"`
	MetricCode         string `json:"metricCode,omitempty"`
	WeekStart          string `json:"weekStart,omitempty"`
	RelatedGroupID     string `json:"relatedGroupId,omitempty"`
	RelatedGroupTitle  string `json:"relatedGroupTitle,omitempty"`
	RelatedStudentID   string `json:"relatedStudentId,omitempty"`
	RelatedStudentName string `json:"relatedStudentName,omitempty"`
	ThreadID           string `json:"threadId,omitempty"`
}

type UrgentSubRequest struct {
	RequestID      string   `json:"requestId"`
	TutorID        string   `json:"tutorId"`
	TutorName      string   `json:"tutorName"`
	LMSLessonID    string   `json:"lmsLessonId"`
	GroupTitle     string   `json:"groupTitle"`
	CourseTitle    string   `json:"courseTitle"`
	LessonStartsAt string   `json:"lessonStartsAt"`
	Reason         string   `json:"reason"`
	ProofURLs      []string `json:"proofUrls"`
	Status         string   `json:"status"`
	CreatedAt      string   `json:"createdAt"`
}

type RegularFeedbackReport struct {
	ReportID      string `json:"reportId"`
	TutorID       string `json:"tutorId"`
	WeekStart     string `json:"weekStart"`
	LMSLessonID   string `json:"lmsLessonId"`
	LMSStudentID  string `json:"lmsStudentId"`
	StudentName   string `json:"studentName"`
	ParentPhone   string `json:"parentPhone"`
	GroupTitle    string `json:"groupTitle"`
	LessonNumber  int    `json:"lessonNumber"`
	CourseCode    string `json:"courseCode"`
	CourseName    string `json:"courseName"`
	MonthNo       int    `json:"monthNo"`
	Attendance    string `json:"attendance"`
	Engagement    string `json:"engagement"`
	Tasks         string `json:"tasks"`
	Custom1       string `json:"custom1"`
	Custom2       string `json:"custom2"`
	Stars         int    `json:"stars"`
	ProjectURL    string `json:"projectUrl"`
	AssembledText string `json:"assembledText"`
	UpdatedAt     string `json:"updatedAt"`
}

type MTFeedbackEntry struct {
	EntryID         string `json:"entryId"`
	TutorID         string `json:"tutorId"`
	WeekStart       string `json:"weekStart"`
	LMSLessonID     string `json:"lmsLessonId"`
	LMSStudentID    string `json:"lmsStudentId"`
	StudentName     string `json:"studentName"`
	LessonNumber    int    `json:"lessonNumber"`
	LessonTopic     string `json:"lessonTopic"`
	RecordingURL    string `json:"recordingUrl"`
	Understanding   int    `json:"understanding"`
	Independence    int    `json:"independence"`
	Accuracy        int    `json:"accuracy"`
	Engagement      string `json:"engagement"`
	WhatWorked      string `json:"whatWorked"`
	DevelopmentArea string `json:"developmentArea"`
	Cause           string `json:"cause"`
	NextSteps       string `json:"nextSteps"`
	TutorComment    string `json:"tutorComment"`
	IsOnTime        bool   `json:"isOnTime"`
	Source          string `json:"source"`
	CreatedAt       string `json:"createdAt"`
}

type AvailabilitySlot struct {
	Weekday   int    `json:"weekday"`
	StartMin  int    `json:"startMin"`
	Status    string `json:"status"`
	SheetCell string `json:"sheetCell"`
}

type RescheduleRequest struct {
	RequestID        string   `json:"requestId"`
	TutorID          string   `json:"tutorId"`
	TutorName        string   `json:"tutorName"`
	LMSLessonID      string   `json:"lmsLessonId"`
	LMSGroupID       string   `json:"lmsGroupId"`
	GroupName        string   `json:"groupName"`
	GroupFormat      string   `json:"groupFormat"`
	LessonNumber     int      `json:"lessonNumber"`
	ReqType          string   `json:"type"` // "otr" | "permanent"
	OriginalStartsAt string   `json:"originalStartsAt"`
	NewStartsAt      string   `json:"newStartsAt"`
	Note             string   `json:"note"`
	ProofURLs        []string `json:"proofUrls"`
	Status           string   `json:"status"`
	TLNote           string   `json:"tlNote,omitempty"`
	CreatedAt        string   `json:"createdAt"`
	ReviewedAt       string   `json:"reviewedAt,omitempty"`
}

type WorkloadRequest struct {
	RequestID string `json:"requestId"`
	TutorID   string `json:"tutorId"`
	TutorName string `json:"tutorName"`
	ReqType   string `json:"reqType"`
	Reason    string `json:"reason"`
	Status    string `json:"status"`
	CreatedAt string `json:"createdAt"`
}

type TLReport struct {
	ReportID    string `json:"reportId"`
	TutorID     string `json:"tutorId"`
	TutorName   string `json:"tutorName"`
	ReportType  string `json:"reportType"`
	GroupID     string `json:"groupId"`
	GroupTitle  string `json:"groupTitle"`
	StudentID   string `json:"studentId"`
	StudentName string `json:"studentName"`
	Urgency     string `json:"urgency"`
	Message     string `json:"message"`
	Status      string `json:"status"`
	TLReply     string `json:"tlReply,omitempty"`
	CreatedAt   string `json:"createdAt"`
}

type AssessmentQuestion struct {
	ID       string   `json:"id"`
	Program  string   `json:"program"` // "all" | "coding" | "regular_math" | "mt"
	Section  string   `json:"section"`
	Kind     string   `json:"kind"` // "single" | "checkbox" | "scale" | "paragraph" | "short"
	Prompt   string   `json:"prompt"`
	Options  []string `json:"options,omitempty"`
	Required bool     `json:"required"`
}

type AssessmentCycle struct {
	CycleCode   string                `json:"cycleCode"`
	Title       string                `json:"title"`
	Channel     string                `json:"channel"`
	DeadlineAt  string                `json:"deadlineAt"`
	IsOpen      bool                  `json:"isOpen"`
	Questions   []AssessmentQuestion  `json:"questions"`
	Submission  *AssessmentSubmission `json:"submission,omitempty"`
}

type AssessmentSubmission struct {
	SubmissionID string         `json:"submissionId"`
	CycleCode    string         `json:"cycleCode"`
	Programs     []string       `json:"programs"`
	Answers      map[string]any `json:"answers"`
	ScorePct     int            `json:"scorePct"`
	SubmittedAt  string         `json:"submittedAt"`
}

type AssessmentHistoryItem struct {
	ID         int64  `json:"id"`
	CycleCode  string `json:"cycleCode"`
	CycleLabel string `json:"cycleLabel"`
	MonthKey   string `json:"monthKey"`
	Program    string `json:"program"`
	ScorePct   int    `json:"scorePct"`
	MaxPct     int    `json:"maxPct"`
	Status     string `json:"status"`
	GradedAt   string `json:"gradedAt"`
}

type KBArticle struct {
	ArticleID    string `json:"articleId"`
	Category     string `json:"category"`
	Locale       string `json:"locale"`
	Title        string `json:"title"`
	Summary      string `json:"summary"`
	ContentMD    string `json:"contentMd"`
	HelpfulCount int    `json:"helpfulCount"`
}

type Thread struct {
	ThreadID          string    `json:"threadId"`
	TutorID           string    `json:"tutorId"`
	Subject           string    `json:"subject"`
	RequestType       string    `json:"requestType"`
	TargetTeam        string    `json:"targetTeam"`
	Priority          string    `json:"priority"`
	Origin            string    `json:"origin"`
	RelatedGroupRef   string    `json:"relatedGroupRef"`
	RelatedStudentRef string    `json:"relatedStudentRef"`
	UnreadTutor       int       `json:"unreadTutor"`
	UnreadTL          int       `json:"unreadTl"`
	Status            string    `json:"status"`
	LastMessage       string    `json:"lastMessage"`
	UpdatedAt         string    `json:"updatedAt"`
	CreatedAt         string    `json:"createdAt"`
	Messages          []Message `json:"messages,omitempty"`
}

type Message struct {
	MessageID     string `json:"messageId"`
	ThreadID      string `json:"threadId"`
	SenderRole    string `json:"senderRole"`
	SenderName    string `json:"senderName"`
	Kind          string `json:"kind"`
	Body          string `json:"body"`
	AttachmentURL string `json:"attachmentUrl,omitempty"`
	CreatedAt     string `json:"createdAt"`
}

type SavedView struct {
	ViewID      string `json:"viewId"`
	PageKey     string `json:"pageKey"`
	Name        string `json:"name"`
	QueryString string `json:"queryString"`
	CreatedAt   string `json:"createdAt"`
}

type PortalEvent struct {
	EventID     string `json:"eventId"`
	Title       string `json:"title"`
	Description string `json:"description"`
	StartsAt    string `json:"startsAt"`
	EndsAt      string `json:"endsAt"`
	MeetingURL  string `json:"meetingUrl"`
	CreatedBy   string `json:"createdBy"`
}

type SurveyTarget struct {
	SurveyID         string            `json:"surveyId"`
	TutorID          string            `json:"tutorId"`
	StudentID        string            `json:"studentId"`
	StudentName      string            `json:"studentName"`
	GroupTitle       string            `json:"groupTitle"`
	CourseName       string            `json:"courseName"`
	LessonsRemaining int               `json:"lessonsRemaining"`
	DeadlineAt       string            `json:"deadlineAt"`
	Status           string            `json:"status"` // "pending" | "completed"
	CreatedAt        string            `json:"createdAt"`
	Submission       *SurveySubmission `json:"submission,omitempty"`
}

type SurveySubmission struct {
	SubmissionID        string `json:"submissionId"`
	SurveyID            string `json:"surveyId"`
	TutorID             string `json:"tutorId"`
	StudentID           string `json:"studentId"`
	NominationBadge     string `json:"nominationBadge"`
	ObservationNotes    string `json:"observationNotes"`
	ComprehensionScore  int    `json:"comprehensionScore"`
	IndependenceScore   int    `json:"independenceScore"`
	FocusScore          int    `json:"focusScore"`
	StabilityScore      int    `json:"stabilityScore"`
	MotivationScore     int    `json:"motivationScore"`
	PrimaryMotivator    string `json:"primaryMotivator"`
	ParentFeedbackNotes string `json:"parentFeedbackNotes"`
	SubmittedAt         string `json:"submittedAt"`
}

type RoutinesSummary struct {
	FeedbackRoutine struct {
		Filled int    `json:"filled"`
		Total  int    `json:"total"`
		Pct    int    `json:"pct"`
		Label  string `json:"label"`
	} `json:"feedbackRoutine"`
	GraduationRoutine struct {
		Pending int    `json:"pending"`
		Label   string `json:"label"`
	} `json:"graduationRoutine"`
	AttendanceRoutine struct {
		Pct   int    `json:"pct"`
		Done  int    `json:"done"`
		Total int    `json:"total"`
		Label string `json:"label"`
	} `json:"attendanceRoutine"`
	AssessmentRoutine struct {
		CycleCode string `json:"cycleCode"`
		Status    string `json:"status"`
		Label     string `json:"label"`
	} `json:"assessmentRoutine"`
	SurveyRoutine struct {
		PendingCount int    `json:"pendingCount"`
		DeadlineAt   string `json:"deadlineAt"`
		Label        string `json:"label"`
	} `json:"surveyRoutine"`
}
