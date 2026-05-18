// Package parsers wires per-platform Parser implementations into a registry
// keyed by shared.Platform.
package parsers

import (
	shared "github.com/suenot/w-popularity-shared"

	facebook "github.com/suenot/w-popularity-parser-facebook"
	habr "github.com/suenot/w-popularity-parser-habr"
	instagram "github.com/suenot/w-popularity-parser-instagram"
	linkedin "github.com/suenot/w-popularity-parser-linkedin"
	smartlab "github.com/suenot/w-popularity-parser-smartlab"
	stackoverflow "github.com/suenot/w-popularity-parser-stackoverflow"
	tbankpulse "github.com/suenot/w-popularity-parser-tbank-pulse"
	telegram "github.com/suenot/w-popularity-parser-telegram"
	x "github.com/suenot/w-popularity-parser-x"
	youtube "github.com/suenot/w-popularity-parser-youtube"

	"github.com/suenot/w-popularity-backend/internal/config"
)

// Registry maps a platform to its configured Parser. Platforms without any
// credentials are still present so the API can report ErrNotImplemented /
// ErrAuth consistently rather than 404.
type Registry map[shared.Platform]shared.Parser

// Build constructs the registry from runtime config.
func Build(cfg config.Config) Registry {
	r := Registry{}

	r[shared.PlatformYouTube] = youtube.New(youtube.Config{APIKey: cfg.YouTubeAPIKey})
	r[shared.PlatformX] = x.New(x.Config{Credential: cfg.XCredential})
	r[shared.PlatformTelegram] = telegram.New(telegram.Config{Credential: cfg.TelegramCredential})
	r[shared.PlatformFacebook] = facebook.New(facebook.Config{Credential: cfg.FacebookCredential})
	r[shared.PlatformInstagram] = instagram.New(instagram.Config{Credential: cfg.InstagramCredential})
	r[shared.PlatformLinkedIn] = linkedin.New(linkedin.Config{Credential: cfg.LinkedInCredential})
	r[shared.PlatformHabr] = habr.New(habr.Config{Credential: cfg.HabrCredential})
	r[shared.PlatformStackOverflow] = stackoverflow.New(stackoverflow.Config{Credential: cfg.StackOverflowCredential})
	r[shared.PlatformTBankPulse] = tbankpulse.New(tbankpulse.Config{Credential: cfg.TBankPulseCredential})
	r[shared.PlatformSmartLab] = smartlab.New(smartlab.Config{Credential: cfg.SmartLabCredential})

	return r
}

// Get returns the parser for p, or nil if no parser is registered for it.
func (r Registry) Get(p shared.Platform) shared.Parser {
	return r[p]
}
