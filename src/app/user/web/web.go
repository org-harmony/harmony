package web

import (
	"errors"
	"net/http"
	"time"

	"github.com/org-harmony/harmony/src/app/user"
	"github.com/org-harmony/harmony/src/core/auth"
	"github.com/org-harmony/harmony/src/core/config"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/trans"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/web"
)

const Pkg = "app.user.web"

// ErrUpdateUser is returned when the user could not be updated. It is the error message for the user.edit.form template.
var ErrUpdateUser = errors.New("user.settings.update-error")

// RegisterController registers the web controllers for the user module.
// It registers the following routes:
//   - GET /user/me/language/{locale} For updating the user language.
//   - GET /auth/login For displaying various OAuth2 login buttons.
//   - GET /auth/logout For logging out the user.
//   - GET /user/me For displaying the user profile.
//   - POST /user/me For updating the user profile.
//
// If OAuth2 is enabled in the configuration, it also registers the following routes:
//   - GET /auth/login/{provider} For redirecting the user to the OAuth2 provider with the necessary parameters.
//   - GET /auth/login/{provider}/success For handling the OAuth2 callback and logging the user in.
func RegisterController(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	registerNavigation(appCtx, webCtx)
	registerTemplateDataExtensions(appCtx, webCtx)

	router := webCtx.Router

	authCfg := &auth.Cfg{}
	util.Ok(config.C(authCfg, config.From("auth"), config.Validate(appCtx.Validator)))

	router.Get("/user/me/language/{locale}", userLanguageController(appCtx, webCtx).ServeHTTP)
	router.Get("/auth/login", loginController(appCtx, webCtx, authCfg).ServeHTTP)
	router.Get("/auth/logout", logoutController(appCtx, webCtx).ServeHTTP)

	if authCfg.EnablePwdLogin {
		router.Post("/auth/magic/login", postMagicLoginController(appCtx, webCtx, authCfg).ServeHTTP)
		router.Post("/auth/magic/login/{id}", postMagicLoginExecController(appCtx, webCtx, authCfg).ServeHTTP)
	}

	userRouter := router.With(user.LoggedInMiddleware(appCtx))
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
		Position: 1200,
	})

	webCtx.Navigation.Add("user.login", web.NavItem{
		URL:  "/auth/login",
		Name: "harmony.menu.login",
		Display: func(io web.IO) (bool, error) {
			u, _ := user.CtxUser(io.Context())
			return u == nil, nil
		},
		Position: 1200,
	})

	webCtx.Navigation.Add("user.language.de", web.NavItem{
		URL:  "/user/me/language/de",
		Name: "harmony.menu.language.de",
		Display: func(io web.IO) (bool, error) {
			locale, err := io.Request().Cookie(trans.LocaleSessionKey)
			if err != nil {
				return false, nil
			}

			return locale.Value != "de", nil
		},
		Position: 1100,
	})

	webCtx.Navigation.Add("user.language.en", web.NavItem{
		URL:  "/user/me/language/en",
		Name: "harmony.menu.language.en",
		Display: func(io web.IO) (bool, error) {
			locale, err := io.Request().Cookie(trans.LocaleSessionKey)
			if err != nil {
				return true, nil
			}

			return locale.Value != "en", nil
		},
		Position: 1100,
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

func postMagicLoginController(appCtx *hctx.AppCtx, webCtx *web.Ctx, authCfg *auth.Cfg) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.Render(authCfg, "auth.magic.login", "user/auth/magic-login.go.html")
	})
}

func postMagicLoginExecController(appCtx *hctx.AppCtx, webCtx *web.Ctx, authCfg *auth.Cfg) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.Redirect("/user/me", http.StatusTemporaryRedirect)
	})
}

func loginController(appCtx *hctx.AppCtx, webCtx *web.Ctx, authCfg *auth.Cfg) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		_, err := user.CtxUser(io.Context())
		if err == nil {
			return io.Redirect("/user/me", http.StatusTemporaryRedirect)
		}

		return io.Render(authCfg, "auth.login", "user/auth/login.go.html")
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

func userLanguageController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		request := io.Request()
		locale := web.URLParam(request, "locale")

		cookie := http.Cookie{
			Name:     trans.LocaleSessionKey,
			Value:    locale,
			Expires:  time.Now().Add(365 * 24 * time.Hour),
			SameSite: http.SameSiteLaxMode,
			Path:     "/",
		}

		http.SetCookie(io.Response(), &cookie)

		return io.Redirect("/", http.StatusTemporaryRedirect)
	})
}

func userProfileController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.Render(
			web.NewFormData(user.MustCtxUser(io.Context()).ToUpdate(), nil),
			"user.edit",
			"user/edit.go.html",
			"user/_form-edit.go.html",
		)
	})
}

func userProfileEditController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	userRepository := util.UnwrapType[user.Repository](appCtx.Repository(user.RepositoryName))
	sessionStore := user.SessionStore(appCtx)

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		context := io.Context()
		request := io.Request()
		toUpdate := user.MustCtxUser(context).ToUpdate()

		err, validationErrs := web.ReadForm(request, toUpdate, appCtx.Validator)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		if validationErrs != nil {
			return renderUserEditForm(io, web.NewFormData(toUpdate, nil, validationErrs...))
		}

		session := util.Unwrap(user.SessionFromRequest(request, sessionStore))
		updatedUser, err := user.UpdateUser(context, toUpdate, session, userRepository, sessionStore)
		if err != nil {
			appCtx.Error(Pkg, "error updating user", err)
			return renderUserEditForm(io, web.NewFormData(toUpdate, nil, ErrUpdateUser))
		}

		return renderUserEditForm(io, web.NewFormData(updatedUser.ToUpdate(), []string{"user.settings.updated"}, err))
	})
}

func renderUserEditForm(io web.IO, data any) error {
	return io.Render(data, "user.edit.form", "user/_form-edit.go.html")
}

func registerOAuth2Controller(appCtx *hctx.AppCtx, webCtx *web.Ctx, authCfg *auth.Cfg) {
	providers := authCfg.Providers
	router := webCtx.Router

	router.Get("/auth/login/{provider}", oAuthLoginController(appCtx, webCtx, providers).ServeHTTP)
	router.Get("/auth/login/{provider}/success", oAuthLoginSuccessController(appCtx, webCtx, providers, user.Adapters()).ServeHTTP)
}
