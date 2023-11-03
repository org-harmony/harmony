package user

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestMiddleware_LoggedIn(t *testing.T) {
	registerCleanupUserAndSessionTables(t)
	user, session := setupMockUserAndSession(t)

	middleware := Middleware(sessionStore)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := CtxUser(r.Context())
		assert.NoError(t, err)

		assert.Equal(t, user.ID, MustCtxUser(r.Context()).ID)
	})
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: session.ID.String()})
	recorder := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestMiddleware_DefaultNotLoggedInHandler(t *testing.T) {
	registerCleanupUserAndSessionTables(t)
	setupMockUserAndSession(t)

	middleware := Middleware(sessionStore)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Fail(t, "Should not be called")
	})
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusTemporaryRedirect, recorder.Code)
	assert.Equal(t, "/auth/login", recorder.Header().Get("Location"))
}

func TestMiddleware_NotLoggedInHandler(t *testing.T) {
	registerCleanupUserAndSessionTables(t)
	setupMockUserAndSession(t)

	middleware := Middleware(sessionStore, NotLoggedInHandler(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	})))
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Fail(t, "Should not be called")
	})
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: "invalid-session"})
	recorder := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusUnauthorized, recorder.Code)
}

func TestMiddleware_AllowAnonymous(t *testing.T) {
	registerCleanupUserAndSessionTables(t)
	setupMockUserAndSession(t)

	middleware := Middleware(sessionStore, AllowAnonymous)
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := CtxUser(r.Context())
		assert.Error(t, err)
		assert.ErrorIs(t, err, ErrNotInContext)

		assert.Panics(t, func() { MustCtxUser(r.Context()) })
	})
	wrappedHandler := middleware(handler)

	req := httptest.NewRequest("GET", "/", nil)
	recorder := httptest.NewRecorder()

	wrappedHandler.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
}

func setupMockUserAndSession(t *testing.T) (*User, *Session) {
	user, err := userRepo.Create(ctx, fooUserToCreate())
	require.NoError(t, err)

	session := NewUserSession(user, time.Hour)
	err = sessionStore.Insert(ctx, session)
	require.NoError(t, err)

	return user, session
}
