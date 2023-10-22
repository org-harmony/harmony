package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/web"
	"golang.org/x/oauth2"
	"net/http"
)

// ProviderCfg is the config for an OAuth2 provider.
type ProviderCfg struct {
	Name           string   `toml:"name" validate:"required"`
	DisplayName    string   `toml:"display_name" validate:"required"`
	AuthorizeURI   string   `toml:"authorize_uri" validate:"required"`
	AccessTokenURI string   `toml:"access_token_uri" validate:"required"`
	UserinfoURI    string   `toml:"userinfo_uri"`
	ClientID       string   `toml:"client_id" validate:"required"`
	ClientSecret   string   `toml:"client_secret" validate:"required"`
	Scopes         []string `toml:"scopes" validate:"required"`
}

// OAuthUserAdapter adapts the OAuth2 user data to the user entity.
// Email method returns the email address of the user this is used to find the user in the database.
// If the user was not find and can therefore not be logged in, the CreateUser method is called.
// CreateUser then returns a UserToCreate struct which is used to create a new user.
type OAuthUserAdapter interface {
	Email(ctx context.Context, token *oauth2.Token, cfg *ProviderCfg, client *http.Client) (string, error)
	CreateUser(ctx context.Context, email string, token *oauth2.Token, cfg *ProviderCfg, client *http.Client) (*UserToCreate, error)
}

// getUserAdapters returns a map of OAuthUserAdapters. They are used to adapt the OAuth2 user data to the user entity.
func getUserAdapters() map[string]OAuthUserAdapter {
	return map[string]OAuthUserAdapter{
		"github": &GitHubUserAdapter{},
	}
}

func oAuthLoginController(appCtx hctx.AppContext, webCtx web.Context, providers map[string]ProviderCfg) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		oAuthCfg, _, err := oAuthConfigByProviderName(
			webCtx.Router().URLParam(io.Request(), "provider"),
			providers,
			webCtx.Configuration().Server.BaseURL,
		)
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
	providers map[string]ProviderCfg,
	adapters map[string]OAuthUserAdapter,
) http.Handler {
	userRepository := util.UnwrapType[UserRepository](appCtx.Repository(UserRepositoryName))
	sessionStore := UserSessionStore(appCtx)

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		request := io.Request()
		reqCtx := request.Context()
		state := request.FormValue("state")
		code := request.FormValue("code")

		oAuthCfg, provider, err := oAuthConfigByProviderName(
			webCtx.Router().URLParam(io.Request(), "provider"),
			providers,
			webCtx.Configuration().Server.BaseURL,
		)
		if err != nil {
			return io.Error(web.ExtErr("Der gewählte OAuth Provider ist leider nicht unterstützt."))
		}

		token, err := oauthVerify(reqCtx, code, state, oAuthCfg)
		if err != nil {
			appCtx.Error(Pkg, "error verifying oauth login", err)
			return io.Error(web.ExtErr("Login per OAuth fehlgeschlagen. Bitte erneut versuchen."))
		}

		adapter, ok := adapters[provider.Name]
		if !ok {
			return io.Error(web.ExtErr("Der gewählte OAuth Provider ist leider nicht unterstützt."))
		}

		session, err := loginWithAdapter(reqCtx, token, provider, adapter, userRepository, sessionStore)
		if err != nil {
			appCtx.Error(Pkg, "error logging in user", err)
			return io.Error(web.ExtErr("Login per OAuth fehlgeschlagen. Bitte erneut versuchen."))
		}

		setSession(io.Response(), session)

		return io.Redirect("/", http.StatusTemporaryRedirect)
	})
}

// oauthVerify verifies the oauth login by verifying the state and exchanging the code for a token.
func oauthVerify(ctx context.Context, code string, state string, cfg *oauth2.Config) (*oauth2.Token, error) {
	if state != "state" {
		return nil, errors.New("invalid oauth state")
	}

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}

	return token, nil
}

// loginWithAdapter logs in the user with the given OAuthUserAdapter.
// First it checks if the user already exists in the database. If so, the user is logged in.
// The email address of the user is used to find the user in the database.
// If the user doesn't exist, the OAuthUserAdapter.CreateUser creates the user.
// After creating the user, the user is logged in and loginWithAdapter returns the session.
func loginWithAdapter(
	ctx context.Context,
	token *oauth2.Token,
	provider *ProviderCfg,
	adapter OAuthUserAdapter,
	userRepo UserRepository,
	sessionStore UserSessionRepository,
) (*UserSession, error) {
	email, err := adapter.Email(ctx, token, provider, http.DefaultClient)
	if err != nil {
		return nil, err
	}

	user, err := userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, persistence.ErrNotFound) {
		return nil, err
	}

	if user != nil {
		session, err := login(ctx, user, sessionStore)
		if err != nil {
			return nil, err
		}

		return session, nil
	}

	userToCreate, err := adapter.CreateUser(ctx, email, token, provider, http.DefaultClient)
	if err != nil {
		return nil, err
	}

	user, err = userRepo.Create(ctx, userToCreate)
	if err != nil {
		return nil, err
	}

	session, err := login(ctx, user, sessionStore)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// oAuthConfigByProviderName returns the OAuth2 config for the given provider name.
func oAuthConfigByProviderName(name string, providers map[string]ProviderCfg, baseURL string) (*oauth2.Config, *ProviderCfg, error) {
	p, ok := providers[name]
	if !ok {
		return nil, nil, fmt.Errorf("auth: provider %s not found", name)
	}

	return oAuthCfgFromProviderCfg(p, baseURL), &p, nil
}

// oAuthCfgFromProviderCfg returns the OAuth2 config for the given provider config.
func oAuthCfgFromProviderCfg(p ProviderCfg, baseURL string) *oauth2.Config {
	return &oauth2.Config{
		ClientID:     p.ClientID,
		ClientSecret: p.ClientSecret,
		Endpoint: oauth2.Endpoint{
			AuthURL:  p.AuthorizeURI,
			TokenURL: p.AccessTokenURI,
		},
		Scopes:      p.Scopes,
		RedirectURL: fmt.Sprintf("%s%s", baseURL, fmt.Sprintf("/auth/login/%s/success", p.Name)),
	}
}
