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

func NewClient(baseURL string, enforceReadonly bool) *Client {
	jar, _ := cookiejar.New(nil)
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		httpClient: &http.Client{
			Jar:     jar,
			Timeout: 8 * time.Second,
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

// VerifyCredentials performs the audited 2-step LMS credential verification against lms.algonova.id
// (/api/v2/user/exists + /s/auth/api/e/user/auth) without touching any portal/teaching data.
func (c *Client) VerifyCredentials(ctx context.Context, login, password string) (UserProfile, error) {
	existsBody, _ := json.Marshal(map[string]string{"login": login})
	req1, err := http.NewRequestWithContext(ctx, http.MethodPost, c.baseURL+"/api/v2/user/exists", bytes.NewReader(existsBody))
	if err != nil {
		return UserProfile{}, err
	}
	req1.Header.Set("Content-Type", "application/json")
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
