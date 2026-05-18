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

	// Optional camoufox CDP endpoint used as scraping fallback by IG/FB/LinkedIn/etc.
	CamoufoxURL string
	// Optional Stack Exchange app key (boosts quota to 10k/day).
	StackExchangeKey string
	// X (Twitter) API v2 bearer token (X_BEARER_TOKEN env).
	XBearerToken string
	// Facebook Page Access Token (FACEBOOK_ACCESS_TOKEN env).
	FacebookAccessToken string
	// LinkedIn li_at session cookie (LINKEDIN_LI_AT env).
	LinkedInLIAT string
	// Optional LinkedIn JSESSIONID companion cookie.
	LinkedInJSESSIONID string
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
		CamoufoxURL:             os.Getenv("CAMOUFOX_URL"),
		StackExchangeKey:        os.Getenv("STACKEXCHANGE_KEY"),
		XBearerToken:            os.Getenv("X_BEARER_TOKEN"),
		FacebookAccessToken:     os.Getenv("FACEBOOK_ACCESS_TOKEN"),
		LinkedInLIAT:            os.Getenv("LINKEDIN_LI_AT"),
		LinkedInJSESSIONID:      os.Getenv("LINKEDIN_JSESSIONID"),
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
