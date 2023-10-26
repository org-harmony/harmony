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

// Session is the user session entity.
// It is a persistence.Session with the User as the payload and SessionMeta as the meta.
type Session struct {
	persistence.Session[User, SessionMeta]
}

// SessionMeta is the metainformation about a user session.
type SessionMeta struct{}

// PGUserSessionRepository is the Postgres implementation of the SessionRepository interface.
// It allows saving and reading user sessions from the application's Postgres database.
type PGUserSessionRepository struct {
	db *pgxpool.Pool
}

// SessionRepository defines the session store for user sessions.
// It is a persistence.SessionRepository with the Session as the session.
type SessionRepository interface {
	persistence.SessionRepository[*Session]
}

// NewPGUserSessionRepository creates a new PGUserSessionRepository. It requires a Postgres database connection pool.
func NewPGUserSessionRepository(db *pgxpool.Pool) SessionRepository {
	return &PGUserSessionRepository{db: db}
}

// RepositoryName returns the name of the repository.
func (r *PGUserSessionRepository) RepositoryName() string {
	return SessionRepositoryName
}

// Read reads a valid user session from the database by id.
// If the session has expired it will return a persistence.ErrReadRow and a persistence.ErrSessionExpired.
// The invalid session is thereafter deleted from the database.
func (r *PGUserSessionRepository) Read(ctx context.Context, id uuid.UUID) (*Session, error) {
	session := &Session{
		Session: persistence.Session[User, SessionMeta]{},
	}

	err := persistence.PGReadValidSession(ctx, r.db, id, &session.Session)
	if err != nil {
		return nil, persistence.PGReadErr(err)
	}

	return session, nil
}

// Write writes a user session to the database. The session is identified by the id *not* the session id.
// Also, the session id will be overwritten by the id passed as second argument to PGUserSessionRepository.Write.
func (r *PGUserSessionRepository) Write(ctx context.Context, id uuid.UUID, session *Session) error {
	session.ID = id

	err := persistence.PGWriteSession(ctx, r.db, &session.Session)
	if err == nil {
		return nil
	}

	return errors.Join(persistence.ErrInsert, err)
}

// Delete deletes a user session from the database by id.
func (r *PGUserSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := persistence.PGDeleteSession(ctx, r.db, id)

	if err == nil {
		return nil
	}

	return errors.Join(persistence.ErrDelete, err)
}

// Insert inserts a new user session into the database.
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
func NewUserSession(user *User, duration time.Duration) *Session {
	return &Session{
		Session: persistence.Session[User, SessionMeta]{
			Type:      SessionType,
			Payload:   *user,
			Meta:      SessionMeta{},
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(duration),
		},
	}
}
