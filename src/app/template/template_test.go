package template

import (
	"context"
	"encoding/json"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/src/app/user"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	db = persistence.InitTestDB("./../../../")
	templateRepo = NewRepository(db)
	templateSetRepo = NewSetRepository(db)
	userRepo = user.NewUserRepository(db)
	ctx = context.Background()
	result := m.Run()
	db.Close()
	os.Exit(result)
}

var (
	db              *pgxpool.Pool
	templateRepo    Repository
	templateSetRepo SetRepository
	userRepo        user.Repository
	ctx             context.Context
)

func TestNewRepository(t *testing.T) {
	repo1 := NewRepository(db)
	require.NotNil(t, repo1)

	repo2 := NewSetRepository(db)
	require.NotNil(t, repo2)
}

func TestPGRepository(t *testing.T) {
	registerAllCleanup(t)

	u, tmplSet, tmpl := mockTemplate(t)

	t.Run("FindById", func(t *testing.T) {
		found, err := templateRepo.FindByID(ctx, tmpl.ID)
		require.NoError(t, err)
		require.NotNil(t, tmpl)
		unifiedJsonEqual(t, tmpl.Json, found.Json)
		assert.Equal(t, tmplUnify(*tmpl), tmplUnify(*found))
	})

	t.Run("FindByTemplateSet", func(t *testing.T) {
		tmplToCreate := &ToCreate{
			Type: "ebt",
			Json: `{
			"name": "Baz",
			"version": "1.0.0",
			"authors": ["Qux Bar"],
			"license": "MIT",
			"description": "Baz Qux Foo Bar"
		}`,
			TemplateSet: tmplSet.ID,
			CreatedBy:   u.ID,
		}

		_, err := templateRepo.Create(ctx, tmplToCreate)
		require.NoError(t, err)

		found, err := templateRepo.FindByTemplateSetID(ctx, tmplSet.ID)
		require.NoError(t, err)

		assert.Len(t, found, 2)
	})

	t.Run("Create Template", func(t *testing.T) {
		tmplToCreate := &ToCreate{
			Type: "ebt",
			Json: `{
			"name": "Baz",
			"version": "1.0.0",
			"authors": ["Qux Bar"],
			"license": "MIT",
			"description": "Baz Qux Foo Bar"
		}`,
			TemplateSet: tmplSet.ID,
			CreatedBy:   u.ID,
		}

		tmpl, err := templateRepo.Create(ctx, tmplToCreate)
		require.NoError(t, err)
		require.NotNil(t, tmpl)

		assert.NotEmpty(t, tmpl.ID)
		assert.NotEmpty(t, tmpl.CreatedAt)
		assert.Nil(t, tmpl.UpdatedAt) // should be nil because it's a new template
		assert.Equal(t, tmpl.Type, "ebt")
		assert.Equal(t, tmpl.Name, "Baz")
		assert.Equal(t, tmpl.Version, "1.0.0")
		assert.Equal(t, tmpl.TemplateSet, tmplSet.ID)
		assert.Equal(t, tmpl.CreatedBy, u.ID)
		unifiedJsonEqual(t, tmplToCreate.Json, tmpl.Json)
	})

	t.Run("Update Template", func(t *testing.T) {
		_, _, toCreate := fooToCreate()
		toCreate.TemplateSet = tmplSet.ID
		toCreate.CreatedBy = u.ID
		newTmpl, err := templateRepo.Create(ctx, toCreate)
		require.NoError(t, err)
		require.NotNil(t, newTmpl)
		require.Nil(t, newTmpl.UpdatedAt)

		toUpdate := newTmpl.ToUpdate()
		toUpdate.Type = "foo"
		toUpdate.Json = `{
			"name": "Bizzo",
			"version": "2.0.0",
			"authors": ["Qux Bar"],
			"license": "MIT",
			"description": "Baz Qux Foo Bar"
		}`

		update, err := templateRepo.Update(ctx, toUpdate)
		require.NoError(t, err)
		require.NotNil(t, update)

		assert.NotEmpty(t, update.UpdatedAt)
		assert.Equal(t, update.Type, "foo")
		assert.Equal(t, update.Name, "Bizzo")
		assert.Equal(t, update.Version, "2.0.0")
		unifiedJsonEqual(t, toUpdate.Json, update.Json)
	})

	t.Run("Delete Template", func(t *testing.T) {
		_, _, toCreate := fooToCreate()
		toCreate.TemplateSet = tmplSet.ID
		toCreate.CreatedBy = u.ID
		newTmpl, err := templateRepo.Create(ctx, toCreate)
		require.NoError(t, err)
		require.NotNil(t, newTmpl)

		err = templateRepo.Delete(ctx, newTmpl.ID)
		require.NoError(t, err)

		_, err = templateRepo.FindByID(ctx, newTmpl.ID)
		assert.ErrorIs(t, err, persistence.ErrNotFound)
	})

	t.Run("Invalid CreatedBy", func(t *testing.T) {
		_, _, toCreate := fooToCreate()
		toCreate.TemplateSet = tmplSet.ID
		toCreate.CreatedBy = uuid.New()
		_, err := templateRepo.Create(ctx, toCreate)
		assert.ErrorIs(t, err, persistence.ErrInsert)
	})

	t.Run("Invalid TemplateSet", func(t *testing.T) {
		_, _, toCreate := fooToCreate()
		toCreate.TemplateSet = uuid.New()
		toCreate.CreatedBy = u.ID
		_, err := templateRepo.Create(ctx, toCreate)
		assert.ErrorIs(t, err, persistence.ErrInsert)
	})
}

