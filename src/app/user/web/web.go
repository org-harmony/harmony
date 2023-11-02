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
	registerNavigation(appCtx, webCtx)
	registerTemplateDataExtensions(appCtx, webCtx)

	router := webCtx.Router

	authCfg := &auth.Cfg{}
	util.Ok(config.C(authCfg, config.From("auth"), config.Validate(appCtx.Validator)))

	router.Get("/auth/login", loginController(appCtx, webCtx, authCfg).ServeHTTP)
	router.Get("/auth/logout", logoutController(appCtx, webCtx).ServeHTTP)

	userRouter := router.With(user.Middleware(user.SessionStore(appCtx)))
	userRouter.Get("/user/me", userProfileController(appCtx, webCtx).ServeHTTP)
	userRouter.Post("/user/me", userProfileEditController(appCtx, webCtx).ServeHTTP)

	if authCfg.EnableOAuth2 {
		registerOAuth2Controller(appCtx, webCtx, authCfg)
	}
}

func registerNavigation(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	webCtx.Navigation.Add("user.edit", web.NavItem{
		URL:  "/user/me",
		Name: "harmony.menu.user",
		Display: func(io web.IO) (bool, error) {
			u, _ := user.CtxUser(io.Context())
			return u != nil, nil
		},
		Position: 1000,
	})

	webCtx.Navigation.Add("user.logout", web.NavItem{
		Redirect: true,
		URL:      "/auth/logout",
		Name:     "harmony.menu.logout",
		Display: func(io web.IO) (bool, error) {
			u, _ := user.CtxUser(io.Context())
			return u != nil, nil
		},
		Position: 1100,
	})

	webCtx.Navigation.Add("user.login", web.NavItem{
		URL:  "/auth/login",
		Name: "harmony.menu.login",
		Display: func(io web.IO) (bool, error) {
			u, _ := user.CtxUser(io.Context())
			return u == nil, nil
		},
		Position: 1000,
	})
}

func registerTemplateDataExtensions(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	webCtx.Extensions.Add("user", func(io web.IO, data *web.BaseTemplateData) error {
		u, err := user.CtxUser(io.Context())
		if err != nil {
			return nil
		}

		data.Extra["User"] = u
		return nil
	})
}

func loginController(appCtx *hctx.AppCtx, webCtx *web.Ctx, authCfg *auth.Cfg) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		_, err := user.CtxUser(io.Context())
		if err == nil {
			return io.Redirect("/user/me", http.StatusTemporaryRedirect)
		}

		return io.Render("auth.login", "user/auth/login.go.html", authCfg)
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

func userProfileController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		u := util.Unwrap(user.CtxUser(io.Context()))

		return io.RenderJoined(
			web.NewFormData(u.ToUpdate(), nil),
			"user.edit",
			"user/edit.go.html",
			"user/_form-edit.go.html",
		)
	})
}

func userProfileEditController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	userRepository := util.UnwrapType[user.Repository](appCtx.Repository(user.RepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		context := io.Context()
		u := util.Unwrap(user.CtxUser(context))
		request := io.Request()
		toUpdate := u.ToUpdate()

		err, validationErrs := web.ReadForm(request, toUpdate, appCtx.Validator)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		if validationErrs != nil {
			return io.Render("user.edit.form", "user/_form-edit.go.html", web.NewFormData(toUpdate, nil, validationErrs...))
		}

		err = user.UpdateUser(context, u, toUpdate, userRepository)

		return io.Render(
			"user.edit.form",
			"user/_form-edit.go.html",
			web.NewFormData(u.ToUpdate(), []string{"user.settings.updated"}, err),
		)
	})
}

func registerOAuth2Controller(appCtx *hctx.AppCtx, webCtx *web.Ctx, authCfg *auth.Cfg) {
	providers := authCfg.Providers
	router := webCtx.Router

	router.Get("/auth/login/{provider}", oAuthLoginController(appCtx, webCtx, providers).ServeHTTP)
	router.Get("/auth/login/{provider}/success", oAuthLoginSuccessController(appCtx, webCtx, providers, user.Adapters()).ServeHTTP)
}
