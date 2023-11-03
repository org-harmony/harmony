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
	assert.Equal(t, update, &readSession.Payload)
}

func registerCleanupUserAndSessionTables(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec(ctx, "DELETE FROM users")
		require.NoError(t, err)

		_, err = db.Exec(ctx, "DELETE FROM sessions")
		require.NoError(t, err)
	})
}
