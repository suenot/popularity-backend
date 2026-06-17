// Package parsers wires per-platform Parser implementations into a registry
// keyed by shared.Platform.
package parsers

import (
	shared "github.com/suenot/socials-auto"

	discord "github.com/suenot/discord-auto"
	facebook "github.com/suenot/facebook-auto"
	githubp "github.com/suenot/github-auto"
	githubrepo "github.com/suenot/github-repo-auto"
	habr "github.com/suenot/habr-auto"
	instagram "github.com/suenot/instagram-auto"
	linkedin "github.com/suenot/linkedin-auto"
	mmauth "github.com/suenot/marketmaker-auth-auto"
	reddit "github.com/suenot/reddit-auto"
	smartlab "github.com/suenot/smartlab-auto"
	stackoverflow "github.com/suenot/stackoverflow-auto"
	tbankpulse "github.com/suenot/tbank-pulse-auto"
	telegram "github.com/suenot/telegram-auto"
	x "github.com/suenot/x-auto"
	youtube "github.com/suenot/youtube-auto"

	"github.com/suenot/popularity-backend/internal/config"
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
		APIKey: cfg.YouTubeAPIKey, // kept for back-compat; new parser scrapes HTML.
	})

	r[shared.PlatformX] = x.New(x.Config{
		BearerToken: cfg.XBearerToken,
		CamoufoxURL: cfg.CamoufoxURL,
	})

	r[shared.PlatformTelegram] = telegram.New(telegram.Config{
		Credential:  cfg.TelegramCredential,
		CamoufoxURL: cfg.CamoufoxURL,
	})

	r[shared.PlatformFacebook] = facebook.New(facebook.Config{
		AccessToken: cfg.FacebookAccessToken,
		CamoufoxURL: cfg.CamoufoxURL,
	})

	r[shared.PlatformInstagram] = instagram.New(instagram.Config{
		CamoufoxURL: cfg.CamoufoxURL,
	})

	r[shared.PlatformLinkedIn] = linkedin.New(linkedin.Config{
		LIATCookie:  cfg.LinkedInLIAT,
		JSESSIONID:  cfg.LinkedInJSESSIONID,
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

	r[shared.PlatformReddit] = reddit.New(reddit.Config{
		BearerToken: cfg.RedditBearerToken,
	})

	r[shared.PlatformGitHub] = githubp.New(githubp.Config{
		Token: cfg.GitHubToken,
	})

	r[shared.PlatformGitHubRepo] = githubrepo.New(githubrepo.Config{
		Token: cfg.GitHubToken,
	})

	r[shared.PlatformDiscord] = discord.New(discord.Config{
		Token: cfg.DiscordBotToken,
	})

	r[shared.PlatformMarketmakerAuth] = mmauth.New(mmauth.Config{})

	return r
}

// Get returns the parser for p, or nil if no parser is registered for it.
func (r Registry) Get(p shared.Platform) shared.Parser {
	return r[p]
}
