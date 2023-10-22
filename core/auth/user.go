package auth

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/core/persistence"
	"github.com/org-harmony/harmony/core/util"
	"net/http"
	"time"
)

const UserRepositoryName = "UserRepository"
const UserContextKey = "harmony-app-user"

var (
	ErrNotInContext = errors.New("user not in context")
)

// User is the user entity.
// The User is also part of the UserSession which is stored in the session store.
// The UserSession.ID is stored in a cookie on the client the default session store is the PGUserSessionRepository.
//
// Important: The user information is not loaded upon each request. Instead, the user information is loaded once and stored in the session.
// This means that if the user information changes, the session needs to be updated or deleted. The user session might become stale.
type User struct {
	ID        uuid.UUID
	Email     string
	Firstname string
	Lastname  string
	CreatedAt time.Time
	UpdatedAt *time.Time
}

// UserToCreate is the user entity without the id and dates.
// This user can be passed to the UserRepository.Create method.
type UserToCreate struct {
	Email     string
	Firstname string
	Lastname  string
}

// PGUserRepository is the Postgres implementation of the UserRepository interface.
type PGUserRepository struct {
	db *pgxpool.Pool
}

// UserRepository is the interface for the user repository.
type UserRepository interface {
	persistence.Repository

	FindByEmail(email string, ctx context.Context) (*User, error)  // FindByEmail returns a user by email. Returns ErrNotFound if no user was found.
	FindByID(id uuid.UUID, ctx context.Context) (*User, error)     // FindByID returns a user by id. Returns ErrNotFound if no user was found.
	Create(user *UserToCreate, ctx context.Context) (*User, error) // Create creates a new user and returns it. Returns ErrInsert if the user could not be created.
	Delete(id uuid.UUID, ctx context.Context) error                // Delete deletes a user by id. Returns ErrDelete if the user could not be deleted.
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &PGUserRepository{db: db}
}

// RepositoryName returns the name of the repository.
func (r *PGUserRepository) RepositoryName() string {
	return UserRepositoryName
}

// FindByEmail returns a user by email. Returns ErrNotFound if no user was found.
func (r *PGUserRepository) FindByEmail(email string, ctx context.Context) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(ctx, "SELECT id, email, firstname, lastname, created_at, updated_at FROM users WHERE email = $1", email).
		Scan(&user.ID, &user.Email, &user.Firstname, &user.Lastname, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, persistence.PGReadErr(err)
	}

	return user, nil
}

// FindByID returns a user by id. Returns ErrNotFound if no user was found.
func (r *PGUserRepository) FindByID(id uuid.UUID, ctx context.Context) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(ctx, "SELECT id, email, firstname, lastname, created_at, updated_at FROM users WHERE id = $1", id).
		Scan(&user.ID, &user.Email, &user.Firstname, &user.Lastname, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, persistence.PGReadErr(err)
	}

	return user, nil
}

// Create creates a new user and return it. CreatedAt and ID are set.
// Returns ErrInsert if the user could not be created.
func (r *PGUserRepository) Create(user *UserToCreate, ctx context.Context) (*User, error) {
	newUser := &User{
		ID:        uuid.New(),
		Email:     user.Email,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
		CreatedAt: time.Now(),
	}

	_, err := r.db.Exec(
		ctx,
		"INSERT INTO users (id, email, firstname, lastname, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
		newUser.ID, newUser.Email, newUser.Firstname, newUser.Lastname, newUser.CreatedAt, newUser.UpdatedAt,
	)

	if err != nil {
		return nil, util.ErrErr(persistence.ErrInsert, err)
	}

	return newUser, nil
}

// Delete deletes a user by id.
// Returns ErrDelete if the user could not be deleted.
func (r *PGUserRepository) Delete(id uuid.UUID, ctx context.Context) error {
	_, err := r.db.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return util.ErrErr(persistence.ErrDelete, err)
	}

	return nil
}

// CtxUser returns the user from the context. It will return ErrNotInContext if the user is not in the context.
// This is ideally paired with the auth.Middleware.
func CtxUser(ctx context.Context) (*User, error) {
	user := ctx.Value(UserContextKey)
	if user == nil {
		return nil, ErrNotInContext
	}

	u, ok := user.(*User)
	if !ok || u == nil {
		return nil, ErrNotInContext
	}

	return u, nil
}

// TODO add auto-refresh of sessions -> soft and hard expiry
// TODO add refresh token for remember me functionality
// TODO add logout everywhere functionality -> delete all sessions for a user

// login logs the user in and returns the session.
func login(ctx context.Context, user *User, sessionStore UserSessionRepository) (*UserSession, error) {
	session := NewUserSession(user, time.Hour)
	err := sessionStore.Insert(ctx, session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// setSession sets the user session cookie on the response.
// The session id is used as the cookie value.
// The session expires at the time of the session.
func setSession(w http.ResponseWriter, session *UserSession) {
	http.SetCookie(w, &http.Cookie{
		Name:     UserSessionCookieName,
		Value:    session.ID.String(),
		Expires:  session.ExpiresAt,
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	})
}

// clearSession clears the user session cookie on the response.
func clearSession(w http.ResponseWriter) {
	http.SetCookie(w, &http.Cookie{
		Name:     UserSessionCookieName,
		Value:    "",
		Expires:  time.Now(),
		SameSite: http.SameSiteStrictMode,
		Path:     "/",
		Secure:   true,
		HttpOnly: true,
	})
}
