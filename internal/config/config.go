// Package config loads runtime configuration from environment variables.
//
// Defaults are inlined here; missing vars never panic on construction, but
// callers must validate required fields (DATABASE_URL, AUTH_JWKS_URL) for
// the modes they enable.
package config

import (
	"os"
	"strconv"
	"strings"
)

// Mode controls which entrypoints the binary runs.
type Mode string

const (
	ModeAPI       Mode = "api"
	ModeScheduler Mode = "scheduler"
	ModeBoth      Mode = "both"
)

// Config is the resolved environment for one process.
type Config struct {
	Mode Mode

	DatabaseURL string
	BackendPort string

	AuthJWKSURL     string
	AuthIssuer      string
	AuthServiceName string

	FrontendURL string

	FetchCron    string
	FetchWorkers int

	// Platform credentials. Empty strings mean the parser is inactive in
	// the registry (except YouTube which simply returns ErrAuth).
	YouTubeAPIKey      string
	XCredential        string
	TelegramCredential string
	FacebookCredential string
	InstagramCredential string
	LinkedInCredential string
	HabrCredential     string
	StackOverflowCredential string
	TBankPulseCredential string
	SmartLabCredential string
}

// FromEnv reads configuration from the process environment.
func FromEnv() Config {
	c := Config{
		Mode:            Mode(getEnv("MODE", string(ModeBoth))),
		DatabaseURL:     os.Getenv("DATABASE_URL"),
		BackendPort:     getEnv("BACKEND_PORT", "8080"),
		AuthJWKSURL:     getEnv("AUTH_JWKS_URL", "https://auth.marketmaker.cc/.well-known/jwks.json"),
		AuthIssuer:      getEnv("AUTH_ISSUER", "auth.marketmaker.cc"),
		AuthServiceName: getEnv("AUTH_SERVICE_NAME", "popularity"),
		FrontendURL:     getEnv("NEXT_PUBLIC_FRONTEND_URL", "*"),
		FetchCron:       getEnv("FETCH_CRON", "0 3 * * *"),
		FetchWorkers:    getEnvInt("FETCH_WORKERS", 4),

		YouTubeAPIKey:           os.Getenv("YOUTUBE_API_KEY"),
		XCredential:             os.Getenv("X_CREDENTIAL"),
		TelegramCredential:      os.Getenv("TELEGRAM_CREDENTIAL"),
		FacebookCredential:      os.Getenv("FACEBOOK_CREDENTIAL"),
		InstagramCredential:     os.Getenv("INSTAGRAM_CREDENTIAL"),
		LinkedInCredential:      os.Getenv("LINKEDIN_CREDENTIAL"),
		HabrCredential:          os.Getenv("HABR_CREDENTIAL"),
		StackOverflowCredential: os.Getenv("STACKOVERFLOW_CREDENTIAL"),
		TBankPulseCredential:    os.Getenv("TBANK_PULSE_CREDENTIAL"),
		SmartLabCredential:      os.Getenv("SMARTLAB_CREDENTIAL"),
	}
	c.Mode = Mode(strings.ToLower(string(c.Mode)))
	return c
}

func getEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func getEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}
