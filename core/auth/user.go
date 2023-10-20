package auth

import (
	"context"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/core/persistence"
	"time"
)

const UserRepositoryName = "UserRepository"

// User is the user entity.
type User struct {
	Id        uuid.UUID
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

	FindByEmail(email string, ctx context.Context) (*User, error)  // GetByEmail returns a user by email. Returns NotFoundError if no user was found.
	FindById(id uuid.UUID, ctx context.Context) (*User, error)     // GetById returns a user by id. Returns NotFoundError if no user was found.
	Create(user *UserToCreate, ctx context.Context) (*User, error) // Create creates a new user and returns it. Returns InsertError if the user could not be created.
	Delete(id uuid.UUID, ctx context.Context) error                // Delete deletes a user by id. Returns DeleteError if the user could not be deleted.
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *pgxpool.Pool) UserRepository {
	return &PGUserRepository{db: db}
}

func (r *PGUserRepository) RepositoryName() string {
	return UserRepositoryName
}

// FindByEmail returns a user by email. Returns NotFoundError if no user was found.
func (r *PGUserRepository) FindByEmail(email string, ctx context.Context) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(ctx, "SELECT id, email, firstname, lastname, created_at, updated_at FROM users WHERE email = $1", email).
		Scan(&user.Id, &user.Email, &user.Firstname, &user.Lastname, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, persistence.NotFoundError
		}

		return nil, fmt.Errorf("%w: %w", persistence.ReadRowError, err)
	}

	return user, nil
}

// FindById returns a user by id. Returns NotFoundError if no user was found.
func (r *PGUserRepository) FindById(id uuid.UUID, ctx context.Context) (*User, error) {
	user := &User{}
	err := r.db.QueryRow(ctx, "SELECT id, email, firstname, lastname, created_at, updated_at FROM users WHERE id = $1", id).
		Scan(&user.Id, &user.Email, &user.Firstname, &user.Lastname, &user.CreatedAt, &user.UpdatedAt)

	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, persistence.NotFoundError
		}

		return nil, fmt.Errorf("%w: %w", persistence.ReadRowError, err)
	}

	return user, nil
}

// Create creates a new user and return it. CreatedAt and ID are set.
// Returns InsertError if the user could not be created.
func (r *PGUserRepository) Create(user *UserToCreate, ctx context.Context) (*User, error) {
	newUser := &User{
		Id:        uuid.New(),
		Email:     user.Email,
		Firstname: user.Firstname,
		Lastname:  user.Lastname,
		CreatedAt: time.Now(),
	}

	_, err := r.db.Exec(
		ctx,
		"INSERT INTO users (id, email, firstname, lastname, created_at, updated_at) VALUES ($1, $2, $3, $4, $5, $6)",
		newUser.Id, newUser.Email, newUser.Firstname, newUser.Lastname, newUser.CreatedAt, newUser.UpdatedAt,
	)

	if err != nil {
		return nil, fmt.Errorf("%w: %w", persistence.InsertError, err)
	}

	return newUser, nil
}

// Delete deletes a user by id.
// Returns DeleteError if the user could not be deleted.
func (r *PGUserRepository) Delete(id uuid.UUID, ctx context.Context) error {
	_, err := r.db.Exec(ctx, "DELETE FROM users WHERE id = $1", id)
	if err != nil {
		return fmt.Errorf("%w: %w", persistence.DeleteError, err)
	}

	return nil
}
