// Package auth provides authentication details and logic for HARMONY.
// Auth is a part of the core package as it provides user authentication for all domains.
package auth

import (
	"fmt"
	"github.com/org-harmony/harmony/core/config"
	"github.com/org-harmony/harmony/core/ctx"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/util"
	"github.com/org-harmony/harmony/core/web"
	"golang.org/x/oauth2"
	"io"
	"net/http"
)

const (
	Pkg                      = "sys.auth"
	OAuthLoginPattern        = "/auth/login/%s"
	OAuthLoginSuccessPattern = "/auth/login/%s/success"
)

// Cfg is the config for the auth package.
type Cfg struct {
	// Provider contains a list of OAuth2 providers.
	Provider     map[string]ProviderCfg `toml:"provider"`
	EnableOAuth2 bool                   `toml:"enable_oauth2"`
}

// ProviderCfg is the config for an OAuth2 provider.
type ProviderCfg struct {
	Name           string   `toml:"name" validate:"required"`
	AuthorizeURI   string   `toml:"authorize_uri" validate:"required"`
	AccessTokenURI string   `toml:"access_token_uri" validate:"required"`
	UserinfoURI    string   `toml:"userinfo_uri" validate:"required"`
	ClientID       string   `toml:"client_id" validate:"required"`
	ClientSecret   string   `toml:"client_secret" validate:"required"`
	Scopes         []string `toml:"scopes" validate:"required"`
}

func RegisterAuth(app ctx.App, ctx web.Context) {
	cfg := &Cfg{}
	util.Ok(config.C(cfg, config.From("auth"), config.Validate(app.Validator())))

	registerRoutes(cfg, app, ctx)
}

func registerRoutes(cfg *Cfg, app ctx.App, ctx web.Context) {
	lp := util.Unwrap(ctx.TemplaterStore().Templater(web.LandingPageTemplateName))
	errT := util.Unwrap(lp.Template("error", "error.go.html"))

	router := ctx.Router()
	router.Get("/auth/login", login(lp, app.Logger()))

	if !cfg.EnableOAuth2 {
		return
	}

	webCfg := ctx.Configuration()
	providers := cfg.Provider

	router.Get(fmt.Sprintf(OAuthLoginPattern, "{provider}"), func(w http.ResponseWriter, r *http.Request) {
		oAuthCfg, _, err := oAuthConfigFromProviderName(router.URLParam(r, "provider"), providers, webCfg.Server.BaseURL)
		if err != nil {
			util.Ok(errT.Execute(w, web.NewErrorTemplateData(
				r.Context(),
				"auth.error.invalid-provider",
			)))
			return
		}

		url := oAuthCfg.AuthCodeURL("state")

		http.Redirect(w, r, url, http.StatusTemporaryRedirect)
	})

	router.Get(fmt.Sprintf(OAuthLoginSuccessPattern, "{provider}"), func(w http.ResponseWriter, r *http.Request) {
		oAuthCfg, provider, err := oAuthConfigFromProviderName(router.URLParam(r, "provider"), providers, webCfg.Server.BaseURL)
		if err != nil {
			util.Ok(errT.Execute(w, web.NewErrorTemplateData(
				r.Context(),
				"auth.error.invalid-provider",
			)))
			return
		}

		state := r.FormValue("state")
		code := r.FormValue("code")

		if state != "state" {
			util.Ok(errT.Execute(w, web.NewErrorTemplateData(
				r.Context(),
				"auth.error.invalid-state",
			)))
			return
		}

		token, err := oAuthCfg.Exchange(r.Context(), code)
		if err != nil {
			util.Ok(errT.Execute(w, web.NewErrorTemplateData(
				r.Context(),
				"auth.error.invalid-token",
			)))
			return
		}

		req, err := http.NewRequest(http.MethodGet, provider.UserinfoURI, nil)
		if err != nil {
			util.Ok(errT.Execute(w, web.NewErrorTemplateData(
				r.Context(),
				"auth.error.invalid-userinfo-request",
			)))
			return
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

		c := http.DefaultClient
		userinfo, err := c.Do(req)
		if err != nil {
			util.Ok(errT.Execute(w, web.NewErrorTemplateData(
				r.Context(),
				"auth.error.invalid-userinfo-response",
			)))
			return
		}
		defer userinfo.Body.Close()

		contents, err := io.ReadAll(userinfo.Body)
		if err != nil {
			util.Ok(errT.Execute(w, web.NewErrorTemplateData(
				r.Context(),
				"auth.error.invalid-userinfo-response",
			)))
			return
		}

		fmt.Printf("userinfo: %s", contents)

		fmt.Printf("access token: %s, token: %+v", token.AccessToken, token)
	})
}

func login(lp web.Templater, l trace.Logger) http.HandlerFunc {
	tmpl := util.Unwrap(lp.Template("auth.login", "auth/login.go.html"))

	return func(w http.ResponseWriter, r *http.Request) {
		err := tmpl.Execute(w, nil)
		_ = web.MaybeIntErr(err, l, w, r)
	}
}

func oAuthConfigFromProviderName(name string, providers map[string]ProviderCfg, baseURL string) (*oauth2.Config, *ProviderCfg, error) {
	p, ok := providers[name]
	if !ok {
		return nil, nil, fmt.Errorf("auth: provider %s not found", name)
	}

	return oAuthCfgFromProviderCfg(p, baseURL), &p, nil
}

func oAuthCfgFromProviderCfg(p ProviderCfg, baseURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  p.AuthorizeURI,
			TokenURL: p.AccessTokenURI,
		},
		Scopes:      p.Scopes,
		RedirectURL: fmt.Sprintf("%s%s", baseURL, fmt.Sprintf(OAuthLoginSuccessPattern, p.Name)),
	}
}
