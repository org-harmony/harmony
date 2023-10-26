package web

import (
	"github.com/org-harmony/harmony/src/app/user"
	"github.com/org-harmony/harmony/src/core/auth"
	"github.com/org-harmony/harmony/src/core/config"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/web"
	"net/http"
)

const Pkg = "app.user"

func RegisterController(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	router := webCtx.Router

	authCfg := &auth.Cfg{}
	util.Ok(config.C(authCfg, config.From("auth"), config.Validate(appCtx.Validator)))

	router.With(user.Middleware(user.SessionStore(appCtx), user.AllowAnonymous)).
		Get("/auth/login", loginController(appCtx, webCtx, authCfg).ServeHTTP)

	router.Get("/auth/logout", logoutController(appCtx, webCtx).ServeHTTP)

	if authCfg.EnableOAuth2 {
		registerOAuth2Controller(appCtx, webCtx, authCfg)
	}
}

func loginController(appCtx *hctx.AppCtx, webCtx *web.Ctx, providers *auth.Cfg) http.Handler {
	loginTemplate := util.Unwrap(util.Unwrap(webCtx.TemplaterStore.Templater(web.LandingPageTemplateName)).
		Template("auth.login", "auth/login.go.html"))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		_, err := user.CtxUser(io.Context())
		if err == nil {
			return io.Redirect("/auth/user", http.StatusTemporaryRedirect)
		}

		return io.Render(loginTemplate, web.NewTemplateData(providers))
	})
}

func logoutController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	sessionStore := user.SessionStore(appCtx)

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		sessionID, err := user.SessionIDFromRequest(io.Request())
		if err != nil {
			return io.Redirect("/", http.StatusTemporaryRedirect)
		}

		auth.ClearSession(io.Response(), user.SessionCookieName)
		err = sessionStore.Delete(io.Context(), sessionID)
		if err != nil {
			return err
		}

		return io.Redirect("/", http.StatusTemporaryRedirect)
	})
}

func userController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	userTemplate := util.Unwrap(util.Unwrap(webCtx.TemplaterStore.Templater(web.LandingPageTemplateName)).
		Template("auth.user", "auth/user.go.html"))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		user := util.Unwrap(user.CtxUser(io.Context()))

		return io.Render(userTemplate, user)
	})
}

func registerOAuth2Controller(appCtx *hctx.AppCtx, webCtx *web.Ctx, authCfg *auth.Cfg) {
	providers := authCfg.Providers
	router := webCtx.Router

	router.Get("/auth/login/{provider}", oAuthLoginController(appCtx, webCtx, providers).ServeHTTP)
	router.Get("/auth/login/{provider}/success", oAuthLoginSuccessController(appCtx, webCtx, providers, user.Adapters()).ServeHTTP)
}
