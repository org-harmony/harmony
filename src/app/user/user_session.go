package user

import (
	"context"
	"errors"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/util"
	"time"
)

const (
	SessionRepositoryName = "UserSessionRepository"
	SessionCookieName     = "harmony_session"
	SessionType           = "user"
)

// Session is a persistence.Session with the User as the payload and SessionMeta as the meta.
type Session struct {
	persistence.Session[User, SessionMeta]
}

// SessionMeta is the meta for a user session. It contains extra settings for the user session and the first login time.
// FirstLoginAt allows for soft-/hard-expiry of user sessions.
type SessionMeta struct {
	Settings     map[string]string
	FirstLoginAt time.Time
	ExtendedAt   *time.Time
}

// PGUserSessionRepository is a PostgreSQL implementation of the SessionRepository interface for user sessions.
// It implements the SessionRepository interface and by that the persistence.SessionRepository interface.
// For more details see the SessionRepository interface.
type PGUserSessionRepository struct {
	db *pgxpool.Pool
}

// SessionRepository allows to interface with user sessions in the database. Concrete implementations provide the database access.
// In general the SessionRepository inherits from the persistence.SessionRepository interface.
// Thus, it is a persistence.SessionStore (persistence.KVStore + Insert method).
// It allows to read, write and delete user sessions from the database.
// Insert should usually be preferred over Write as it does not require the id to be passed.
// Write can be used to insert new items but also to update existing ones (upsert).
type SessionRepository interface {
	persistence.SessionRepository[*Session]
}

// NewPGUserSessionRepository creates a new PGUserSessionRepository with the given database connection pool.
func NewPGUserSessionRepository(db *pgxpool.Pool) SessionRepository {
	return &PGUserSessionRepository{db: db}
}

// RepositoryName returns the name of the repository. It is used to register the repository in the application context.
func (r *PGUserSessionRepository) RepositoryName() string {
	return SessionRepositoryName
}

// Read reads a valid/invalid user session from the database by id.
// If the session has expired it will still be returned and no error will be returned.
func (r *PGUserSessionRepository) Read(ctx context.Context, id uuid.UUID) (*Session, error) {
	session := &Session{
		Session: persistence.Session[User, SessionMeta]{},
	}

	err := persistence.PGReadSession(ctx, r.db, id, &session.Session)
	if err != nil {
		return nil, persistence.PGReadErr(err)
	}

	return session, nil
}

// Write writes a user session to the database, identified by the id passed in *not* the session's id on the struct.
// The session structs id will be overwritten by the id passed as second argument to PGUserSessionRepository.Write.
func (r *PGUserSessionRepository) Write(ctx context.Context, id uuid.UUID, session *Session) error {
	session.ID = id

	err := persistence.PGWriteSession(ctx, r.db, &session.Session)
	if err == nil {
		return nil
	}

	return errors.Join(persistence.ErrInsert, err)
}

// Delete deletes a user session from the database by id. If the session does not exist it returns nil.
// If the session could not be deleted it returns persistence.ErrDelete.
func (r *PGUserSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := persistence.PGDeleteSession(ctx, r.db, id)

	if err == nil {
		return nil
	}

	return errors.Join(persistence.ErrDelete, err)
}

// Insert inserts a new user session into the database. A new uuid.UUID will be generated and set on the session struct.
// Therefore, Insert has a side effect on the session struct. Insert should be preferred over Write for new sessions.
// If the session could not be inserted it returns persistence.ErrInsert.
func (r *PGUserSessionRepository) Insert(ctx context.Context, session *Session) error {
	id := uuid.New()
	session.ID = id

	err := persistence.PGWriteSession(ctx, r.db, &session.Session)
	if err != nil {
		return errors.Join(persistence.ErrInsert, err)
	}

	return nil
}

// SessionStore returns the user session store from the application context.
// It panics if the user session store is not registered in the application context.
// Thus, it should only be used after the application context has been initialized.
func SessionStore(app *hctx.AppCtx) SessionRepository {
	return util.UnwrapType[SessionRepository](app.Repository(SessionRepositoryName))
}

// NewUserSession creates a new user session with the given user that expires now + duration.
// The SessionMeta.FirstLoginAt will be set to the current time.
// The id will be set to a zero uuid.UUID value.
func NewUserSession(user *User, duration time.Duration) *Session {
	return &Session{
		Session: persistence.Session[User, SessionMeta]{
			Type:    SessionType,
			Payload: *user,
			Meta: SessionMeta{
				FirstLoginAt: time.Now(),
			},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(duration),
		},
	}
}

// IsHardExpired checks if a session has hard expired. This means the session has expired more than 24 hours ago or
// the session has expired and not been extended within the last 3 hours or the first login was more than 24 hours ago.
func (s *Session) IsHardExpired() bool {
	if !s.IsExpired() {
		return false
	}

	olderThan24h := s.ExpiresAt.Before(time.Now().Add(-24 * time.Hour))
	if olderThan24h {
		return true
	}

	firstLoginOlderThan24h := s.Meta.FirstLoginAt.Before(time.Now().Add(-24 * time.Hour))
	if firstLoginOlderThan24h {
		return true
	}

	createdInTime := s.CreatedAt.After(time.Now().Add(-3 * time.Hour))
	extendedInTime := s.Meta.ExtendedAt != nil && s.Meta.ExtendedAt.After(time.Now().Add(-3*time.Hour))
	if s.IsExpired() && !createdInTime && !extendedInTime {
		return true
	}

	return false
}
