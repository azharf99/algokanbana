package config

import (
	"bufio"
	"os"
	"strconv"
	"strings"
)

type AppMode string

const (
	ModeHybridSafe AppMode = "hybrid_safe"
	ModeSeedDemo   AppMode = "seed_demo"
	ModeProduction AppMode = "production"
)

type Config struct {
	AppMode              AppMode
	Port                 string
	DBPath               string
	UploadDir            string
	SessionSecret        string
	SessionCookieName    string
	CookieSecure         string // "true" | "false" | "auto"
	CookieSameSite       string // "Lax" | "Strict" | "None"
	SafeReadonlyUpstream bool
	LMSBaseURL           string
	UpstreamTimeoutMs    int
}

func Load() Config {
	loadDotEnv(".env")

	mode := AppMode(getEnv("APP_MODE", string(ModeHybridSafe)))
	switch mode {
	case ModeHybridSafe, ModeSeedDemo, ModeProduction:
	default:
		mode = ModeHybridSafe
	}

	safeReadonly := getEnvBool("SAFE_READONLY_UPSTREAM", true)
	if mode != ModeProduction {
		// Enforce read-only upstream safety in all non-production modes.
		safeReadonly = true
	}

	lmsURL := getEnv("LMS_BASE_URL", "")
	if lmsURL == "" {
		lmsURL = getEnv("UPSTREAM_BASE", "https://lms.algonova.id")
	}

	timeoutMs, err := strconv.Atoi(getEnv("UPSTREAM_TIMEOUT_MS", "30000"))
	if err != nil || timeoutMs < 5000 {
		timeoutMs = 30000
	}

	return Config{
		AppMode:              mode,
		Port:                 getEnv("PORT", "8080"),
		DBPath:               getEnv("DB_PATH", "./data/algokanbana.db"),
		UploadDir:            getEnv("UPLOAD_DIR", "./data/uploads"),
		SessionSecret:        getEnv("SESSION_SECRET", "algokanbana-local-hmac-secret-key-2026"),
		SessionCookieName:    getEnv("SESSION_COOKIE_NAME", "ak_session"),
		CookieSecure:         strings.ToLower(getEnv("COOKIE_SECURE", "auto")),
		CookieSameSite:       getEnv("COOKIE_SAME_SITE", "Lax"),
		SafeReadonlyUpstream: safeReadonly,
		LMSBaseURL:           strings.TrimRight(lmsURL, "/"),
		UpstreamTimeoutMs:    timeoutMs,
	}
}

func getEnv(key, fallback string) string {
	if v := strings.TrimSpace(os.Getenv(key)); v != "" {
		return v
	}
	return fallback
}

func getEnvBool(key string, fallback bool) bool {
	v := strings.TrimSpace(os.Getenv(key))
	if v == "" {
		return fallback
	}
	b, err := strconv.ParseBool(v)
	if err != nil {
		return fallback
	}
	return b
}

func loadDotEnv(path string) {
	f, err := os.Open(path)
	if err != nil {
		return
	}
	defer func() { _ = f.Close() }()

	scanner := bufio.NewScanner(f)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		k, v, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key := strings.TrimSpace(k)
		val := strings.Trim(strings.TrimSpace(v), `"'`)
		if key != "" && os.Getenv(key) == "" {
			_ = os.Setenv(key, val)
		}
	}

	if scanner.Err() != nil {
		return
	}

}
