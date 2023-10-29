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

	userRouter := router.With(user.Middleware(user.SessionStore(appCtx)))
	userRouter.Get("/user/me", userController(appCtx, webCtx).ServeHTTP)
	userRouter.Post("/user/me", userEditController(appCtx, webCtx).ServeHTTP)

	if authCfg.EnableOAuth2 {
		registerOAuth2Controller(appCtx, webCtx, authCfg)
	}
}

func loginController(appCtx *hctx.AppCtx, webCtx *web.Ctx, providers *auth.Cfg) http.Handler {
	loginTemplate := util.Unwrap(util.Unwrap(webCtx.TemplaterStore.Templater(web.LandingPageTemplateName)).
		Template("auth.login", "user/auth/login.go.html"))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		u, err := user.CtxUser(io.Context())
		if err == nil {
			return io.Redirect("/user/me", http.StatusTemporaryRedirect)
		}

		return io.Render(loginTemplate, user.NewTemplateData[*auth.Cfg](u, providers))
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
	templater := util.Unwrap(webCtx.TemplaterStore.Templater(web.LandingPageTemplateName))
	userEditTemplate := util.Unwrap(templater.JoinedTemplate("user.edit", "user/edit.go.html", "user/_form-edit.go.html"))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		u := util.Unwrap(user.CtxUser(io.Context()))

		return io.Render(userEditTemplate, user.NewTemplateData(u, web.NewFormTemplateData(u.ToUpdate())))
	})
}

func userEditController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templater := util.Unwrap(webCtx.TemplaterStore.Templater(web.EmptyTemplateName))
	userEditTemplate := util.Unwrap(templater.Template("user.edit.form", "user/_form-edit.go.html"))
	userRepository := util.UnwrapType[user.Repository](appCtx.Repository(user.RepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		context := io.Context()
		u := util.Unwrap(user.CtxUser(context))
		request := io.Request()
		if err := request.ParseForm(); err != nil {
			return io.Error()
		}

		toUpdate := u.ToUpdate()
		err := web.ReadForm(request, toUpdate, appCtx.Validator)
		if err != nil {
			// TODO map validation errors to form errors and create utility struct to display filtered above form (for * errors) and below each field (for field errors).
			return io.Error()
		}

		err = user.UpdateUser(context, u, toUpdate, userRepository)
		if err != nil {
			return io.Error() // make form violation error for *
		}

		return io.Render(userEditTemplate, web.NewFormTemplateData(u.ToUpdate()))
	})
}

func registerOAuth2Controller(appCtx *hctx.AppCtx, webCtx *web.Ctx, authCfg *auth.Cfg) {
	providers := authCfg.Providers
	router := webCtx.Router

	router.Get("/auth/login/{provider}", oAuthLoginController(appCtx, webCtx, providers).ServeHTTP)
	router.Get("/auth/login/{provider}/success", oAuthLoginSuccessController(appCtx, webCtx, providers, user.Adapters()).ServeHTTP)
}
