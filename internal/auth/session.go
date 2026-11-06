package auth

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"
)

const CookieName = "ak_session"

type SessionClaims struct {
	UserID         string   `json:"userId"`
	TutorID        string   `json:"tutorId"`
	LMSUserID      int64    `json:"lmsUserId"`
	Login          string   `json:"login"`
	FullName       string   `json:"fullName"`
	ActiveRole     string   `json:"activeRole"`     // "tutor" | "team_lead"
	AvailableRoles []string `json:"availableRoles"` // ["tutor", "team_lead"]
	IssuedAt       int64    `json:"iat"`
	ExpiresAt      int64    `json:"exp"`
}

type Manager struct {
	secret []byte
}

func NewManager(secret string) *Manager {
	return &Manager{secret: []byte(secret)}
}

func (m *Manager) Sign(claims SessionClaims) (string, error) {
	now := time.Now().Unix()
	if claims.IssuedAt == 0 {
		claims.IssuedAt = now
	}
	if claims.ExpiresAt == 0 {
		claims.ExpiresAt = now + int64((7 * 24 * time.Hour).Seconds())
	}
	raw, err := json.Marshal(claims)
	if err != nil {
		return "", err
	}
	payload := base64.RawURLEncoding.EncodeToString(raw)
	sig := m.computeMAC(payload)
	return payload + "." + sig, nil
}

func (m *Manager) Verify(token string) (SessionClaims, error) {
	payload, sig, ok := strings.Cut(token, ".")
	if !ok || payload == "" || sig == "" {
		return SessionClaims{}, errors.New("malformed session token")
	}
	expected := m.computeMAC(payload)
	if !hmac.Equal([]byte(sig), []byte(expected)) {
		return SessionClaims{}, errors.New("invalid session signature")
	}
	raw, err := base64.RawURLEncoding.DecodeString(payload)
	if err != nil {
		return SessionClaims{}, err
	}
	var claims SessionClaims
	if err := json.Unmarshal(raw, &claims); err != nil {
		return SessionClaims{}, err
	}
	if time.Now().Unix() > claims.ExpiresAt {
		return SessionClaims{}, errors.New("session expired")
	}
	return claims, nil
}

func (m *Manager) SetCookie(w http.ResponseWriter, claims SessionClaims, secure bool) error {
	tok, err := m.Sign(claims)
	if err != nil {
		return err
	}
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
	})
	return nil
}

func (m *Manager) ClearCookie(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     CookieName,
		Value:    "",
		Path:     "/",
		HttpOnly: true,
		SameSite: http.SameSiteLaxMode,
		MaxAge:   -1,
	})
}

func (m *Manager) FromRequest(r *http.Request) (SessionClaims, error) {
	c, err := r.Cookie(CookieName)
	if err == nil && c.Value != "" {
		return m.Verify(c.Value)
	}
	if authz := r.Header.Get("Authorization"); strings.HasPrefix(authz, "Bearer ") {
		return m.Verify(strings.TrimPrefix(authz, "Bearer "))
	}
	return SessionClaims{}, errors.New("unauthenticated")
}

func (m *Manager) computeMAC(payload string) string {
	mac := hmac.New(sha256.New, m.secret)
	_, _ = mac.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}
