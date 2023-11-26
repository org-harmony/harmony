package web

import (
	"github.com/org-harmony/harmony/src/core/event"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/trace"
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

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("GET", "/static/test.js", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "console.log('test');")

	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("GET", "/static/test.css", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "body { color: red; }")

	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("GET", "/static/not-found", nil))
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("GET", "/static", nil))
	assert.Equal(t, http.StatusMovedPermanently, recorder.Code)
}

func TestController(t *testing.T) {
	app, ctx := setupMockCtxs(t)

	partial := NewController(app, ctx, func(io IO) error {
		return io.Render(nil, "partial", "partial.go.html")
	})
	errorHandler := NewController(app, ctx, func(io IO) error {
		return io.Error()
	})
	redirect := NewController(app, ctx, func(io IO) error {
		return io.Redirect("/", http.StatusFound)
	})
	inlineError := NewController(app, ctx, func(io IO) error {
		return io.InlineError()
	})
	htmxOnly := NewController(app, ctx, func(io IO) error {
		assert.True(t, io.IsHTMX())
		return io.Render(nil, "partial", "partial.go.html")
	})
	renderJoined := NewController(app, ctx, func(io IO) error {
		return io.Render("content-string", "printer", "partial.go.html", "printer.go.html")
	})

	router := ctx.Router
	router.Get("/test", partial.ServeHTTP)
	router.Get("/error", errorHandler.ServeHTTP)
	router.Get("/inline-error", inlineError.ServeHTTP)
	router.Get("/redirect", redirect.ServeHTTP)
	router.Get("/htmx-only", htmxOnly.ServeHTTP)
	router.Get("/render-joined", renderJoined.ServeHTTP)

	recorder := httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/test", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "Hello partial-appendix")

	recorder = httptest.NewRecorder()
	req := httptest.NewRequest("GET", "/error", nil)
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "before content; harmony.error.generic-reload; after")
	assert.NotContains(t, recorder.Body.String(), "appendix")

	recorder = httptest.NewRecorder()
	req.Header.Set("HX-Request", "true")
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "appendix")

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/inline-error", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "harmony.error.generic-reload")
	assert.NotContains(t, recorder.Body.String(), "before content;")
	assert.NotContains(t, recorder.Body.String(), "after;")

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/redirect", nil))
	assert.Equal(t, http.StatusFound, recorder.Code)
	assert.Equal(t, "/", recorder.Header().Get("Location"))

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/not-found", nil))
	assert.Equal(t, http.StatusNotFound, recorder.Code)

	recorder = httptest.NewRecorder()
	req = httptest.NewRequest("GET", "/htmx-only", nil)
	req.Header.Set("HX-Request", "true")
	router.ServeHTTP(recorder, req)
	assert.Equal(t, http.StatusOK, recorder.Code)

	recorder = httptest.NewRecorder()
	router.ServeHTTP(recorder, httptest.NewRequest("GET", "/render-joined", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Contains(t, recorder.Body.String(), "content-string")
	assert.Contains(t, recorder.Body.String(), "partial-appendix")
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

	err, _ := ReadForm(req, &ts, v)
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

	err, validationErrs := ReadForm(req, &ts, v)
	assert.NoError(t, err)
	assert.Error(t, validationErrs[0])
	assert.ErrorContains(t, validationErrs[0], "required")
	assert.ErrorContains(t, validationErrs[0], "Name")
	assert.ErrorContains(t, validationErrs[1], "positive")
	assert.Empty(t, ts.Name)
	assert.Equal(t, -30, ts.Age)
}

func TestReadFormPanicsForNonPointer(t *testing.T) {
	ts := TestStruct{} // not a pointer

	req, _ := http.NewRequest("GET", "/", nil)
	req.PostForm = url.Values{"Name": {"John"}}

	assert.PanicsWithError(t, ErrNotPointerToStruct.Error(), func() {
		_, _ = ReadForm(req, ts, nil)
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
	logger := trace.NewLogger()

	return hctx.NewAppCtx(
			logger,
			validation.New(),
			nil,
			event.NewManager(logger),
		),
		&Ctx{
			Router:         r,
			Config:         setupConfig(templatesDir, baseDir),
			TemplaterStore: ts,
			Navigation:     NewNavigation(),
			Extensions:     NewExtensions(),
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

	indexContent := "{{define \"index\"}}before content; {{block \"content\" .}}Hello from index{{end}}; after{{end}}"
	err = os.WriteFile(filepath.Join(baseDir, "index.go.html"), []byte(indexContent), 0644)
	require.NoError(t, err)

	baseContent := "{{define \"base\"}}{{ template \"index\" .}}{{end}}"
	err = os.WriteFile(filepath.Join(baseDir, "base.go.html"), []byte(baseContent), 0644)
	require.NoError(t, err)

	partialContent := "{{define \"partial\"}}{{block \"index\" .}}{{block \"content\" .}}Hello{{end}} partial-appendix{{end}}{{end}}"
	err = os.WriteFile(filepath.Join(templatesDir, "partial.go.html"), []byte(partialContent), 0644)
	require.NoError(t, err)

	emptyContent := "{{define \"index\"}}{{block \"content\" .}}empty{{end}}{{end}}"
	err = os.WriteFile(filepath.Join(templatesDir, "empty.go.html"), []byte(emptyContent), 0644)
	require.NoError(t, err)

	errorPageContent := "{{define \"error\"}}{{template \"index\" .}}{{end}}{{define \"content\"}}{{.Data}}{{end}}"
	err = os.WriteFile(filepath.Join(templatesDir, "error.go.html"), []byte(errorPageContent), 0644)
	require.NoError(t, err)

	printerPageContent := "{{define \"printer\"}}{{template \"index\" .}}{{end}}{{define \"content\"}}{{.Data}}{{end}}"
	err = os.WriteFile(filepath.Join(templatesDir, "printer.go.html"), []byte(printerPageContent), 0644)
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