func TestPGSetRepository(t *testing.T) {
	registerAllCleanup(t)

	u, tmplSet, _ := mockTemplate(t)

	t.Run("FindById", func(t *testing.T) {
		found, err := templateSetRepo.FindByID(ctx, tmplSet.ID)
		require.NoError(t, err)
		require.NotNil(t, tmplSet)
		assert.Equal(t, tmplSetUnify(*tmplSet), tmplSetUnify(*found))
	})

	t.Run("FindByCreatedBy", func(t *testing.T) {
		tmplSetToCreate := &SetToCreate{
			Name:        "Baz",
			Description: "Baz Qux Foo Bar",
			CreatedBy:   u.ID,
		}

		_, err := templateSetRepo.Create(ctx, tmplSetToCreate)
		require.NoError(t, err)

		found, err := templateSetRepo.FindByCreatedBy(ctx, u.ID)
		require.NoError(t, err)

		assert.Len(t, found, 2)
	})

	t.Run("Create TemplateSet", func(t *testing.T) {
		tmplSetToCreate := &SetToCreate{
			Name:        "Baz",
			Description: "Baz Qux Foo Bar",
			CreatedBy:   u.ID,
		}

		tmplSet, err := templateSetRepo.Create(ctx, tmplSetToCreate)
		require.NoError(t, err)
		require.NotNil(t, tmplSet)

		assert.NotEmpty(t, tmplSet.ID)
		assert.NotEmpty(t, tmplSet.CreatedAt)
		assert.Nil(t, tmplSet.UpdatedAt) // should be nil because it's a new template set
		assert.Equal(t, tmplSet.Name, "Baz")
		assert.Equal(t, tmplSet.Description, "Baz Qux Foo Bar")
		assert.Equal(t, tmplSet.CreatedBy, u.ID)
	})

	t.Run("Update TemplateSet", func(t *testing.T) {
		_, toCreate, _ := fooToCreate()
		toCreate.CreatedBy = u.ID
		newTmplSet, err := templateSetRepo.Create(ctx, toCreate)
		require.NoError(t, err)
		require.NotNil(t, newTmplSet)
		require.Nil(t, newTmplSet.UpdatedAt)

		toUpdate := newTmplSet.ToUpdate()
		toUpdate.Name = "Baz"
		toUpdate.Description = "Baz Qux Foo Bar"

		update, err := templateSetRepo.Update(ctx, toUpdate)
		require.NoError(t, err)
		require.NotNil(t, update)

		assert.NotEmpty(t, update.UpdatedAt)
		assert.Equal(t, update.Name, "Baz")
		assert.Equal(t, update.Description, "Baz Qux Foo Bar")
	})

	t.Run("Delete TemplateSet", func(t *testing.T) {
		_, toCreate, _ := fooToCreate()
		toCreate.CreatedBy = u.ID
		newTmplSet, err := templateSetRepo.Create(ctx, toCreate)
		require.NoError(t, err)
		require.NotNil(t, newTmplSet)

		err = templateSetRepo.Delete(ctx, newTmplSet.ID)
		require.NoError(t, err)

		_, err = templateSetRepo.FindByID(ctx, newTmplSet.ID)
		assert.ErrorIs(t, err, persistence.ErrNotFound)
	})

	t.Run("Invalid CreatedBy", func(t *testing.T) {
		_, err := templateSetRepo.Create(ctx, &SetToCreate{
			Name:        "Foo",
			Description: "Foo Bar",
			CreatedBy:   uuid.New(),
		})
		assert.ErrorIs(t, err, persistence.ErrInsert)
	})
}

