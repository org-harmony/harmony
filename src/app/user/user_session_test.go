package user

import (
	"github.com/google/uuid"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestNewPGUserSessionRepository(t *testing.T) {
	repo := NewPGUserSessionRepository(nil)
	assert.NotNil(t, repo)
}

func TestNewUserSession(t *testing.T) {
	session := NewUserSession(&User{}, time.Hour)
	assert.NotNil(t, session)
	assert.Empty(t, session.ID)
	assert.NotEmpty(t, session.CreatedAt)
	assert.NotEmpty(t, session.ExpiresAt)
	assert.Equal(t, session.ExpiresAt.Truncate(time.Second), session.CreatedAt.Add(time.Hour).Truncate(time.Second))
	assert.Equal(t, session.Type, SessionType)
	assert.NotNil(t, session.Payload)
	assert.NotNil(t, session.Meta)
}

func TestPGUserSessionRepository_Write(t *testing.T) {
	registerCleanupUserSessionTable(t)
	session := fooUserSession()
	err := sessionStore.Write(ctx, session.ID, session)
	assert.NoError(t, err)
}

func TestPGUserSessionRepository_Read(t *testing.T) {
	registerCleanupUserSessionTable(t)
	session := fooUserSession()
	err := sessionStore.Write(ctx, session.ID, session)
	assert.NoError(t, err)

	readSession, err := sessionStore.Read(ctx, session.ID)
	assert.NoError(t, err)
	assert.NotNil(t, readSession)

	persistence.TruncateSessionDates(&session.Session)
	persistence.TruncateSessionDates(&readSession.Session)

	assert.Equal(t, session, readSession)
}

func TestPGUserSessionRepository_Read_Expired(t *testing.T) {
	registerCleanupUserSessionTable(t)
	session := fooUserSession()
	session.ExpiresAt = time.Now().Add(-time.Hour)
	err := sessionStore.Write(ctx, session.ID, session)
	assert.NoError(t, err)

	readSession, err := sessionStore.Read(ctx, session.ID)
	assert.NoError(t, err)
	assert.NotNil(t, readSession)
}

func TestPGUserSessionRepository_Insert(t *testing.T) {
	registerCleanupUserSessionTable(t)
	session := fooUserSession()
	session.ID = uuid.Nil

	err := sessionStore.Insert(ctx, session)
	assert.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, session.ID) // should be set by the Insert method

	readSession, err := sessionStore.Read(ctx, session.ID)
	assert.NoError(t, err)
	assert.NotNil(t, readSession)

	persistence.TruncateSessionDates(&session.Session)
	persistence.TruncateSessionDates(&readSession.Session)

	assert.Equal(t, session, readSession)
}

func TestPGUserSessionRepository_Delete(t *testing.T) {
	registerCleanupUserSessionTable(t)
	session := fooUserSession()
	err := sessionStore.Write(ctx, session.ID, session)
	assert.NoError(t, err)

	err = sessionStore.Delete(ctx, session.ID)
	assert.NoError(t, err)

	readSession, err := sessionStore.Read(ctx, session.ID)
	assert.Error(t, err)
	assert.ErrorIs(t, err, persistence.ErrNotFound)
	assert.Nil(t, readSession)

	err = sessionStore.Delete(ctx, session.ID)
	assert.NoError(t, err)
}

func TestSession_IsHardExpired(t *testing.T) {
	s := NewUserSession(&User{}, time.Hour)
	assert.False(t, s.IsHardExpired())

	s.ExpiresAt = time.Now().Add(-24 * time.Hour)
	assert.True(t, s.IsHardExpired())

	s.ExpiresAt = time.Now().Add(-23 * time.Hour)
	assert.False(t, s.IsHardExpired())

	s.ExpiresAt = time.Now().Add(24 * time.Hour)
	assert.False(t, s.IsHardExpired())

	s.ExpiresAt = time.Now().Add(-90 * time.Minute)
	s.CreatedAt = time.Now().Add(-8 * time.Hour)
	assert.True(t, s.IsHardExpired())

	extendedAt := time.Now().Add(-2 * time.Hour)
	s.Meta.ExtendedAt = &extendedAt
	assert.False(t, s.IsHardExpired())

	s.Meta.ExtendedAt = nil
	assert.True(t, s.IsHardExpired())

	now := time.Now()
	s.ExpiresAt = now.Add(-23 * time.Hour)
	s.CreatedAt = now.Add(-3 * time.Hour)
	s.Meta.ExtendedAt = &now
	assert.False(t, s.IsHardExpired())

	s.Meta.ExtendedAt = nil
	assert.True(t, s.IsHardExpired())

	s.Meta.FirstLoginAt = now.Add(-25 * time.Hour)
	assert.True(t, s.IsHardExpired())
}

func fooUserSession() *Session {
	return &Session{
		Session: persistence.Session[User, SessionMeta]{
			ID:        uuid.New(),
			Type:      SessionType,
			CreatedAt: time.Now(),
			ExpiresAt: time.Now().Add(time.Hour),
			Payload: User{
				ID:        uuid.New(),
				Firstname: "Foo",
				Lastname:  "Bar",
			},
			Meta: SessionMeta{},
		},
	}
}

func registerCleanupUserSessionTable(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec(ctx, "DELETE FROM sessions")
		require.NoError(t, err)
	})
}
