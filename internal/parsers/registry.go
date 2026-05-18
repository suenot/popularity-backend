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

// Registry maps a platform to its configured Parser. Every platform is wired
// so the scheduler can dispatch jobs uniformly; parsers may individually
// return ErrAuth / ErrNotImplemented based on their own configuration.
type Registry map[shared.Platform]shared.Parser

// Build constructs the registry from runtime config. Each parser receives
// only the fields its own Config struct supports — they are not symmetric.
func Build(cfg config.Config) Registry {
	r := Registry{}

	r[shared.PlatformYouTube] = youtube.New(youtube.Config{
		APIKey: cfg.YouTubeAPIKey,
		// yt-dlp is resolved via PATH inside the container.
	})

	r[shared.PlatformX] = x.New(x.Config{
		CamoufoxURL: cfg.CamoufoxURL,
		// NitterMirrors left empty → uses DefaultNitterMirrors. Operators
		// can override at runtime by editing the parser config layer.
	})

	r[shared.PlatformTelegram] = telegram.New(telegram.Config{
		Credential:  cfg.TelegramCredential,
		CamoufoxURL: cfg.CamoufoxURL,
	})

	r[shared.PlatformFacebook] = facebook.New(facebook.Config{
		Credential:  cfg.FacebookCredential,
		CamoufoxURL: cfg.CamoufoxURL,
	})

	r[shared.PlatformInstagram] = instagram.New(instagram.Config{
		CamoufoxURL: cfg.CamoufoxURL,
	})

	r[shared.PlatformLinkedIn] = linkedin.New(linkedin.Config{
		CamoufoxURL: cfg.CamoufoxURL,
	})

	r[shared.PlatformHabr] = habr.New(habr.Config{})

	r[shared.PlatformStackOverflow] = stackoverflow.New(stackoverflow.Config{
		AppKey: cfg.StackExchangeKey,
	})

	r[shared.PlatformTBankPulse] = tbankpulse.New(tbankpulse.Config{
		CamoufoxURL: cfg.CamoufoxURL,
	})

	r[shared.PlatformSmartLab] = smartlab.New(smartlab.Config{
		Credential:  cfg.SmartLabCredential,
		CamoufoxURL: cfg.CamoufoxURL,
	})

	return r
}

// Get returns the parser for p, or nil if no parser is registered for it.
func (r Registry) Get(p shared.Platform) shared.Parser {
	return r[p]
}