// mockTemplate will create a user, template set and template in the database and return them.
func mockTemplate(t *testing.T) (*user.User, *Set, *Template) {
	userToCreate, templateSetToCreate, templateToCreate := fooToCreate()
	return createTemplate(t, userToCreate, templateSetToCreate, templateToCreate)
}

func createTemplate(t *testing.T, userToCreate *user.ToCreate, tmplSetToCreate *SetToCreate, tmplToCreate *ToCreate) (*user.User, *Set, *Template) {
	u, err := userRepo.Create(ctx, userToCreate)
	require.NoError(t, err)

	tmplSetToCreate.CreatedBy = u.ID
	templateSet, err := templateSetRepo.Create(ctx, tmplSetToCreate)
	require.NoError(t, err)

	tmplToCreate.TemplateSet = templateSet.ID
	tmplToCreate.CreatedBy = u.ID
	template, err := templateRepo.Create(ctx, tmplToCreate)
	require.NoError(t, err)

	return u, templateSet, template
}

func fooToCreate() (*user.ToCreate, *SetToCreate, *ToCreate) {
	return &user.ToCreate{
			Email:     "foo@bar.com",
			Firstname: "Foo",
			Lastname:  "Bar",
		}, &SetToCreate{
			Name:        "Foo",
			Description: "Foo Bar",
		}, &ToCreate{
			Type: "ebt",
			Json: `{
				"name": "Foo",
				"version": "1.0.0",
				"authors": ["Foo Bar"],
				"license": "MIT",
				"description": "Foo Bar"
			}`,
		}
}

// tmplUnify unifies the template for comparison.
// It sets the json to "{}" as different whitespaces may lead to different json strings while the content is identical.
// It truncates the time to seconds as the database does not store milliseconds.
func tmplUnify(tmpl Template) Template {
	tmpl.Json = "{}"
	tmpl.CreatedAt = tmpl.CreatedAt.Truncate(time.Second)
	if tmpl.UpdatedAt != nil {
		*tmpl.UpdatedAt = tmpl.UpdatedAt.Truncate(time.Second)
	}

	return tmpl
}

// unifiedJsonEqual compares two json strings by unmarshalling them into a map[string]any.
// Even with different whitespaces the json strings are considered equal if the content is equal.
func unifiedJsonEqual(t *testing.T, expected string, actual string) {
	expectedJson := make(map[string]any)
	actualJson := make(map[string]any)

	err := json.Unmarshal([]byte(expected), &expectedJson)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(actual), &actualJson)
	require.NoError(t, err)

	assert.Equal(t, expectedJson, actualJson)
}

// tmplSetUnify unifies the template set for comparison. It truncates the time to seconds as the database does not store milliseconds.
func tmplSetUnify(tmplSet Set) Set {
	tmplSet.CreatedAt = tmplSet.CreatedAt.Truncate(time.Second)
	if tmplSet.UpdatedAt != nil {
		*tmplSet.UpdatedAt = tmplSet.UpdatedAt.Truncate(time.Second)
	}

	return tmplSet
}

// registerAllCleanup registers cleanups for template, template set and user tables after each test.
func registerAllCleanup(t *testing.T) {
	t.Cleanup(func() {
		cleanupTemplateTable(t)
		cleanupTemplateSetTable(t)
		cleanupUserTable(t)
	})
}

func cleanupTemplateTable(t *testing.T) {
	_, err := db.Exec(ctx, "TRUNCATE TABLE templates CASCADE")
	require.NoError(t, err)
}

func cleanupTemplateSetTable(t *testing.T) {
	_, err := db.Exec(ctx, "TRUNCATE TABLE template_sets CASCADE")
	require.NoError(t, err)
}

func cleanupUserTable(t *testing.T) {
	_, err := db.Exec(ctx, "TRUNCATE TABLE users CASCADE")
	require.NoError(t, err)
}
