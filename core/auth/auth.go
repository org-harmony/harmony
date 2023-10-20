// Package auth provides authentication details and logic for HARMONY.
// Auth is a part of the core package as it provides user authentication for all domains.
package auth

import (
	"github.com/org-harmony/harmony/core/config"
	"github.com/org-harmony/harmony/core/hctx"
	"github.com/org-harmony/harmony/core/util"
	"github.com/org-harmony/harmony/core/web"
	"net/http"
)

const Pkg = "sys.auth"

// Cfg is the config for the auth package.
type Cfg struct {
	// Providers contains a list of OAuth2 providers.
	Providers    map[string]ProviderCfg `toml:"provider"`
	EnableOAuth2 bool                   `toml:"enable_oauth2"`
}

func RegisterAuth(appCtx hctx.AppContext, webCtx web.Context) {
	cfg := &Cfg{}
	util.Ok(config.C(cfg, config.From("auth"), config.Validate(appCtx.Validator())))

	registerRoutes(cfg, appCtx, webCtx)
}

func registerRoutes(cfg *Cfg, appCtx hctx.AppContext, webCtx web.Context) {
	router := webCtx.Router()
	router.Get("/auth/login", loginController(cfg, appCtx, webCtx).ServeHTTP)

	if !cfg.EnableOAuth2 {
		return
	}

	providers := cfg.Providers
	router.Get("/auth/login/{provider}", oAuthLoginController(appCtx, webCtx, providers).ServeHTTP)
	router.Get("/auth/login/{provider}/success", oAuthLoginSuccessController(appCtx, webCtx, providers, getUserAdapters()).ServeHTTP)
}

func loginController(cfg *Cfg, appCtx hctx.AppContext, webCtx web.Context) http.Handler {
	loginT := util.Unwrap(util.Unwrap(webCtx.TemplaterStore().Templater(web.LandingPageTemplateName)).
		Template("auth.login", "auth/login.go.html"))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.Render(loginT, cfg)
	})
}
