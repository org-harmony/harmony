package user

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/org-harmony/harmony/src/core/auth"
	"github.com/org-harmony/harmony/src/core/persistence"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"strings"
)

type GitHubUserAdapter struct{}

// OAuthUserAdapter adapts the OAuth2 user data to the user entity.
// The Email method returns the email address of the user this is used to find the user in the database.
// If the user was not found and can therefore not be logged in, the CreateUser method is called.
// CreateUser then returns a ToCreate struct which is used to create a new user.
type OAuthUserAdapter interface {
	Email(ctx context.Context, token *oauth2.Token, cfg *auth.ProviderCfg, client *http.Client) (string, error)
	CreateUser(ctx context.Context, email string, token *oauth2.Token, cfg *auth.ProviderCfg, client *http.Client) (*ToCreate, error)
}

// Adapters returns a map of OAuthUserAdapters.
// These adapters are used to adapt the OAuth2 user data to the user entity.
func Adapters() map[string]OAuthUserAdapter {
	return map[string]OAuthUserAdapter{
		"github": &GitHubUserAdapter{},
	}
}

// LoginWithAdapter logs in the user with the given OAuthUserAdapter.
// First it checks if the user already exists in the database. If so, the user is logged in.
// The email address of the user is used to find the user in the database.
// If the user doesn't exist, the OAuthUserAdapter.CreateUser creates the user.
// After creating the user, the user is logged in and LoginWithAdapter returns the session.
func LoginWithAdapter(
	ctx context.Context,
	token *oauth2.Token,
	provider *auth.ProviderCfg,
	adapter OAuthUserAdapter,
	userRepo Repository,
	sessionStore SessionRepository,
) (*Session, error) {
	email, err := adapter.Email(ctx, token, provider, http.DefaultClient)
	if err != nil {
		return nil, err
	}

	user, err := userRepo.FindByEmail(ctx, email)
	if err != nil && !errors.Is(err, persistence.ErrNotFound) {
		return nil, err
	}

	if user != nil {
		session, err := Login(ctx, user, sessionStore)
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

	session, err := Login(ctx, user, sessionStore)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// Email on the GitHubUserAdapter returns the email address of the user.
// It will first try to get the email from the userinfo endpoint.
// If that fails, it will try to get the primary email from the GitHub api.
func (g *GitHubUserAdapter) Email(ctx context.Context, token *oauth2.Token, cfg *auth.ProviderCfg, client *http.Client) (string, error) {
	userinfo, err := githubGetUserinfo(ctx, token.AccessToken, cfg.UserinfoURI, client)
	if err != nil {
		return "", err
	}

	email, err := emailFromUserinfo(userinfo)
	if err != nil {
		email, err = githubPrimaryEmail(ctx, token.AccessToken, client)
	}

	if email == "" {
		return "", fmt.Errorf("no email found in userinfo or as primary email at github")
	}

	return email, nil
}

// CreateUser on the GitHubUserAdapter creates a new user with the given email address and name from the userinfo endpoint.
// It splits the name into firstname and lastname.
func (g *GitHubUserAdapter) CreateUser(ctx context.Context, email string, token *oauth2.Token, cfg *auth.ProviderCfg, client *http.Client) (*ToCreate, error) {
	userinfo, err := githubGetUserinfo(ctx, token.AccessToken, cfg.UserinfoURI, client)
	if err != nil {
		return nil, err
	}

	firstname, lastname, err := namesFromUserInfo(userinfo)
	if err != nil {
		return nil, err
	}

	return &ToCreate{
		Email:     email,
		Firstname: firstname,
		Lastname:  lastname,
	}, nil
}

// githubGetUserinfo returns the userinfo from the userinfo endpoint.
func githubGetUserinfo(ctx context.Context, token string, url string, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	return string(content), nil
}

// githubPrimaryEmail returns the primary email from the GitHub api.
func githubPrimaryEmail(ctx context.Context, token string, client *http.Client) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, "https://api.github.com/user/emails", nil)
	if err != nil {
		return "", err
	}

	req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", token))

	response, err := client.Do(req)
	if err != nil {
		return "", err
	}
	defer response.Body.Close()

	content, err := io.ReadAll(response.Body)
	if err != nil {
		return "", err
	}

	var emails []struct {
		Primary bool   `json:"primary"`
		Email   string `json:"email"`
	}
	err = json.Unmarshal(content, &emails)
	if err != nil {
		return "", err
	}

	for _, e := range emails {
		if e.Primary {
			return e.Email, nil
		}
	}

	return "", fmt.Errorf("no primary email found at github")
}

// emailFromUserinfo returns the email from the userinfo.
func emailFromUserinfo(userinfo string) (string, error) {
	var email struct{ Email string }
	err := json.Unmarshal([]byte(userinfo), &email)
	if err != nil {
		return "", err
	}

	emailString := strings.ToLower(email.Email)
	if emailString == "" {
		return "", fmt.Errorf("no email found in userinfo")
	}

	return emailString, nil
}

// namesFromUserInfo returns the firstname and lastname from the userinfo.
// The first string is the firstname, the second string is the lastname.
func namesFromUserInfo(userinfo string) (string, string, error) {
	var name struct{ Name string }
	err := json.Unmarshal([]byte(userinfo), &name)
	if err != nil {
		return "", "", err
	}

	nameParts := strings.Split(name.Name, " ")
	firstname := nameParts[0]
	lastname := "<HARMONY Anwender>"

	if len(nameParts) > 1 {
		lastname = nameParts[1]
	}

	return firstname, lastname, nil
}
