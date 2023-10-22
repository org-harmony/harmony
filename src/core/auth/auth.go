// Package auth provides authentication details and logic for HARMONY.
// Auth is a part of the core package as it provides user authentication for all domains.
package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/web"
	"golang.org/x/oauth2"
	"net/http"
	"time"
)

// Cfg is the config for the auth package.
type Cfg struct {
	// Providers contains a list of OAuth2 providers.
	Providers    map[string]ProviderCfg `toml:"provider"`
	EnableOAuth2 bool                   `toml:"enable_oauth2"`
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

func OAuthLogin[P, M any](
	request *http.Request,
	providers map[string]ProviderCfg,
	baseURL string,
	login func(context.Context, *oauth2.Token, *ProviderCfg) (*persistence.Session[P, M], error),
) (*persistence.Session[P, M], error) {
	ctx := request.Context()
	name := web.URLParam(request, "provider")
	state := request.FormValue("state")
	code := request.FormValue("code")

	oAuthCfg, provider, err := OAuthConfig(name, providers, baseURL)
	if err != nil {
		return nil, err
	}

	token, err := OAuthVerify(ctx, code, state, oAuthCfg)
	if err != nil {
		return nil, err
	}

	return login(ctx, token, provider)
}

// OAuthVerify verifies the oauth login by verifying the state and exchanging the code for a token.
func OAuthVerify(ctx context.Context, code string, state string, cfg *oauth2.Config) (*oauth2.Token, error) {
	if state != "state" {
		return nil, errors.New("invalid oauth state")
	}

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}

	return token, nil
}

// OAuthConfig returns the OAuth2 config for the given provider name.
func OAuthConfig(name string, providers map[string]ProviderCfg, baseURL string) (*oauth2.Config, *ProviderCfg, error) {
	p, ok := providers[name]
	if !ok {
		return nil, nil, fmt.Errorf("auth: provider %s not found", name)
	}

	return OAuthCfgFromProviderCfg(p, baseURL), &p, nil
}

// OAuthCfgFromProviderCfg returns the OAuth2 config for the given provider config.
func OAuthCfgFromProviderCfg(p ProviderCfg, baseURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  p.AuthorizeURI,
			TokenURL: p.AccessTokenURI,
		},
		Scopes:      p.Scopes,
		RedirectURL: fmt.Sprintf("%s%s", baseURL, fmt.Sprintf("/auth/login/%s/success", p.Name)),
	}
}

// SetSession sets the user session cookie on the response.
// The session id is used as the cookie value.
// The session expires at the time of the session.
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

// ClearSession clears the user session cookie on the response.
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
