package web

import (
	"github.com/org-harmony/harmony/core/trans"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestMountFileServer(t *testing.T) {
	r, _ := setupMock(t)
	setupAssetsFileServer(t, r)

	req, _ := http.NewRequest("GET", "/static/test.js", nil)
	resp := executeRequest(req, r)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "console.log('test');")

	req, _ = http.NewRequest("GET", "/static/test.css", nil)
	resp = executeRequest(req, r)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "body { color: red; }")

	req, _ = http.NewRequest("GET", "/static/not-found", nil)
	resp = executeRequest(req, r)
	assert.Equal(t, http.StatusNotFound, resp.Code)

	req, _ = http.NewRequest("GET", "/static/test.js/", nil)
	resp = executeRequest(req, r)
	assert.Equal(t, http.StatusMovedPermanently, resp.Code)
}

func setupMock(t *testing.T) (Router, TemplaterStore) {
	templateDir, baseDir := setupDirectories(t)
	cfg := setupConfig(templateDir, baseDir)
	tr := trans.NewTranslator()

	s, err := SetupTemplaterStore(cfg.UI, tr)
	require.NoError(t, err)

	return NewRouter(), s
}

func setupAssetsFileServer(t *testing.T, r Router) {
	assetsDir := setupAssetsDirectory(t)

	MountFileServer(r, &FileServerCfg{
		Root:  assetsDir,
		Route: "/static",
	})
}

// setupDirectories sets up the directories and writes templates. It returns the paths to the created directories.
func setupDirectories(t *testing.T) (string, string) {
	tempDir := t.TempDir()
	templatesDir := filepath.Join(tempDir, "templates")
	baseDir := filepath.Join(tempDir, "templates", "base")

	err := os.Mkdir(templatesDir, 0755)
	require.NoError(t, err)
	err = os.Mkdir(baseDir, 0755)
	require.NoError(t, err)

	indexContent := "{{define \"index\"}}Hello from index{{end}}"
	err = os.WriteFile(filepath.Join(baseDir, "index.go.html"), []byte(indexContent), 0644)
	require.NoError(t, err)

	landingPageContent := "{{define \"landing-page\"}}Hello from landing page{{end}}"
	err = os.WriteFile(filepath.Join(templatesDir, "landing-page.go.html"), []byte(landingPageContent), 0644)
	require.NoError(t, err)

	return templatesDir, baseDir
}

// setupConfig returns a basic server configuration.
func setupConfig(d string, bd string) *Cfg {
	return &Cfg{
		Server: &ServerCfg{
			Addr:    "localhost",
			Port:    "8080",
			BaseURL: "http://localhost:8080",
		},
		UI: &UICfg{
			Templates: &TemplatesCfg{
				Dir:     d,
				BaseDir: bd,
			},
		},
	}
}

// setupAssetsDirectory sets up the assets directory with sample files and returns its path.
func setupAssetsDirectory(t *testing.T) string {
	tempDir := t.TempDir()
	assetsDir := filepath.Join(tempDir, "assets")

	err := os.Mkdir(assetsDir, 0755)
	require.NoError(t, err)

	jsContent := "console.log('test');"
	err = os.WriteFile(filepath.Join(assetsDir, "test.js"), []byte(jsContent), 0644)
	require.NoError(t, err)

	cssContent := "body { color: red; }"
	err = os.WriteFile(filepath.Join(assetsDir, "test.css"), []byte(cssContent), 0644)
	require.NoError(t, err)

	return assetsDir
}

// executeRequest executes the request and returns the response recorder.
func executeRequest(req *http.Request, r Router) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)
	return rr
}

// checkResponseCode checks the response code and fails the test if it is not the expected code.
func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}
