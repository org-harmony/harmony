package web

import (
	"context"
	"encoding/json"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/src/app/template"
	"github.com/org-harmony/harmony/src/app/user"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	db = persistence.InitTestDB("./../../../../")
	templateRepo = template.NewRepository(db)
	templateSetRepo = template.NewSetRepository(db)
	userRepo = user.NewUserRepository(db)
	ctx = context.Background()
	result := m.Run()
	db.Close()
	os.Exit(result)
}

var (
	db              *pgxpool.Pool
	templateRepo    template.Repository
	templateSetRepo template.SetRepository
	userRepo        user.Repository
	ctx             context.Context
)

func TestCopyTemplate(t *testing.T) {
	registerAllCleanup(t)

	usr, tmplSet, tmpl := mockTemplate(t)

	t.Run("copy template with name change", func(t *testing.T) {
		copyTmpl, err := CopyTemplate(ctx, tmpl, tmplSet.ID, usr.ID, "New Name", templateRepo)
		require.NoError(t, err)

		require.Equal(t, "New Name", copyTmpl.Name)

		fetchedTmpl, err := templateRepo.FindByID(ctx, copyTmpl.ID)
		require.NoError(t, err)
		require.Equal(t, tmplUnify(*copyTmpl), tmplUnify(*fetchedTmpl))
	})
}

// mockTemplate will create a user, template set and template in the database and return them.
func mockTemplate(t *testing.T) (*user.User, *template.Set, *template.Template) {
	userToCreate, templateSetToCreate, templateToCreate := initialToCreate()
	return createTemplate(t, userToCreate, templateSetToCreate, templateToCreate)
}

func createTemplate(t *testing.T, userToCreate *user.ToCreate, tmplSetToCreate *template.SetToCreate, tmplToCreate *template.ToCreate) (*user.User, *template.Set, *template.Template) {
	u, err := userRepo.Create(ctx, userToCreate)
	require.NoError(t, err)

	tmplSetToCreate.CreatedBy = u.ID
	templateSet, err := templateSetRepo.Create(ctx, tmplSetToCreate)
	require.NoError(t, err)

	tmplToCreate.TemplateSet = templateSet.ID
	tmplToCreate.CreatedBy = u.ID
	tmpl, err := templateRepo.Create(ctx, tmplToCreate)
	require.NoError(t, err)

	return u, templateSet, tmpl
}

func initialToCreate() (*user.ToCreate, *template.SetToCreate, *template.ToCreate) {
	return &user.ToCreate{
			Email:     "foo@bar.com",
			Firstname: "Firstname",
			Lastname:  "Lastname",
		}, &template.SetToCreate{
			Name:        "Test Template Set",
			Description: "Description: Foo Bar Baz Qux Qux",
		}, &template.ToCreate{
			Type: "ebt",
			Config: `{
				"name": "Initial Template",
				"version": "1.0.0",
				"authors": ["Author Foo", "Author Bar"],
				"license": "MIT",
				"description": "Foo Bar"
			}`,
		}
}

// tmplUnify unifies the template for comparison.
// It sets the config json to "{}" as different whitespaces may lead to different json strings while the content is identical.
// It truncates the time to seconds as the database does not store milliseconds.
func tmplUnify(tmpl template.Template) template.Template {
	tmpl.Config = "{}"
	tmpl.CreatedAt = tmpl.CreatedAt.Truncate(time.Second)
	if tmpl.UpdatedAt != nil {
		*tmpl.UpdatedAt = tmpl.UpdatedAt.Truncate(time.Second)
	}

	return tmpl
}

// unifiedConfigEqual compares two config json strings by unmarshalling them into a map[string]any.
// Even with different whitespaces the config json strings are considered equal if the content is equal.
func unifiedConfigEqual(t *testing.T, expectedJson string, actualJson string) {
	expectedConfig := make(map[string]any)
	actualConfig := make(map[string]any)

	err := json.Unmarshal([]byte(expectedJson), &expectedConfig)
	require.NoError(t, err)
	err = json.Unmarshal([]byte(actualJson), &actualConfig)
	require.NoError(t, err)

	assert.Equal(t, expectedConfig, actualConfig)
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
