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
	// ErrInvalidOAuthState is returned after an invalid OAuth2 state is detected.
	// This error might occur if a stale OAuth2 state is passed to the login success handler.
	ErrInvalidOAuthState = errors.New("invalid oauth state")
	// ErrCodeExchangeFailed is returned after an OAuth2 code exchange fails.
	// This error might occur if the OAuth2 code is invalid or expired.
	// The user should be redirected to the login page with an error message.
	ErrCodeExchangeFailed = errors.New("code exchange failed")
)

// Cfg is the config for the auth package. It contains necessary information about the OAuth2 providers.
type Cfg struct {
	Providers    map[string]*ProviderCfg `toml:"provider"` // Providers contains a list of OAuth2 providers.
	EnableOAuth2 bool                    `toml:"enable_oauth2"`
}

// ProviderCfg is the config for an OAuth2 provider.
// The config struct can be used to show the login page and handle the login callback based on various providers.
type ProviderCfg struct {
	Enabled        bool     `toml:"enabled"`
	Name           string   `toml:"name" hvalidate:"required"`
	DisplayName    string   `toml:"display_name" hvalidate:"required"`
	AuthorizeURI   string   `toml:"authorize_uri" hvalidate:"required"`
	AccessTokenURI string   `toml:"access_token_uri" hvalidate:"required"`
	UserinfoURI    string   `toml:"userinfo_uri"`
	ClientID       string   `toml:"client_id" hvalidate:"required"`
	ClientSecret   string   `toml:"client_secret" hvalidate:"required"`
	Scopes         []string `toml:"scopes" hvalidate:"required"`
}

// LoginFunc is the callback function for the OAuthLogin function it is responsible for creating the user session.
type LoginFunc[P, M any] func(context.Context, *oauth2.Token, *ProviderCfg) (*persistence.Session[P, M], error)

// OAuthLogin handles the OAuth2 login process, including state and code verification.
// The login callback is responsible for creating the user session.
//
// Login happens through a callback because the user session is not part of the auth package rather it is domain specific.
func OAuthLogin[P, M any](ctx context.Context, state, code, redirectURL string, provider *ProviderCfg, login LoginFunc[P, M]) (*persistence.Session[P, M], error) {
	oAuthCfg := OAuthCfgFromProviderCfg(provider, redirectURL) // some providers require a redirect URL to match (Google does while GitHub doesn't)

	token, err := OAuthVerify(ctx, code, state, oAuthCfg)
	if err != nil {
		return nil, err
	}

	return login(ctx, token, provider)
}

// OAuthVerify verifies the OAuth2 state and exchanges the code for a token.
func OAuthVerify(ctx context.Context, code, state string, cfg *oauth2.Config) (*oauth2.Token, error) {
	if state != "state" { // TODO add checks for dynamic state
		return nil, ErrInvalidOAuthState
	}

	cfg.Endpoint.AuthStyle = oauth2.AuthStyleInParams                            // google requires this
	authCodeOption := oauth2.SetAuthURLParam("grant_type", "authorization_code") // and this
	redirectOption := oauth2.SetAuthURLParam("redirect_uri", cfg.RedirectURL)    // and this
	token, err := cfg.Exchange(ctx, code, authCodeOption, redirectOption)

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
// The cookie expires at the same time as the session + 48 hours.
// This allows for db cleanup when using an expired session that is still sent to the backend.
func SetSession[P, M any](w http.ResponseWriter, name string, session *persistence.Session[P, M]) {
	http.SetCookie(w, &http.Cookie{
		Name:     name,
		Value:    session.ID.String(),
		Expires:  session.ExpiresAt.Add(48 * time.Hour), // will be validated by the backend but allows for db cleanup
		SameSite: http.SameSiteLaxMode,                  // must be lax for OAuth2, otherwise redirect will lead to weird states
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
