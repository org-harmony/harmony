package user

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/org-harmony/harmony/src/core/auth"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/trace"
	"github.com/org-harmony/harmony/src/core/util"
	"net/http"
	"time"
)

const MiddlewarePkg = "user.middleware"

// MiddlewareOptions define possible options for Middleware they should be set through MiddlewareOption.
type MiddlewareOptions struct {
	requireAuth        bool
	notLoggedInHandler http.Handler
	sessionStore       SessionRepository
	userRepository     Repository
	logger             trace.Logger
}

// ErrNotInContext is returned by the CtxUser function if the user is not in the context.
var ErrNotInContext = errors.New("user not in context")

// MiddlewareOption modifies MiddlewareOptions and is used to set options for the Middleware.
type MiddlewareOption func(*MiddlewareOptions)

// RedirectToLogin redirects the user to the login page.
// This is the default NotLoggedInHandler.
// TODO add a cookie to redirect the user back to the page he was on before logging-in.
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
//	router.With(user.Middleware(sessionStore, user.NotLoggedInHandler(auth.RedirectToLogin))).Get("/some/route", someHandler)
func NotLoggedInHandler(h func(w http.ResponseWriter, r *http.Request)) MiddlewareOption {
	return func(o *MiddlewareOptions) {
		o.notLoggedInHandler = http.HandlerFunc(h)
	}
}

// AlwaysFetchUser sets the middleware to always fetch the user from the database.
// This option ensures that the user in the context is always up-to-date,
// but this comes at the cost of a database query per request from a seemingly logged-in user.
func AlwaysFetchUser(repository Repository) MiddlewareOption {
	return func(o *MiddlewareOptions) {
		o.userRepository = repository
	}
}

// WithLogger sets the middleware to use the passed in logger. The default logger will be created by trace.NewLogger.
func WithLogger(logger trace.Logger) MiddlewareOption {
	return func(o *MiddlewareOptions) {
		o.logger = logger
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

			if m.userRepository != nil {
				user, err = m.userRepository.FindByID(r.Context(), user.ID)
				if err != nil {
					m.handleUserNotFound(w, r, err)
					return
				}
			}

			withUser := context.WithValue(r.Context(), ContextKey, user)
			r = r.WithContext(withUser)

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(f)
	}
}

// LoggedInMiddleware is a convenience function that creates a middleware with the AlwaysFetchUser and WithLogger options.
// It looks all the required dependencies up in the passed-in hctx.AppCtx. Extra options can be passed in.
func LoggedInMiddleware(appCtx *hctx.AppCtx, opts ...MiddlewareOption) func(next http.Handler) http.Handler {
	userRepository := util.UnwrapType[Repository](appCtx.Repository(RepositoryName))
	opts = append(opts, AlwaysFetchUser(userRepository), WithLogger(appCtx.Logger))

	return Middleware(SessionStore(appCtx), opts...)
}

// LoggedInUser reads the session id from the request, reads the user from the passed in session store and returns it.
// If the user is not logged in, an error is returned.
//
// Important: The function does not look the user up in the database. It simply returns the user from the session.
func LoggedInUser(r *http.Request, sessionStore SessionRepository) (*User, error) {
	userSession, err := SessionFromRequest(r, sessionStore)
	if err != nil {
		return nil, err
	}

	if userSession.IsExpired() {
		err = TryExtendSession(r.Context(), userSession, time.Hour, sessionStore)
		if err != nil && !errors.Is(err, ErrHardSessionExpiry) {
			return nil, err
		}

		if err != nil && errors.Is(err, ErrHardSessionExpiry) {
			err = sessionStore.Delete(r.Context(), userSession.ID)
			return nil, err
		}
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

// MustCtxUser returns the user from the context. It will panic if the user is not in the context.
// It calls CtxUser internally. It is safe to call this function if the user is required to be logged in for the route.
func MustCtxUser(ctx context.Context) *User {
	u, err := CtxUser(ctx)
	if err != nil {
		panic(err)
	}

	return u
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

func defaultUserMiddlewareOptions(sessionStore SessionRepository) *MiddlewareOptions {
	return &MiddlewareOptions{
		requireAuth:        true,
		notLoggedInHandler: http.HandlerFunc(RedirectToLogin),
		sessionStore:       sessionStore,
		logger:             trace.NewLogger(),
	}
}

func (m *MiddlewareOptions) handleUserNotFound(w http.ResponseWriter, r *http.Request, err error) {
	m.logger.Error(MiddlewarePkg, "user not found but session exists", err)

	sessionId, err := SessionIDFromRequest(r)
	if err != nil {
		m.logger.Error(MiddlewarePkg, "failed to get session from id after user was requested", err)
		m.notLoggedInHandler.ServeHTTP(w, r)
		return
	}

	err = m.sessionStore.Delete(r.Context(), sessionId)
	if err != nil {
		m.logger.Error(MiddlewarePkg, "failed to delete session after user was requested", err)
		m.notLoggedInHandler.ServeHTTP(w, r)
		return
	}

	auth.ClearSession(w, SessionCookieName)
	m.notLoggedInHandler.ServeHTTP(w, r)
}
