package web

import (
	"errors"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/trace"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/validation"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"testing"
)

type SimpleTestStruct struct {
	Name string `hvalidate:"required"`
	Age  int    `hvalidate:"positive"`
}

type TestStruct struct {
	Name          string
	Age           uint
	Height        float32
	CookieConsent bool
	Offset        int64
	Roles         []string
	Inner         *SimpleTestStruct
}

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

func TestController(t *testing.T) {
	app, ctx := setupMockCtxs(t)

	tmplter := util.Unwrap(ctx.TemplaterStore.Templater(LandingPageTemplateName))
	lp := util.Unwrap(tmplter.Base())

	c := NewController(app, ctx, func(io IO) error {
		return io.Render(lp, nil)
	})
	e := NewController(app, ctx, func(io IO) error {
		return io.Error(errors.New("test error"))
	})
	re := NewController(app, ctx, func(io IO) error {
		return io.Redirect("/", http.StatusFound)
	})
	verifyTemplate := NewController(app, ctx, func(io IO) error {
		tmplter2 := util.Unwrap(io.TemplaterStore().Templater(LandingPageTemplateName))
		lp2 := util.Unwrap(tmplter2.Base())
		assert.Equal(t, tmplter2, tmplter)
		assert.Equal(t, lp.Name(), lp2.Name()) // items are cloned so the names are the same
		return nil
	})

	router := ctx.Router
	router.Get("/test", c.ServeHTTP)
	router.Get("/error", e.ServeHTTP)
	router.Get("/redirect", re.ServeHTTP)
	router.Get("/verify-template", verifyTemplate.ServeHTTP)

	req, _ := http.NewRequest("GET", "/test", nil)
	resp := executeRequest(req, router)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "Hello from landing page")

	req, _ = http.NewRequest("GET", "/error", nil)
	resp = executeRequest(req, router)
	assert.Equal(t, http.StatusOK, resp.Code)
	assert.Contains(t, resp.Body.String(), "test error")

	req, _ = http.NewRequest("GET", "/redirect", nil)
	resp = executeRequest(req, router)
	assert.Equal(t, http.StatusFound, resp.Code)

	req, _ = http.NewRequest("GET", "/verify-template", nil)
	resp = executeRequest(req, router)
	assert.Equal(t, http.StatusOK, resp.Code)
}

func TestValuesIntoStruct(t *testing.T) {
	ts := TestStruct{}
	values := map[string][]string{
		"Name":          {"John"},
		"Age":           {"30"},
		"Height":        {"1.865"},
		"CookieConsent": {"true"},
		"Offset":        {"-1"},
		"Roles":         {"admin", "user"},
		"Inner.Name":    {"John"},
	}

	err := ValuesIntoStruct(values, &ts)
	assert.NoError(t, err)
	assert.Equal(t, "John", ts.Name)
	assert.Equal(t, uint(30), ts.Age)
	assert.Equal(t, float32(1.865), ts.Height)
	assert.Equal(t, true, ts.CookieConsent)
	assert.Equal(t, int64(-1), ts.Offset)
	assert.Nil(t, ts.Roles) // not supported yet
	assert.Nil(t, ts.Inner) // not supported yet

	// Invalid values where error occurs should be skipped
	ts = TestStruct{}
	values = map[string][]string{
		"Name": {"John"},
		"Age":  {"-30"},
	}

	err = ValuesIntoStruct(values, &ts)
	assert.NoError(t, err)
	assert.Equal(t, "John", ts.Name)
	assert.Equal(t, uint(0), ts.Age)
}

func TestReadForm(t *testing.T) {
	v := validation.New()

	ts := SimpleTestStruct{}
	values := map[string][]string{
		"Name": {"John"},
		"Age":  {"30"},
	}

	req, _ := http.NewRequest("GET", "/", nil)
	req.PostForm = values

	err := ReadForm(req, &ts, v)
	assert.NoError(t, err)
	assert.Equal(t, "John", ts.Name)
	assert.Equal(t, 30, ts.Age)

	// Invalid values
	ts = SimpleTestStruct{}
	values = map[string][]string{
		"Age": {"-30"},
	}

	req, _ = http.NewRequest("GET", "/", nil)
	req.PostForm = values

	err = ReadForm(req, &ts, v)
	assert.Error(t, err)
	assert.ErrorContains(t, err, "required")
	assert.ErrorContains(t, err, "Name")
	assert.ErrorContains(t, err, "positive")
	assert.Empty(t, ts.Name)
	assert.Equal(t, -30, ts.Age)
}

func TestReadFormPanicsForNonPointer(t *testing.T) {
	ts := TestStruct{} // not a pointer

	req, _ := http.NewRequest("GET", "/", nil)
	req.PostForm = url.Values{"Name": {"John"}}

	assert.PanicsWithError(t, ErrNotPointerToStruct.Error(), func() {
		_ = ReadForm(req, ts, nil)
	}, "ReadForm should panic when data is not a pointer to a struct")
}

func TestValuesIntoStructNoPanicForNonPointer(t *testing.T) {
	ts := TestStruct{} // not a pointer
	values := url.Values{"Name": {"John"}}

	err := ValuesIntoStruct(values, ts)
	assert.ErrorIs(t, err, ErrNotPointerToStruct)

	assert.Equal(t, TestStruct{}, ts)
}

func setupMockCtxs(t *testing.T) (*hctx.AppCtx, *Ctx) {
	r, ts := setupMock(t)
	templatesDir, baseDir := setupDirectories(t)

	return hctx.NewAppCtx(
			trace.NewLogger(),
			validation.New(),
			nil,
		),
		&Ctx{
			Router:         r,
			Config:         setupConfig(templatesDir, baseDir),
			TemplaterStore: ts,
		}
}

func setupMock(t *testing.T) (Router, TemplaterStore) {
	templateDir, baseDir := setupDirectories(t)
	cfg := setupConfig(templateDir, baseDir)

	s, err := SetupTemplaterStore(cfg.UI)
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

	baseContent := "{{define \"base\"}}{{ template \"index\" . }}{{end}}"
	err = os.WriteFile(filepath.Join(baseDir, "base.go.html"), []byte(baseContent), 0644)
	require.NoError(t, err)

	landingPageContent := "{{define \"landing-page\"}}Hello from landing page{{end}}"
	err = os.WriteFile(filepath.Join(templatesDir, "landing-page.go.html"), []byte(landingPageContent), 0644)
	require.NoError(t, err)

	errorPageContent := "{{define \"error\"}}{{ .Err }}{{end}}"
	err = os.WriteFile(filepath.Join(templatesDir, "error.go.html"), []byte(errorPageContent), 0644)
	require.NoError(t, err)

	return templatesDir, baseDir
}

// setupConfig returns a basic server configuration.
func setupConfig(dir string, baseDir string) *Cfg {
	return &Cfg{
		Server: &ServerCfg{
			Addr:    "localhost",
			Port:    "8080",
			BaseURL: "http://localhost:8080",
		},
		UI: &UICfg{
			Templates: &TemplatesCfg{
				Dir:     dir,
				BaseDir: baseDir,
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
