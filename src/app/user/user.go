package user

// TODO add auto-refresh of sessions -> soft and hard expiry
// TODO add refresh token for remember me functionality
// TODO add logout everywhere functionality -> delete all sessions for a user
// TODO remove expired sessions from database job

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/web"
	"time"
)

const RepositoryName = "Repository"
const ContextKey = "harmony-app-user"

type TemplateData[T any] struct {
	User *User

	*web.BaseTemplateData[T]
}

// User is the user entity.
// The User is also part of the Session which is stored in the session store.
// The Session.ID is stored in a cookie on the client the default session store is the PGUserSessionRepository.
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

// ToCreate is the user entity without the id and dates.
// This user can be passed to the Repository.Create method.
type ToCreate struct {
	Email     string `hvalidate:"required,email"`
	Firstname string `hvalidate:"required"`
	Lastname  string `hvalidate:"required"`
}

// ToUpdate is the user entity that can be updated.
type ToUpdate struct {
	Email     string `hvalidate:"required,email"`
	Firstname string `hvalidate:"required"`
	Lastname  string `hvalidate:"required"`
}

type PGUserRepository struct {
	db *pgxpool.Pool
}

type Repository interface {
	persistence.Repository

	FindByEmail(ctx context.Context, email string) (*User, error) // FindByEmail returns a user by email. Returns ErrNotFound if no user was found.
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)    // FindByID returns a user by id. Returns ErrNotFound if no user was found.
	Create(ctx context.Context, user *ToCreate) (*User, error)    // Create creates a new user and returns it. Returns ErrInsert if the user could not be created.
	Delete(ctx context.Context, id uuid.UUID) error               // Delete deletes a user by id. Returns ErrDelete if the user could not be deleted.
}

func NewTemplateData[T any](user *User, data T) *TemplateData[T] {
	return &TemplateData[T]{User: user, BaseTemplateData: web.NewTemplateData(data)}
}

func NewUserRepository(db *pgxpool.Pool) Repository {
	return &PGUserRepository{db: db}
}

func (r *PGUserRepository) RepositoryName() string {
	return RepositoryName
}

// FindByEmail returns a user by email. Returns ErrNotFound if no user was found.
func (r *PGUserRepository) FindByEmail(ctx context.Context, email string) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(ctx, "SELECT id, email, firstname, lastname, created_at, updated_at FROM users WHERE email = $1", email).
		Scan(&user.ID, &user.Email, &user.Firstname, &user.Lastname, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, persistence.PGReadErr(err)
	}

	return user, nil
}

// FindByID returns a user by id. Returns ErrNotFound if no user was found.
func (r *PGUserRepository) FindByID(ctx context.Context, id uuid.UUID) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(ctx, "SELECT id, email, firstname, lastname, created_at, updated_at FROM users WHERE id = $1", id).
		Scan(&user.ID, &user.Email, &user.Firstname, &user.Lastname, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		return nil, persistence.PGReadErr(err)
	}

	return user, nil
}

// Create creates a new user and return it. CreatedAt and id are set.
// Returns ErrInsert if the user could not be created.
func (r *PGUserRepository) Create(ctx context.Context, user *ToCreate) (*User, error) {
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
		return nil, errors.Join(persistence.ErrInsert, err)
	}

	return newUser, nil
}

// Delete deletes a user by id.
// Returns ErrDelete if the user could not be deleted.
func (r *PGUserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	_, err := r.db.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return errors.Join(persistence.ErrDelete, err)
	}

	return nil
}

func Login(ctx context.Context, user *User, sessionStore SessionRepository) (*Session, error) {
	session := NewUserSession(user, time.Hour)
	err := sessionStore.Insert(ctx, session)
	if err != nil {
		return nil, err
	}

	return session, nil
}

// ToUpdate transform the user to a ToUpdate.
func (u *User) ToUpdate() *ToUpdate {
	return &ToUpdate{
		Email:     u.Email,
		Firstname: u.Firstname,
		Lastname:  u.Lastname,
	}
}
