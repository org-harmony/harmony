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

func oAuthLoginController(appCtx *hctx.AppCtx, webCtx web.Context, providers map[string]*auth.ProviderCfg) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		name := web.URLParam(io.Request(), "provider")
		redirectURL := oAuthProviderRedirectURL(webCtx, name)

		oAuthCfg, _, err := oAuthCfgFromProviderMap(name, providers, redirectURL)
		if err != nil {
			return io.Error(web.ExtErr("auth.error.invalid-provider"))
		}

		url := oAuthCfg.AuthCodeURL("state") // TODO dynamize state through method in auth.go

		return io.Redirect(url, http.StatusTemporaryRedirect)
	})
}

func oAuthLoginSuccessController(
	appCtx *hctx.AppCtx,
	webCtx web.Context,
	providers map[string]*auth.ProviderCfg,
	adapters map[string]user.OAuthUserAdapter,
) http.Handler {
	userRepository := util.UnwrapType[user.Repository](appCtx.Repository(user.RepositoryName))
	sessionStore := user.SessionStore(appCtx)

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		request := io.Request()
		name := web.URLParam(request, "provider")
		provider, ok := providers[name]
		if !ok {
			return io.Error(web.ExtErr("auth.error.invalid-provider"))
		}

		session, err := auth.OAuthLogin(
			request.Context(),
			request.FormValue("state"),
			request.FormValue("code"),
			provider,
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

// oAuthProviderRedirectURL returns the redirect URL for a specified provider.
func oAuthProviderRedirectURL(webCtx web.Context, providerName string) string {
	return fmt.Sprintf(
		"%s%s",
		webCtx.Configuration().Server.BaseURL,
		fmt.Sprintf("/auth/login/%s/success", providerName),
	)
}

// oAuthCfgFromProviderMap returns the OAuth2 configuration for a specified provider.
func oAuthCfgFromProviderMap(name string, providers map[string]*auth.ProviderCfg, redirectURL string) (*oauth2.Config, *auth.ProviderCfg, error) {
	p, ok := providers[name]
	if !ok {
		return nil, nil, fmt.Errorf("auth: provider %s not found", name)
	}

	return auth.OAuthCfgFromProviderCfg(p, redirectURL), p, nil
}
