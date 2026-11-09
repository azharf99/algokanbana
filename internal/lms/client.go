package lms

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/cookiejar"
	"strings"
	"time"
)

var ErrMutatingUpstreamBlocked = errors.New("safety guard: mutating HTTP request to upstream Algonova/NanoKanbana is strictly blocked in safe mode")

// SafeReadonlyTransport wraps an http.RoundTripper and blocks any non-read HTTP request
// (except the explicitly permitted LMS credential check endpoints during login).
type SafeReadonlyTransport struct {
	Base         http.RoundTripper
	EnforceGuard bool
}

func (t *SafeReadonlyTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	base := t.Base
	if base == nil {
		base = http.DefaultTransport
	}
	if t.EnforceGuard {
		method := strings.ToUpper(req.Method)
		if method != http.MethodGet && method != http.MethodHead && method != http.MethodOptions {
			path := req.URL.Path
			// Only read-only login verification endpoints on lms.algonova.id are permitted when user opts into LMS auth.
			allowedAuth := method == http.MethodPost && (path == "/api/v2/user/exists" || path == "/s/auth/api/e/user/auth")
			if !allowedAuth {
				return nil, fmt.Errorf("%w: %s %s", ErrMutatingUpstreamBlocked, method, req.URL.Redacted())
			}
		}
	}
	return base.RoundTrip(req)
}

type UserProfile struct {
	LMSUserID int64  `json:"lmsUserId"`
	Login     string `json:"login"`
	FullName  string `json:"fullName"`
	EmailMask string `json:"emailMask"`
}

type Client struct {
	baseURL    string
	httpClient *http.Client
}

func NewClient(baseURL string, enforceReadonly bool, timeout time.Duration) *Client {
	if timeout <= 0 {
		timeout = 30 * time.Second
	}
	jar, _ := cookiejar.New(nil)
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: timeout,
			Transport: &SafeReadonlyTransport{
				Base:         http.DefaultTransport,
				EnforceGuard: enforceReadonly,
			},
		},
	}
}

func (c *Client) HTTPClient() *http.Client {
	return c.httpClient
}

func (c *Client) setHeaders(req *http.Request) {
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/157.0.0.0 Safari/537.36")
	req.Header.Set("X-Requested-With", "XMLHttpRequest")
	req.Header.Set("Accept", "application/json, text/plain, */*")
}

// VerifyCredentials performs the audited 2-step LMS credential verification against lms.algonova.id
// (/api/v2/user/exists + /s/auth/api/e/user/auth) without touching any portal/teaching data.
func (c *Client) VerifyCredentials(ctx context.Context, login, password string) (UserProfile, error) {
	existsBody, _ := json.Marshal(map[string]string{"login": login})
	req1, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v2/user/exists", bytes.NewReader(existsBody))
	if err != nil {
		return UserProfile{}, err
	}
	req1.Header.Set("Content-Type", "application/json")
	c.setHeaders(req1)
	resp1, err := c.httpClient.Do(req1)
	if err != nil {
		return UserProfile{}, err
	}
	defer func() { _ = resp1.Body.Close() }()

	var existsResp struct {
		Status string `json:"status"`
		Data   struct {
			ID       int64  `json:"id"`
			FullName string `json:"fullName"`
			Email    string `json:"email"`
		} `json:"data"`
	}
	_ = json.NewDecoder(resp1.Body).Decode(&existsResp)

	authBody, _ := json.Marshal(map[string]string{"login": login, "password": password})
	req2, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/s/auth/api/e/user/auth", bytes.NewReader(authBody))
	if err != nil {
		return UserProfile{}, err
	}
	req2.Header.Set("Content-Type", "application/json")
	c.setHeaders(req2)
	resp2, err := c.httpClient.Do(req2)
	if err != nil {
		return UserProfile{}, err
	}
	defer func() { _ = resp2.Body.Close() }()
	if resp2.StatusCode >= 400 {
		return UserProfile{}, fmt.Errorf("LMS authentication rejected with HTTP %d", resp2.StatusCode)
	}

	fullName := strings.TrimSpace(existsResp.Data.FullName)
	if fullName == "" {
		fullName = login
	}
	return UserProfile{
		LMSUserID: existsResp.Data.ID,
		Login:     login,
		FullName:  fullName,
		EmailMask: existsResp.Data.Email,
	}, nil
}

type RawLMSGroup struct {
	ID        int64  `json:"id"`
	Title     string `json:"title"`
	Type      string `json:"type"`   // "intensive" | "regular"
	Status    string `json:"status"` // "active" | "inactive" | "destroyed"
	StartTime string `json:"startTime"`
}

