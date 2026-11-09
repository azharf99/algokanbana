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
	secret     []byte
	cookieName string
	sameSite   http.SameSite
}

func NewManager(secret, cookieName, sameSite string) *Manager {
	if strings.TrimSpace(cookieName) == "" {
		cookieName = CookieName
	}
	ss := http.SameSiteLaxMode
	switch strings.ToLower(sameSite) {
	case "strict":
		ss = http.SameSiteStrictMode
	case "none":
		ss = http.SameSiteNoneMode
	}
	return &Manager{
		secret:     []byte(secret),
		cookieName: strings.TrimSpace(cookieName),
		sameSite:   ss,
	}
}

func (m *Manager) ResolveSecure(r *http.Request, pref string) bool {
	switch strings.ToLower(pref) {
	case "true":
		// Only set Secure if request is actual HTTPS or forwarded HTTPS.
		// If tutor is accessing via http://localhost:8083, setting Secure=true will cause
		// modern browsers to silently drop the cookie.
		isHTTPS := r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
		return isHTTPS
	case "false":
		return false
	default: // "auto"
		return r.TLS != nil || strings.EqualFold(r.Header.Get("X-Forwarded-Proto"), "https")
	}
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
		Name:     m.cookieName,
		Value:    tok,
		Path:     "/",
		HttpOnly: true,
		Secure:   secure,
		SameSite: m.sameSite,
		MaxAge:   int((7 * 24 * time.Hour).Seconds()),
	})
	return nil
}

func (m *Manager) ClearCookie(w http.ResponseWriter) {
	names := []string{m.cookieName}
	for _, n := range []string{"ak_session", "algokanbana_session"} {
		if n != m.cookieName {
			names = append(names, n)
		}
	}
	for _, name := range names {
		http.SetCookie(w, &http.Cookie{
			Name:     name,
			Value:    "",
			Path:     "/",
			HttpOnly: true,
			SameSite: m.sameSite,
			MaxAge:   -1,
		})
	}
}

func (m *Manager) FromRequest(r *http.Request) (SessionClaims, error) {
	// 1. Try configured cookie name
	if c, err := r.Cookie(m.cookieName); err == nil && c.Value != "" {
		return m.Verify(c.Value)
	}
	// 2. Fallback to alternative cookie names
	for _, name := range []string{"ak_session", "algokanbana_session"} {
		if name != m.cookieName {
			if c, err := r.Cookie(name); err == nil && c.Value != "" {
				return m.Verify(c.Value)
			}
		}
	}
	// 3. Try Authorization Bearer header
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
