package web

import (
	"context"
	"fmt"
	"github.com/org-harmony/harmony/src/app/user"
	"github.com/org-harmony/harmony/src/core/auth"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/web"
	"golang.org/x/oauth2"
	"net/http"
)

func oAuthLoginController(appCtx hctx.AppContext, webCtx web.Context, providers map[string]auth.ProviderCfg) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		name := web.URLParam(io.Request(), "provider")

		oAuthCfg, _, err := auth.OAuthConfig(name, providers, webCtx.Configuration().Server.BaseURL)
		if err != nil {
			return io.Error(web.ExtErr("auth.error.invalid-provider"))
		}

		url := oAuthCfg.AuthCodeURL("state") // TODO state

		return io.Redirect(url, http.StatusTemporaryRedirect)
	})
}

func oAuthLoginSuccessController(
	appCtx hctx.AppContext,
	webCtx web.Context,
	providers map[string]auth.ProviderCfg,
	adapters map[string]user.OAuthUserAdapter,
) http.Handler {
	userRepository := util.UnwrapType[user.Repository](appCtx.Repository(user.RepositoryName))
	sessionStore := user.SessionStore(appCtx)

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		request := io.Request()

		session, err := auth.OAuthLogin(
			request,
			providers,
			webCtx.Configuration().Server.BaseURL,
			func(
				ctx context.Context,
				token *oauth2.Token,
				provider *auth.ProviderCfg,
			) (*persistence.Session[user.User, user.SessionMeta], error) {
				userAdapter, ok := adapters[provider.Name]
				if !ok {
					return nil, fmt.Errorf("oauth user adapter for provider %s not found", provider.Name)
				}

				userSession, err := user.LoginWithAdapter(ctx, token, provider, userAdapter, userRepository, sessionStore)
				if err != nil {
					return nil, err
				}

				return &userSession.Session, nil
			},
		)

		if err != nil {
			appCtx.Error(Pkg, "error logging in with oauth", err)
			return io.Error(web.ExtErr("An error occurred while logging in with OAuth 2, please again later."))
		}

		auth.SetSession(io.Response(), user.SessionCookieName, session)

		return io.Redirect("/auth/user", http.StatusTemporaryRedirect)
	})
}
