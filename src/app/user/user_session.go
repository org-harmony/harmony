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

type SessionMeta struct{}

type PGUserSessionRepository struct {
	db *pgxpool.Pool
}

type SessionRepository interface {
	persistence.SessionRepository[*Session]
}

func NewPGUserSessionRepository(db *pgxpool.Pool) SessionRepository {
	return &PGUserSessionRepository{db: db}
}

func (r *PGUserSessionRepository) RepositoryName() string {
	return SessionRepositoryName
}

// Read reads a valid user session from the database by id.
// If the session has expired it will be deleted and the Read returns a persistence.ErrSessionExpired.
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

// Write writes a user session to the database, identified by the id passed in *not* the session's id on the struct.
// The session struct's id will be overwritten by the id passed as second argument to PGUserSessionRepository.Write.
func (r *PGUserSessionRepository) Write(ctx context.Context, id uuid.UUID, session *Session) error {
	session.ID = id

	err := persistence.PGWriteSession(ctx, r.db, &session.Session)
	if err == nil {
		return nil
	}

	return errors.Join(persistence.ErrInsert, err)
}

func (r *PGUserSessionRepository) Delete(ctx context.Context, id uuid.UUID) error {
	err := persistence.PGDeleteSession(ctx, r.db, id)

	if err == nil {
		return nil
	}

	return errors.Join(persistence.ErrDelete, err)
}

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
