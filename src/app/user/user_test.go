package user

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	db = persistence.InitTestDB("./../../../")
	userRepo = NewUserRepository(db)
	sessionStore = NewPGUserSessionRepository(db)
	ctx = context.Background()
	result := m.Run()
	db.Close()
	os.Exit(result)
}

var (
	db           *pgxpool.Pool
	userRepo     Repository
	sessionStore SessionRepository
	ctx          context.Context
)

func TestNewUserRepository(t *testing.T) {
	repo := NewUserRepository(db)
	assert.NotNil(t, repo)
}

func TestPGUserRepository_Create(t *testing.T) {
	registerCleanupUserTable(t)

	user, err := userRepo.Create(ctx, fooUserToCreate())

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, user.ID)
	assert.NotEmpty(t, user.CreatedAt)
	assert.Nil(t, user.UpdatedAt) // should be nil because it's a new user
	assert.Equal(t, user.Firstname, "Foo")
	assert.Equal(t, user.Lastname, "Bar")
}

func TestPGUserRepository_Update(t *testing.T) {
	registerCleanupUserTable(t)

	user, err := userRepo.Create(ctx, fooUserToCreate())
	assert.NoError(t, err)

	user.Firstname = "Baz"
	user.Lastname = "Qux"

	user, err = userRepo.Update(ctx, user.ToUpdate())

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, user.UpdatedAt)
	assert.Equal(t, user.Firstname, "Baz")
	assert.Equal(t, user.Lastname, "Qux")
}

func TestPGUserRepository_Delete(t *testing.T) {
	registerCleanupUserTable(t)

	user, err := userRepo.Create(ctx, fooUserToCreate())
	assert.NoError(t, err)

	_, err = userRepo.FindByID(ctx, user.ID)
	assert.NoError(t, err)

	err = userRepo.Delete(ctx, user.ID)
	assert.NoError(t, err)

	user, err = userRepo.FindByID(ctx, user.ID)
	assert.ErrorIs(t, err, persistence.ErrNotFound)
}

func TestPGUSerRepository_FindBy(t *testing.T) {
	registerCleanupUserTable(t)

	user, err := userRepo.Create(ctx, fooUserToCreate())
	assert.NoError(t, err)

	user, err = userRepo.FindByEmail(ctx, user.Email)
	assert.NoError(t, err)

	user, err = userRepo.FindByID(ctx, user.ID)
	assert.NoError(t, err)
}

func registerCleanupUserTable(t *testing.T) {
	t.Cleanup(func() {
		_, err := db.Exec(ctx, "DELETE FROM users")
		require.NoError(t, err)
	})
}

func fooUserToCreate() *ToCreate {
	return &ToCreate{
		Email:     "foo@bar.com",
		Firstname: "Foo",
		Lastname:  "Bar",
	}
}
