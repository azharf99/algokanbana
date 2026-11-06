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
	SafeReadonlyUpstream bool
	LMSBaseURL           string
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

	return Config{
		AppMode:              mode,
		Port:                 getEnv("PORT", "8090"),
		DBPath:               getEnv("DB_PATH", "./data/algokanbana.db"),
		UploadDir:            getEnv("UPLOAD_DIR", "./data/uploads"),
		SessionSecret:        getEnv("SESSION_SECRET", "algokanbana-local-hmac-secret-key-2026"),
		SafeReadonlyUpstream: safeReadonly,
		LMSBaseURL:           strings.TrimRight(getEnv("LMS_BASE_URL", "https://lms.algonova.id"), "/"),
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
}
