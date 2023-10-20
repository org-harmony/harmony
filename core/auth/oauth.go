package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/org-harmony/harmony/core/hctx"
	"github.com/org-harmony/harmony/core/persistence"
	"github.com/org-harmony/harmony/core/util"
	"github.com/org-harmony/harmony/core/web"
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
	Email(token *oauth2.Token, cfg *ProviderCfg, client *http.Client, c context.Context) (string, error)
	CreateUser(email string, token *oauth2.Token, cfg *ProviderCfg, client *http.Client, c context.Context) (*UserToCreate, error)
}

func getUserAdapters() map[string]OAuthUserAdapter {
	return map[string]OAuthUserAdapter{
		"github": &GitHubUserAdapter{},
	}
}

func oAuthLoginController(appCtx hctx.AppContext, webCtx web.Context, providers map[string]ProviderCfg) http.Handler {
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

func oAuthLoginSuccessController(
	appCtx hctx.AppContext,
	webCtx web.Context,
	providers map[string]ProviderCfg,
	adapters map[string]OAuthUserAdapter,
) http.Handler {
	errT := util.Unwrap(util.Unwrap(webCtx.TemplaterStore().Templater(web.LandingPageTemplateName)).
		Template("error", "error.go.html"))
	userRepository := util.UnwrapType[UserRepository](appCtx.Repository(UserRepositoryName))

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
			return io.Error(errT, web.ExtErr("Der gewählte OAuth Provider ist leider nicht unterstützt."))
		}

		token, adapter, err := oauthVerify(reqCtx, code, state, oAuthCfg, provider.Name, adapters)
		if err != nil {
			appCtx.Error(Pkg, "error verifying oauth 2 login", err)
			return io.Error(errT, web.ExtErr("Login per OAuth fehlgeschlagen. Bitte erneut versuchen."))
		}

		user, err := loginWithAdapter(reqCtx, token, provider, adapter, userRepository)
		if err != nil {
			appCtx.Error(Pkg, "error logging in user", err)
			return io.Error(errT, web.ExtErr("Login per OAuth fehlgeschlagen. Bitte erneut versuchen."))
		}

		fmt.Printf("user logged in: %+v\n", user)

		return io.Redirect("/", http.StatusTemporaryRedirect)
	})
}

func oauthVerify(
	ctx context.Context,
	code string,
	state string,
	cfg *oauth2.Config,
	providerName string,
	adapters map[string]OAuthUserAdapter,
) (*oauth2.Token, OAuthUserAdapter, error) {
	if state != "state" {
		return nil, nil, errors.New("invalid oauth state")
	}

	token, err := cfg.Exchange(ctx, code)
	if err != nil {
		return nil, nil, fmt.Errorf("code exchange failed: %s", err.Error())
	}

	adapter, ok := adapters[providerName]
	if !ok {
		return nil, nil, fmt.Errorf("no adapter for provider %s found", providerName)
	}

	return token, adapter, nil
}

func loginWithAdapter(
	ctx context.Context,
	token *oauth2.Token,
	provider *ProviderCfg,
	adapter OAuthUserAdapter,
	userRepo UserRepository,
) (*User, error) {
	email, err := adapter.Email(token, provider, http.DefaultClient, ctx)
	if err != nil {
		return nil, err
	}

	user, err := userRepo.FindByEmail(email, ctx)
	if err != nil && !errors.Is(err, persistence.NotFoundError) {
		return nil, err
	}

	if user != nil {
		return user, nil
	}

	userToCreate, err := adapter.CreateUser(email, token, provider, http.DefaultClient, ctx)
	if err != nil {
		return nil, err
	}

	user, err = userRepo.Create(userToCreate, ctx)
	if err != nil {
		return nil, err
	}

	return user, nil
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
		RedirectURL: fmt.Sprintf("%s%s", baseURL, fmt.Sprintf("/auth/login/%s/success", p.Name)),
	}
}
