package auth

import (
	"context"
	"encoding/json"
	"fmt"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"strings"
)

type GitHubUserAdapter struct{}

func (g *GitHubUserAdapter) Email(ctx context.Context, token *oauth2.Token, cfg *ProviderCfg, client *http.Client) (string, error) {
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

func (g *GitHubUserAdapter) CreateUser(ctx context.Context, email string, token *oauth2.Token, cfg *ProviderCfg, client *http.Client) (*UserToCreate, error) {
	userinfo, err := githubGetUserinfo(ctx, token.AccessToken, cfg.UserinfoURI, client)
	if err != nil {
		return nil, err
	}

	firstname, lastname, err := namesFromUserInfo(userinfo)
	if err != nil {
		return nil, err
	}

	return &UserToCreate{
		Email:     email,
		Firstname: firstname,
		Lastname:  lastname,
	}, nil
}

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
