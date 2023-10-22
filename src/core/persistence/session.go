package persistence

import (
	"context"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"time"
)

// Session is a generic session entity with a payload and metainformation.
// It can be saved in the SessionStore which is a key-value store.
type Session[P any, M any] struct {
	ID        uuid.UUID
	Type      string
	Payload   P
	Meta      M
	CreatedAt time.Time
	ExpiresAt time.Time
	UpdatedAt *time.Time
}

// SessionStore is a key-value store for sessions.
// It adds the Insert method to the KVStore, knowing that the key is an uuid.UUID Insert inserts a new session into the store.
type SessionStore[V any] interface {
	KVStore[uuid.UUID, V]

	// Insert inserts a new session into the store. The key of the session is an uuid.UUID.
	// It is not returned and can instead be accessed via the session's ID field.
	Insert(ctx context.Context, value V) error
}

// SessionRepository combines the Repository and SessionStore interfaces.
// Thereby, a repository for storing sessions is created.
type SessionRepository[V any] interface {
	Repository
	SessionStore[V]
}

// PGReadValidSession reads a session from the database into the session parameter by the key (id).
// If the session has expired it will delete the session from the database and return a persistence.ErrSessionExpired.
func PGReadValidSession[P any, M any](ctx context.Context, db *pgxpool.Pool, key uuid.UUID, session *Session[P, M]) error {
	err := PGReadSession(ctx, db, key, session)
	if err != nil {
		return err
	}

	valid := IsValidSession(session)
	if !valid {
		err := PGDeleteSession(ctx, db, key)
		if err != nil {
			return err
		}

		return ErrSessionExpired
	}

	return nil
}

// PGReadSession reads a session from the database into the session parameter by the key (id).
// It does not check if the session has expired.
func PGReadSession[P any, M any](ctx context.Context, db *pgxpool.Pool, key uuid.UUID, session *Session[P, M]) error {
	return db.QueryRow(ctx, "SELECT id, type, payload, meta, created_at, expires_at, updated_at FROM sessions WHERE id = $1", key).
		Scan(&session.ID, &session.Type, &session.Payload, &session.Meta, &session.CreatedAt, &session.ExpiresAt, &session.UpdatedAt)
}

// PGWriteSession writes a session to the database.
// If insert is true, it will insert the session into the database. Otherwise, it will update the session in the database.
// Upon update, it will also set the updated_at field to the current time.
func PGWriteSession[P any, M any](ctx context.Context, db *pgxpool.Pool, session *Session[P, M]) error {
	return db.QueryRow(
		ctx,
		`INSERT INTO sessions (id, type, payload, meta, created_at, expires_at) 
         VALUES ($1, $2, $3, $4, $5, $6)
         ON CONFLICT (id) 
         DO UPDATE SET 
            type = excluded.type, 
            payload = excluded.payload, 
            meta = excluded.meta, 
            created_at = excluded.created_at, 
            expires_at = excluded.expires_at, 
            updated_at = NOW()
         RETURNING updated_at`,
		session.ID, session.Type, session.Payload, session.Meta, session.CreatedAt, session.ExpiresAt,
	).Scan(&session.UpdatedAt)
}

// PGDeleteSession deletes a session from the database by the key (id).
func PGDeleteSession(ctx context.Context, db *pgxpool.Pool, key uuid.UUID) error {
	_, err := db.Exec(ctx, "DELETE FROM sessions WHERE id = $1", key)

	return err
}

// IsValidSession checks if a session has expired.
func IsValidSession[P, M any](session *Session[P, M]) bool {
	return session.ExpiresAt.After(time.Now())
}
