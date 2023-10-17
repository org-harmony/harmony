// Package auth provides authentication details and logic for HARMONY.
// Auth is a part of the core package as it provides user authentication for all domains.
package auth

import (
	"fmt"
	"github.com/org-harmony/harmony/core/config"
	"github.com/org-harmony/harmony/core/ctx"
	"github.com/org-harmony/harmony/core/util"
	"github.com/org-harmony/harmony/core/web"
	"golang.org/x/oauth2"
	ioutil "io"
	"net/http"
)

const (
	Pkg                      = "sys.auth"
	OAuthLoginPattern        = "/auth/login/%s"
	OAuthLoginSuccessPattern = "/auth/login/%s/success"
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
	UserinfoURI    string   `toml:"userinfo_uri" validate:"required"`
	ClientID       string   `toml:"client_id" validate:"required"`
	ClientSecret   string   `toml:"client_secret" validate:"required"`
	Scopes         []string `toml:"scopes" validate:"required"`
}

func RegisterAuth(appCtx ctx.AppContext, webCtx web.Context) {
	cfg := &Cfg{}
	util.Ok(config.C(cfg, config.From("auth"), config.Validate(appCtx.Validator())))

	registerRoutes(cfg, appCtx, webCtx)
}

func registerRoutes(cfg *Cfg, appCtx ctx.AppContext, webCtx web.Context) {
	router := webCtx.Router()
	router.Get("/auth/login", loginController(cfg, appCtx, webCtx).ServeHTTP)

	if !cfg.EnableOAuth2 {
		return
	}

	providers := cfg.Providers
	router.Get(fmt.Sprintf(OAuthLoginPattern, "{provider}"), oAuthLoginController(appCtx, webCtx, providers).ServeHTTP)
	router.Get(fmt.Sprintf(OAuthLoginSuccessPattern, "{provider}"), oAuthLoginSuccessController(appCtx, webCtx, providers).ServeHTTP)
}

func loginController(cfg *Cfg, appCtx ctx.AppContext, webCtx web.Context) http.Handler {
	loginT := util.Unwrap(util.Unwrap(webCtx.TemplaterStore().Templater(web.LandingPageTemplateName)).
		Template("auth.login", "auth/login.go.html"))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.Render(loginT, cfg)
	})
}

func oAuthLoginController(appCtx ctx.AppContext, webCtx web.Context, providers map[string]ProviderCfg) http.Handler {
	errT := util.Unwrap(util.Unwrap(webCtx.TemplaterStore().Templater(web.LandingPageTemplateName)).
		Template("error", "error.go.html"))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		oAuthCfg, _, err := oAuthConfigByProviderName(
			webCtx.Router().URLParam(io.Request(), "provider"),
			providers,
			webCtx.Configuration().Server.BaseURL,
		)
		if err != nil {
			return io.Error(errT, web.ExtErr("auth.error.invalid-provider"))
		}

		url := oAuthCfg.AuthCodeURL("state")

		return io.Redirect(url, http.StatusTemporaryRedirect)
	})
}

func oAuthLoginSuccessController(appCtx ctx.AppContext, webCtx web.Context, providers map[string]ProviderCfg) http.Handler {
	errT := util.Unwrap(util.Unwrap(webCtx.TemplaterStore().Templater(web.LandingPageTemplateName)).
		Template("error", "error.go.html"))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		request := io.Request()
		state := request.FormValue("state")
		code := request.FormValue("code")

		oAuthCfg, provider, err := oAuthConfigByProviderName(
			webCtx.Router().URLParam(io.Request(), "provider"),
			providers,
			webCtx.Configuration().Server.BaseURL,
		)
		if err != nil {
			return io.Error(errT, web.ExtErr("auth.error.invalid-provider"))
		}

		if state != "state" {
			return io.Error(errT, web.ExtErr("auth.error.invalid-state"))
		}

		token, err := oAuthCfg.Exchange(request.Context(), code)
		if err != nil {
			return io.Error(errT, web.ExtErr("auth.error.invalid-token"))
		}

		req, err := http.NewRequest(http.MethodGet, provider.UserinfoURI, nil)
		if err != nil {
			return io.Error(errT, web.ExtErr("auth.error.invalid-userinfo-request"))
		}
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))

		c := http.DefaultClient
		userinfo, err := c.Do(req)
		if err != nil {
			return io.Error(errT, web.ExtErr("auth.error.invalid-userinfo-response"))
		}
		defer userinfo.Body.Close()

		contents, err := ioutil.ReadAll(userinfo.Body)
		if err != nil {
			return io.Error(errT, web.ExtErr("auth.error.invalid-userinfo-response"))
		}

		fmt.Printf("userinfo: %s", contents)
		fmt.Printf("access token: %s, token: %+v", token.AccessToken, token)

		return nil
	})
}

func oAuthConfigByProviderName(name string, providers map[string]ProviderCfg, baseURL string) (*oauth2.Config, *ProviderCfg, error) {
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