func (c *Client) FetchTutorGroups(ctx context.Context) ([]RawLMSGroup, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/group/index?per-page=100", nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("LMS fetch groups failed with HTTP %d", resp.StatusCode)
	}

	var res struct {
		Status string `json:"status"`
		Data   struct {
			Items []RawLMSGroup `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Data.Items, nil
}

type RawTeacherDashboardGroup struct {
	ID                  int64                `json:"id"`
	GroupTitle          string               `json:"groupTitle"`
	ActiveStudentsCount int                  `json:"activeStudentsCount"`
	NextGroupLesson     *RawNextGroupLesson  `json:"nextGroupLesson"`
	LessonStats         []RawGroupLessonStat `json:"lessonStats"`
}

type RawNextGroupLesson struct {
	LessonID       int64  `json:"lessonId"`
	LessonTitle    string `json:"lessonTitle"`
	StartTime      string `json:"startTime"` // e.g. "2026-11-14 15:30:00"
	ModulePosition int    `json:"modulePosition"`
	LessonPosition int    `json:"lessonPosition"`
}

type RawGroupLessonStat struct {
	LessonID              int64   `json:"lessonId"`
	LessonTitle           string  `json:"lessonTitle"`
	GroupLessonID         int64   `json:"groupLessonId"`
	GroupLessonStartTime  string  `json:"groupLessonStartTime"`
	GroupLessonFinished   bool    `json:"groupLessonFinished"`
	ModulePosition        int     `json:"modulePosition"`
	ModuleTitle           string  `json:"moduleTitle"`
	LessonPosition        int     `json:"lessonPosition"`
	RegularTaskPercent    float64 `json:"regularTaskPercent"`
	BonusTaskPercent      float64 `json:"bonusTaskPercent"`
	AttendedStudentsCount int     `json:"attendedStudentsCount"`
	ActiveStudentsCount   int     `json:"activeStudentsCount"`
}

func (c *Client) FetchTeacherDashboardGroups(ctx context.Context) ([]RawTeacherDashboardGroup, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/api/v2/teacher-dashboard/group/index", nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("LMS teacher-dashboard failed with HTTP %d", resp.StatusCode)
	}

	var res struct {
		Status string `json:"status"`
		Data   struct {
			Items []RawTeacherDashboardGroup `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Data.Items, nil
}

type RawGroupStudent struct {
	ID           int64            `json:"id"`
	FirstName    string           `json:"firstName"`
	LastName     string           `json:"lastName"`
	FullName     string           `json:"fullName"`
	ParentName   string           `json:"parentName"`
	Email        string           `json:"email"`
	Phone        string           `json:"phone"`
	Wallet       *RawWallet       `json:"wallet"`
	ActiveGroups []RawActiveGroup `json:"activeGroups"`
}

type RawWallet struct {
	ID          int64  `json:"id"`
	Credits     int    `json:"credits"`
	CreditsPaid int    `json:"creditsPaid"`
	Amount      string `json:"amount"`
}

type RawActiveGroup struct {
	ID             int64           `json:"id"`
	GroupStudentID int64           `json:"groupStudentId"`
	Title          string          `json:"title"`
	StartTime      string          `json:"startTime"`
	GroupWallet    *RawGroupWallet `json:"groupWallet"`
}

type RawGroupWallet struct {
	ID          int64 `json:"id"`
	StudentID   int64 `json:"studentId"`
	GroupID     int64 `json:"groupId"`
	Credits     int   `json:"credits"`
	CreditsPaid int   `json:"creditsPaid"`
}

func (c *Client) FetchGroupStudents(ctx context.Context, groupID int64) ([]RawGroupStudent, error) {
	url := fmt.Sprintf("%s/api/v2/group/student/index?groupId=%d&expand=lastGroup,wallet,activeGroups,activeGroups.groupWallet", c.baseURL, groupID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("LMS group students failed with HTTP %d", resp.StatusCode)
	}

	var res struct {
		Status string `json:"status"`
		Data   struct {
			Items []RawGroupStudent `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Data.Items, nil
}

type RawGroupLesson struct {
	ID                         int64   `json:"id"`
	Number                     int     `json:"number"`
	IsArchived                 int     `json:"isArchived"`
	LessonID                   int64   `json:"lessonId"`
	GroupID                    int64   `json:"groupId"`
	Title                      string  `json:"title"`
	Content                    string  `json:"content"`
	Status                     int     `json:"status"`
	StartTime                  string  `json:"startTime"` // e.g. "2026-10-18T13:00:00+03:00"
	FinishedAt                 string  `json:"finishedAt"`
	RegularTaskScoreSumPercent float64 `json:"regularTaskScoreSumPercent"`
	IsAdditional               int     `json:"isAdditional"`
}

func (c *Client) FetchGroupLessons(ctx context.Context, groupID int64) ([]RawGroupLesson, error) {
	url := fmt.Sprintf("%s/api/v2/group/lesson/index?groupId=%d&per-page=100", c.baseURL, groupID)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("LMS group lessons failed with HTTP %d", resp.StatusCode)
	}

	var res struct {
		Status string `json:"status"`
		Data   struct {
			Items []RawGroupLesson `json:"items"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Data.Items, nil
}

type RawStudentAttendance struct {
	StudentID  int64               `json:"student_id"`
	Attendance []RawLessonPresence `json:"attendance"`
}

type RawLessonPresence struct {
	LessonID           int64  `json:"lesson_id"`
	LessonTitle        string `json:"lesson_title"`
	StartTimeFormatted string `json:"start_time_formatted"`
	Status             string `json:"status"` // "present" | "absent" | "inactive" | "excused"
}

func (c *Client) FetchGroupAttendance(ctx context.Context, groupID int64, studentIDs []int64) ([]RawStudentAttendance, error) {
	if len(studentIDs) == 0 {
		return nil, nil
	}
	var b strings.Builder
	fmt.Fprintf(&b, "%s/api/v1/stats/default/attendance?group=%d", c.baseURL, groupID)
	for _, id := range studentIDs {
		fmt.Fprintf(&b, "&students%%5B%%5D=%d", id)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, b.String(), nil)
	if err != nil {
		return nil, err
	}
	c.setHeaders(req)
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("LMS attendance failed with HTTP %d", resp.StatusCode)
	}

	var res struct {
		Status string                 `json:"status"`
		Data   []RawStudentAttendance `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&res); err != nil {
		return nil, err
	}
	return res.Data, nil
}
