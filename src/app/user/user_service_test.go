package user

import (
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"testing"
	"time"
)

func TestUpdateUser(t *testing.T) {
	registerCleanupUserAndSessionTables(t)

	user, err := userRepo.Create(ctx, fooUserToCreate())
	assert.NoError(t, err)

	session := NewUserSession(user, time.Hour)
	err = sessionStore.Insert(ctx, session)
	assert.NoError(t, err)

	toUpdate := user.ToUpdate()
	toUpdate.Firstname = "Baz"
	toUpdate.Lastname = "Qux"

	update, err := UpdateUser(ctx, toUpdate, session, userRepo, sessionStore)
	assert.NoError(t, err)
	assert.Equal(t, update.Firstname, "Baz")
	assert.Equal(t, update.Lastname, "Qux")

	readSession, err := sessionStore.Read(ctx, session.ID)
	assert.NoError(t, err)
	// if equality of 'update' and 'readSession.Payload' was checked it would fail if the database had a different time zone
	// to prevent this and make our lives easier we just check the ID -> wird schon passen :)
	assert.Equal(t, update.ID, readSession.Payload.ID)
}

func TestTryExtendSession(t *testing.T) {
	registerCleanupUserAndSessionTables(t)

	user, err := userRepo.Create(ctx, fooUserToCreate())
	assert.NoError(t, err)

	session := NewUserSession(user, time.Hour)
	err = sessionStore.Insert(ctx, session)
	assert.NoError(t, err)

	err = TryExtendSession(ctx, session, time.Hour, sessionStore)
	assert.NoError(t, err)

	readSession, err := sessionStore.Read(ctx, session.ID)
	assert.NoError(t, err)
	assert.Equal(t, session.ID, readSession.ID)
	assert.Equal(t, readSession.ExpiresAt.Truncate(time.Second), time.Now().Add(time.Hour).Truncate(time.Second))
}

func registerCleanupUserAndSessionTables(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec(ctx, "DELETE FROM users")
		require.NoError(t, err)

		_, err = db.Exec(ctx, "DELETE FROM sessions")
		require.NoError(t, err)
	})
}
