package auth

import (
	"context"
	"github.com/google/uuid"
	"net/http"
)

// UserMiddlewareOptions define possible options for UserMiddleware they should be set through UserMiddlewareOption.
type UserMiddlewareOptions struct {
	requireAuth        bool
	notLoggedInHandler http.Handler
	sessionStore       UserSessionRepository
}

// UserMiddlewareOption describes a function modifying the UserMiddlewareOptions.
type UserMiddlewareOption func(*UserMiddlewareOptions)

// AllowAnonymous lets not logged-in users pass the middleware.
// Using the CtxUser function will return an error and a nil-user in this case.
func AllowAnonymous(o *UserMiddlewareOptions) {
	o.requireAuth = false
}

// NotLoggedInHandler sets the handler to be called when a user is not logged in and the middleware requires it.
// With a high likelihood you want to: RedirectToLogin.
//
// Example:
//
//	router.With(auth.Middleware(sessionStore, auth.NotLoggedInHandler(auth.RedirectToLogin))).Get("/some/route", someHandler)
func NotLoggedInHandler(h func(w http.ResponseWriter, r *http.Request)) UserMiddlewareOption {
	return func(o *UserMiddlewareOptions) {
		o.notLoggedInHandler = http.HandlerFunc(h)
	}
}

// Middleware is the auth middleware that checks if a user is logged in and sets the user in the request context.
// If the user is not logged in and the middleware requires it, the NotLoggedInHandler is called.
// Then it should be safe to use the CtxUser function without it returning an error.
//
// If it is required for anonymous users to pass the middleware, use the AllowAnonymous option.
//
// Example:
//
//	router.With(auth.Middleware(sessionStore)).Get("/some/route", someHandler)
func Middleware(sessionStore UserSessionRepository, opts ...UserMiddlewareOption) func(next http.Handler) http.Handler {
	m := defaultUserMiddlewareOptions(sessionStore)
	for _, opt := range opts {
		opt(m)
	}

	return func(next http.Handler) http.Handler {
		f := func(w http.ResponseWriter, r *http.Request) {
			user, err := LoggedInUser(r, m.sessionStore)
			if err != nil && m.requireAuth {
				m.notLoggedInHandler.ServeHTTP(w, r)
				return
			}

			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			withUser := context.WithValue(r.Context(), UserContextKey, user)
			r = r.WithContext(withUser)

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(f)
	}
}

// LoggedInUser reads the session id from the request reads the user from the passed in session store and returns it.
// If the user is not logged in, an error is returned.
//
// Important: The function does not look the user up in the database. It simply returns the user from the session.
func LoggedInUser(r *http.Request, sessionStore UserSessionRepository) (*User, error) {
	userSession, err := UserSessionFromRequest(r, sessionStore)
	if err != nil {
		return nil, err
	}

	return &userSession.Payload, nil
}

// UserSessionFromRequest returns the user session from the request.
// If the user is not logged in, an error is returned.
func UserSessionFromRequest(r *http.Request, sessionStore UserSessionRepository) (*UserSession, error) {
	sessionID, err := SessionIDFromRequest(r)
	if err != nil {
		return nil, err
	}

	userSession, err := sessionStore.Read(r.Context(), sessionID)
	if err != nil {
		return nil, err
	}

	return userSession, nil
}

// SessionIDFromRequest returns the session id from the request.
func SessionIDFromRequest(r *http.Request) (uuid.UUID, error) {
	c, err := r.Cookie(UserSessionCookieName)
	if err != nil {
		return uuid.Nil, err
	}

	err = c.Valid()
	if err != nil {
		return uuid.Nil, err
	}

	id, err := uuid.Parse(c.Value)
	if err != nil {
		return uuid.Nil, err
	}

	return id, nil
}

// defaultUserMiddlewareOptions returns the default options for the Middleware.
func defaultUserMiddlewareOptions(sessionStore UserSessionRepository) *UserMiddlewareOptions {
	return &UserMiddlewareOptions{
		requireAuth: true,
		notLoggedInHandler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, "not logged in when required", http.StatusUnauthorized)
		}),
		sessionStore: sessionStore,
	}
}
