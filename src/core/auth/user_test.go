package auth

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	db = persistence.InitTestDB("./../../../")
	repo = NewUserRepository(db)
	ctx = context.Background()
	result := m.Run()
	db.Close()
	os.Exit(result)
}

var (
	db   *pgxpool.Pool
	repo UserRepository
	ctx  context.Context
)

func TestPGUserRepository(t *testing.T) {
	user, err := repo.Create(ctx, fooUserToCreate())

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, user.ID)
	assert.NotEmpty(t, user.CreatedAt)
	assert.Nil(t, user.UpdatedAt) // should be nil because it's a new user
	assert.Equal(t, "foo@bar.com", user.Email)
	assert.Equal(t, "Foo", user.Firstname)
	assert.Equal(t, "Bar", user.Lastname)

	id := user.ID
	email := user.Email

	err = repo.Delete(ctx, id)
	assert.NoError(t, err)

	user, err = repo.FindByID(ctx, id)
	assert.ErrorIs(t, err, persistence.ErrNotFound)

	user, err = repo.FindByEmail(ctx, email)
	assert.ErrorIs(t, err, persistence.ErrNotFound)

	user, err = repo.Create(ctx, fooUserToCreate())
	assert.NoError(t, err)

	id = user.ID
	email = user.Email

	user, err = repo.FindByEmail(ctx, email)
	assert.NoError(t, err)
	assert.Equal(t, id, user.ID)

	user, err = repo.FindByID(ctx, id)
	assert.NoError(t, err)
	assert.Equal(t, id, user.ID)
}

func fooUserToCreate() *UserToCreate {
	return &UserToCreate{
		Email:     "foo@bar.com",
		Firstname: "Foo",
		Lastname:  "Bar",
	}
}
