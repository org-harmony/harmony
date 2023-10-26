// Package auth manages authentication in HARMONY.
package auth

import (
	"errors"
	"fmt"
	"github.com/org-harmony/harmony/src/core/persistence"
	"golang.org/x/oauth2"
	"net/http"
	"time"
)

import (
	"context"
)

var (
	ErrInvalidOAuthState  = errors.New("invalid oauth state")
	ErrCodeExchangeFailed = errors.New("code exchange failed")
)

type Cfg struct {
	Providers    map[string]*ProviderCfg `toml:"provider"` // Providers contains a list of OAuth2 providers.
	EnableOAuth2 bool                    `toml:"enable_oauth2"`
}

// ProviderCfg is the config for an OAuth2 provider.
type ProviderCfg struct {
	Name           string   `toml:"name" validate:"required"`
	DisplayName    string   `toml:"display_name" validate:"required"`
	AuthorizeURI   string   `toml:"authorize_uri" validate:"required"`
	AccessTokenURI string   `toml:"access_token_uri" validate:"required"`
	UserinfoURI    string   `toml:"userinfo_uri"`
	ClientID       string   `toml:"client_id" validate:"required"`
	ClientSecret   string   `toml:"client_secret" validate:"required"`
	Scopes         []string `toml:"scopes" validate:"required"`
}

// LoginFunc is the callback function for the OAuthLogin function it is responsible for creating the user session.
type LoginFunc[P, M any] func(context.Context, *oauth2.Token, *ProviderCfg) (*persistence.Session[P, M], error)

// OAuthLogin handles the OAuth2 login process, including state and code verification.
// The login callback is responsible for creating the user session.
//
// Login happens through a callback because the user session is not part of the auth package rather it is domain specific.
func OAuthLogin[P, M any](ctx context.Context, state, code string, provider *ProviderCfg, login LoginFunc[P, M]) (*persistence.Session[P, M], error) {
	oAuthCfg := OAuthCfgFromProviderCfg(provider, "") // empty redirect URL because it is not used in this function

	token, err := OAuthVerify(ctx, code, state, oAuthCfg)
	if err != nil {
		return nil, err
	}

	return login(ctx, token, provider)
}

// OAuthVerify verifies the OAuth2 state and exchanges the code for a token.
func OAuthVerify(ctx context.Context, code string, state string, cfg *oauth2.Config) (*oauth2.Token, error) {
	if state != "state" { // TODO add checks for dynamic state
		return nil, ErrInvalidOAuthState
	}

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("%w: %s", ErrCodeExchangeFailed, err.Error())
	}

	return token, nil
}

// OAuthCfgFromProviderCfg returns the oauth2.Config for the given provider and redirect URL config.
func OAuthCfgFromProviderCfg(p *ProviderCfg, redirectURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  p.AuthorizeURI,
			TokenURL: p.AccessTokenURI,
		},
		Scopes:      p.Scopes,
		RedirectURL: redirectURL,
	}
}

// SetSession sets the session cookie on the response.
// The session id is used as the cookie value.
// The cookie expires at the same time as the session.
func SetSession[P, M any](w http.ResponseWriter, name string, session *persistence.Session[P, M]) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    session.ID.String(),
		Expires:  session.ExpiresAt,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	})
}

// ClearSession clears the session cookie on the response.
func ClearSession(w http.ResponseWriter, name string) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    "",
		Expires:  time.Now(),
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	})
}
