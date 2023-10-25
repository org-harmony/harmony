package user

import (
	"context"
	"github.com/google/uuid"
	"github.com/org-harmony/harmony/src/core/util"
	"net/http"
)

// MiddlewareOptions define possible options for Middleware they should be set through MiddlewareOption.
type MiddlewareOptions struct {
	requireAuth        bool
	notLoggedInHandler http.Handler
	sessionStore       SessionRepository
}

// MiddlewareOption describes a function modifying the MiddlewareOptions.
type MiddlewareOption func(*MiddlewareOptions)

// RedirectToLogin redirects the user to the login page.
// This is the default NotLoggedInHandler.
func RedirectToLogin(w http.ResponseWriter, r *http.Request) {
	http.Redirect(w, r, "/auth/login", http.StatusTemporaryRedirect)
}

// AllowAnonymous lets not logged-in users pass the middleware.
// Using the CtxUser function will return an error and a nil-user in this case.
func AllowAnonymous(o *MiddlewareOptions) {
	o.requireAuth = false
}

// NotLoggedInHandler sets the handler to be called when a user is not logged in and the middleware requires it.
// The default handler is RedirectToLogin.
//
// Example:
//
//	router.With(auth.Middleware(sessionStore, auth.NotLoggedInHandler(auth.RedirectToLogin))).Get("/some/route", someHandler)
func NotLoggedInHandler(h func(w http.ResponseWriter, r *http.Request)) MiddlewareOption {
	return func(o *MiddlewareOptions) {
		o.notLoggedInHandler = http.HandlerFunc(h)
	}
}

// Middleware is the auth middleware that checks if a user is logged in and sets the user in the request context.
// If the user is not logged in and the middleware requires it, the NotLoggedInHandler is called (defaults to RedirectToLogin).
// Then it should be safe to use the CtxUser function without it returning an error.
//
// If it is required for anonymous users to pass the middleware, use the AllowAnonymous option.
//
// Example:
//
//	router.With(user.Middleware(sessionStore)).Get("/some/route", someHandler)
func Middleware(sessionStore SessionRepository, opts ...MiddlewareOption) func(next http.Handler) http.Handler {
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

			withUser := context.WithValue(r.Context(), ContextKey, user)
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
func LoggedInUser(r *http.Request, sessionStore SessionRepository) (*User, error) {
	userSession, err := SessionFromRequest(r, sessionStore)
	if err != nil {
		return nil, err
	}

	return &userSession.Payload, nil
}

// SessionFromRequest returns the user session from the request.
// If the user is not logged in, an error is returned.
func SessionFromRequest(r *http.Request, sessionStore SessionRepository) (*Session, error) {
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

// SessionIDFromRequest returns the session id for the SessionCookieName from the request.
func SessionIDFromRequest(r *http.Request) (uuid.UUID, error) {
	c, err := r.Cookie(SessionCookieName)
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

// CtxUser returns the user from the context. It will return ErrNotInContext if the user is not in the context.
// This is ideally paired with the user.Middleware which sets the user in the context with the key user.ContextKey.
// CtxUser looks for the user.ContextKey in the context.
func CtxUser(ctx context.Context) (*User, error) {
	u, ok := util.CtxValue[*User](ctx, ContextKey)
	if !ok {
		return nil, ErrNotInContext
	}

	return u, nil
}

// defaultUserMiddlewareOptions returns the default options for the Middleware.
func defaultUserMiddlewareOptions(sessionStore SessionRepository) *MiddlewareOptions {
	return &MiddlewareOptions{
		requireAuth:        true,
		notLoggedInHandler: http.HandlerFunc(RedirectToLogin),
		sessionStore:       sessionStore,
	}
}
