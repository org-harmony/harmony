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
	defer db.Close()
	repo = NewUserRepository(db)
	c = context.Background()

	os.Exit(m.Run())
}

var (
	db   *pgxpool.Pool
	repo UserRepository
	c    context.Context
)

func TestPGUserRepository(t *testing.T) {
	user, err := repo.Create(fooUserToCreate(), c)

	assert.NoError(t, err)
	assert.NotNil(t, user)
	assert.NotEmpty(t, user.Id)
	assert.NotEmpty(t, user.CreatedAt)
	assert.Nil(t, user.UpdatedAt) // should be nil because it's a new user
	assert.Equal(t, "foo@bar.com", user.Email)
	assert.Equal(t, "Foo", user.Firstname)
	assert.Equal(t, "Bar", user.Lastname)

	id := user.Id
	email := user.Email

	err = repo.Delete(id, c)
	assert.NoError(t, err)

	user, err = repo.GetById(id, c)
	assert.ErrorIs(t, err, persistence.NotFoundError)

	user, err = repo.GetByEmail(email, c)
	assert.ErrorIs(t, err, persistence.NotFoundError)

	user, err = repo.Create(fooUserToCreate(), c)
	assert.NoError(t, err)

	id = user.Id
	email = user.Email

	user, err = repo.GetByEmail(email, c)
	assert.NoError(t, err)
	assert.Equal(t, id, user.Id)

	user, err = repo.GetById(id, c)
	assert.NoError(t, err)
	assert.Equal(t, id, user.Id)
}

func fooUserToCreate() *UserToCreate {
	return &UserToCreate{
		Email:     "foo@bar.com",
		Firstname: "Foo",
		Lastname:  "Bar",
	}
}