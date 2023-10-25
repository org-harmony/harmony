package auth

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/stretchr/testify/assert"
	"golang.org/x/oauth2"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
	"time"
)

type MockUser struct {
	Name string
}

type MockMeta struct {
	Foo bool
}

func TestOAuthLogin(t *testing.T) {
	testServer := mockOAuthServer(t)
	defer testServer.Close()

	providers := map[string]*ProviderCfg{
		"test": { // AuthorizeURI not needed because it is not used in this test
			Name:           "test",
			DisplayName:    "Test",
			AccessTokenURI: testServer.URL,
			ClientID:       "test",
			ClientSecret:   "test_secret",
			Scopes:         []string{"test"},
		},
	}

	oauthLoginServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		session, err := OAuthLogin(
			r.Context(),
			r.FormValue("state"),
			r.FormValue("code"),
			providers["test"],
			func(ctx context.Context, token *oauth2.Token, provider *ProviderCfg) (*persistence.Session[MockUser, MockMeta], error) {
				return &persistence.Session[MockUser, MockMeta]{
					ID:        uuid.New(),
					Type:      "test_user",
					Payload:   MockUser{Name: "test"},
					Meta:      MockMeta{Foo: true},
					CreatedAt: time.Now(),
					ExpiresAt: time.Now(),
				}, nil
			},
		)

		assert.NoError(t, err)
		assert.NotNil(t, session)

		assert.Equal(t, "test_user", session.Type)
		assert.Equal(t, "test", session.Payload.Name)
		assert.Equal(t, true, session.Meta.Foo)

		_, err = w.Write([]byte("Login Successful"))
		assert.NoError(t, err)
	}))
	defer oauthLoginServer.Close()

	resp, err := http.PostForm(oauthLoginServer.URL, url.Values{
		"state": {"state"},
		"code":  {"a_test_code"},
	})

	assert.NoError(t, err)
	assert.Equal(t, http.StatusOK, resp.StatusCode)

	body, _ := io.ReadAll(resp.Body)
	assert.Equal(t, "Login Successful", string(body))
}

func TestSetSession(t *testing.T) {
	w := httptest.NewRecorder()

	session := &persistence.Session[MockUser, MockMeta]{
		ID:        uuid.New(),
		ExpiresAt: time.Now().Add(time.Hour),
	}
	SetSession(w, "test", session)

	cookies := w.Result().Cookies()
	assert.Len(t, cookies, 1)

	cookie := cookies[0]
	assert.Equal(t, "test", cookie.Name)
	assert.Equal(t, session.ID.String(), cookie.Value)
	assert.Equal(t, session.ExpiresAt.UTC().Truncate(time.Second), cookie.Expires.UTC().Truncate(time.Second))
}

func TestClearSession(t *testing.T) {
	w := httptest.NewRecorder()

	SetSession(w, "test", &persistence.Session[MockUser, MockMeta]{
		ID:        uuid.New(),
		ExpiresAt: time.Now().Add(time.Hour),
	})
	ClearSession(w, "test")
	cookies := w.Result().Cookies()

	assert.Len(t, cookies, 2)
	assert.Equal(t, "test", cookies[1].Name)
	assert.Equal(t, "", cookies[1].Value)
	assert.True(t, cookies[1].Expires.Before(time.Now()))
}

// mockOAuthServer returns a mock OAuth2 server that returns a valid token for the code "a_test_code".
func mockOAuthServer(t *testing.T) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		code := r.FormValue("code")
		assert.Equal(t, "a_test_code", code)

		token := oauth2.Token{
			AccessToken:  "a_valid_access_token",
			RefreshToken: "a_valid_refresh_token",
			TokenType:    "Bearer",
			Expiry:       time.Now().Add(time.Hour),
		}

		tokenJson, err := json.Marshal(token)
		assert.NoError(t, err)

		w.Header().Set("Content-Type", "application/json")
		_, err = w.Write(tokenJson)
		assert.NoError(t, err)
	}))
}
