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

// RedirectToLogin redirects the user to the login page.
func RedirectToLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/auth/login", http.StatusTemporaryRedirect)
}

func RegisterAuth(appCtx hctx.AppContext, webCtx web.Context) {
	cfg := &Cfg{}
	util.Ok(config.C(cfg, config.From("auth"), config.Validate(appCtx.Validator())))

	registerRoutes(cfg, appCtx, webCtx)
}

func registerRoutes(cfg *Cfg, appCtx hctx.AppContext, webCtx web.Context) {
	router := webCtx.Router()

	router.With(Middleware(UserSessionStore(appCtx), AllowAnonymous)).
		Get("/auth/login", loginController(cfg, appCtx, webCtx).ServeHTTP)

	router.Get("/auth/logout", logoutController(appCtx, webCtx).ServeHTTP) // middleware not required: session is used, user not

	router.With(Middleware(UserSessionStore(appCtx), NotLoggedInHandler(RedirectToLogin))).
		Get("/auth/user", userController(appCtx, webCtx).ServeHTTP)

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
		_, err := CtxUser(io.Request().Context())
		if err == nil {
			return io.Redirect("/auth/user", http.StatusTemporaryRedirect)
		}

		return io.Render(loginT, cfg)
	})
}

func logoutController(appCtx hctx.AppContext, webCtx web.Context) http.Handler {
	sessionStore := UserSessionStore(appCtx)

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		sessionID, err := SessionIDFromRequest(io.Request())
		if err != nil {
			return io.Redirect("/", http.StatusTemporaryRedirect)
		}

		clearSession(io.Response())
		err = sessionStore.Delete(io.Request().Context(), sessionID)
		if err != nil {
			return err
		}

		return io.Redirect("/", http.StatusTemporaryRedirect)
	})
}

func userController(appCtx hctx.AppContext, webCtx web.Context) http.Handler {
	userT := util.Unwrap(util.Unwrap(webCtx.TemplaterStore().Templater(web.LandingPageTemplateName)).
		Template("auth.user", "auth/user.go.html"))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		user := util.Unwrap(CtxUser(io.Request().Context()))

		return io.Render(userT, user)
	})
}
