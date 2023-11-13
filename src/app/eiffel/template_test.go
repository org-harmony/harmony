package eiffel

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/src/app/user"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	db = persistence.InitTestDB("./../../../")
	templateRepo = NewTemplateRepository(db)
	templateSetRepo = NewTemplateSetRepository(db)
	userRepo = user.NewUserRepository(db)
	ctx = context.Background()
	result := m.Run()
	db.Close()
	os.Exit(result)
}

var (
	db              *pgxpool.Pool
	templateRepo    TemplateRepository
	templateSetRepo TemplateSetRepository
	userRepo        user.Repository
	ctx             context.Context
)

func TestNewRepository(t *testing.T) {
	repo1 := NewTemplateRepository(db)
	require.NotNil(t, repo1)

	repo2 := NewTemplateSetRepository(db)
	require.NotNil(t, repo2)
}

func TestPGTemplateRepository_Create(t *testing.T) {
	registerAllCleanup(t)

	u, tmplSet, _ := mockTemplate(t)

	tmplToCreate := &TemplateToCreate{
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

	originalJson := make(map[string]any)
	dbJson := make(map[string]any)

	err = json.Unmarshal([]byte(tmplToCreate.Json), &originalJson)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(tmpl.Json), &dbJson)
	require.NoError(t, err)

	require.NotEmpty(t, tmpl.ID)
	require.NotEmpty(t, tmpl.CreatedAt)
	require.Nil(t, tmpl.UpdatedAt) // should be nil because it's a new template
	require.Equal(t, tmpl.Type, "ebt")
	require.Equal(t, tmpl.Name, "Baz")
	require.Equal(t, tmpl.Version, "1.0.0")
	require.Equal(t, originalJson, dbJson)
	require.Equal(t, tmpl.TemplateSet, tmplSet.ID)
	require.Equal(t, tmpl.CreatedBy, u.ID)
}

func mockTemplate(t *testing.T) (*user.User, *TemplateSet, *Template) {
	userToCreate, templateSetToCreate, templateToCreate := fooToCreate()
	return createTemplate(t, userToCreate, templateSetToCreate, templateToCreate)
}

func createTemplate(t *testing.T, userToCreate *user.ToCreate, tmplSetToCreate *TemplateSetToCreate, tmplToCreate *TemplateToCreate) (*user.User, *TemplateSet, *Template) {
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

func fooToCreate() (*user.ToCreate, *TemplateSetToCreate, *TemplateToCreate) {
	return &user.ToCreate{
			Email:     "foo@bar.com",
			Firstname: "Foo",
			Lastname:  "Bar",
		}, &TemplateSetToCreate{
			Name:        "Foo",
			Description: "Foo Bar",
		}, &TemplateToCreate{
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
