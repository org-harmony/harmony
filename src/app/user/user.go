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
	"time"
)

// RepositoryName is the name of the user repository. It can be used to retrieve the repository from the persistence.RepositoryProvider.
const RepositoryName = "UserRepository"

// ContextKey is the key for the user in the context.
// Example:
//
//	user := ctx.Value(user.ContextKey).(*user.User)
const ContextKey = "harmony-app-user"

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

// ToUpdate is the user entity that can be updated. It contains an immutable id and the fields that can be updated.
type ToUpdate struct {
	id        uuid.UUID `hvalidate:"required"`
	Email     string    `hvalidate:"required,email"`
	Firstname string    `hvalidate:"required"`
	Lastname  string    `hvalidate:"required"`
}

// PGUserRepository is the user repository for postgres. It holds a reference to the database connection pool.
type PGUserRepository struct {
	db *pgxpool.Pool
}

// Repository is the user repository. It contains all methods to interact with the user table in the database.
// Repository is safe for concurrent use by multiple goroutines.
type Repository interface {
	persistence.Repository

	FindByEmail(ctx context.Context, email string) (*User, error) // FindByEmail returns a user by email. Returns ErrNotFound if no user was found.
	FindByID(ctx context.Context, id uuid.UUID) (*User, error)    // FindByID returns a user by id. Returns ErrNotFound if no user was found.
	Create(ctx context.Context, user *ToCreate) (*User, error)    // Create creates a new user and returns it. Returns ErrInsert if the user could not be created.
	Update(ctx context.Context, user *ToUpdate) (*User, error)    // Update updates a user and returns it. Returns ErrUpdate if the user could not be updated.
	Delete(ctx context.Context, id uuid.UUID) error               // Delete deletes a user by id. Returns ErrDelete if the user could not be deleted.
}

// ToUpdate transform the user to a ToUpdate.
func (u *User) ToUpdate() *ToUpdate {
	return &ToUpdate{
		id:        u.ID,
		Email:     u.Email,
		Firstname: u.Firstname,
		Lastname:  u.Lastname,
	}
}

// ID on to update returns the id of the user. It needs to be retrieved from a getter because the underlying value is immutable.
func (u *ToUpdate) ID() uuid.UUID {
	return u.id
}

// NewUserRepository constructs a new PGUserRepository with the passed in database connection pool./
func NewUserRepository(db *pgxpool.Pool) Repository {
	return &PGUserRepository{db: db}
}

// RepositoryName returns the name of the repository. This name is used to identify the repository in the persistence.RepositoryProvider.
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

// Update updates a user and returns it. Returns ErrUpdate if the user could not be updated.
// UpdatedAt is set.
func (r *PGUserRepository) Update(ctx context.Context, user *ToUpdate) (*User, error) {
	updateUser := &User{
		ID: user.ID(),
	}

	err := r.db.QueryRow(
		ctx,
		`UPDATE users 
		SET email = $1, firstname = $2, lastname = $3, updated_at = NOW() 
		WHERE id = $4 
		RETURNING email, firstname, lastname, created_at, updated_at`,
		user.Email, user.Firstname, user.Lastname, user.ID(),
	).Scan(&updateUser.Email, &updateUser.Firstname, &updateUser.Lastname, &updateUser.CreatedAt, &updateUser.UpdatedAt)

	if err != nil {
		return nil, errors.Join(persistence.ErrUpdate, err)
	}

	return updateUser, nil
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

// Login creates a new user session and stores it in the session store.
// Thereby, the user will be detected as logged in from the application.
func Login(ctx context.Context, user *User, sessionStore SessionRepository) (*Session, error) {
	session := NewUserSession(user, time.Hour)
	err := sessionStore.Insert(ctx, session)
	if err != nil {
		return nil, err
	}

	return session, nil
}
