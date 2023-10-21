package auth

import (
	"context"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/core/persistence"
	"github.com/stretchr/testify/assert"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	db = persistence.InitTestDB("./../../")
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
	user, err := repo.Create(fooUserToCreate(), ctx)

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

	err = repo.Delete(id, ctx)
	assert.NoError(t, err)

	user, err = repo.FindByID(id, ctx)
	assert.ErrorIs(t, err, persistence.NotFoundError)

	user, err = repo.FindByEmail(email, ctx)
	assert.ErrorIs(t, err, persistence.NotFoundError)

	user, err = repo.Create(fooUserToCreate(), ctx)
	assert.NoError(t, err)

	id = user.ID
	email = user.Email

	user, err = repo.FindByEmail(email, ctx)
	assert.NoError(t, err)
	assert.Equal(t, id, user.ID)

	user, err = repo.FindByID(id, ctx)
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
