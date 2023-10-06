package web

import (
	"context"
	"errors"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/org-harmony/harmony/core/event"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/trans"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestStdServerInitialization(t *testing.T) {
	t.Run("new std server", func(t *testing.T) {
		cfg := setupConfig("", "")
		cfg.Server.AssetFsCfg = &FileServerCfg{
			Root:  "/assets",
			Route: "/static",
		}

		// Test NewServer without options
		srv := NewServer(cfg)
		assert.NotNil(t, srv)

		// Check default values
		assert.NotNil(t, srv.router)
		assert.NotNil(t, srv.logger)
		assert.NotNil(t, srv.translator)
	})

	t.Run("new std server with options", func(t *testing.T) {
		d, bd := setupDirectories(t)
		cfg := setupConfig(d, bd)
		srv := setupServerWithMocks(t, cfg)

		// Verify if mocks were correctly set
		assert.NotNil(t, srv)
		assert.Implements(t, (*chi.Router)(nil), srv.router)
		assert.Implements(t, (*trace.Logger)(nil), srv.logger)
		assert.Implements(t, (*trans.Translator)(nil), srv.translator)
		assert.Implements(t, (*event.EventManager)(nil), srv.eventManager)
		assert.Implements(t, (*Templater)(nil), srv.templaters[BaseTemplate])
	})
}

func TestFileServer(t *testing.T) {
	t.Run("serve static assets", func(t *testing.T) {
		srv := setupAssetsFileServer(t)

		// Request JS file
		req, _ := http.NewRequest("GET", "/static/test.js", nil)
		response := executeRequest(req, srv)
		checkResponseCode(t, http.StatusOK, response.Code)
		assert.Equal(t, "console.log('test');", response.Body.String())

		// Request CSS file
		req, _ = http.NewRequest("GET", "/static/test.css", nil)
		response = executeRequest(req, srv)
		checkResponseCode(t, http.StatusOK, response.Code)
		assert.Equal(t, "body { color: red; }", response.Body.String())

		// Request a non-existent file
		req, _ = http.NewRequest("GET", "/static/non-existent.css", nil)
		response = executeRequest(req, srv)
		checkResponseCode(t, http.StatusNotFound, response.Code)
	})
}

func TestControllerFunctionality(t *testing.T) {
	// Using the utility functions for setup
	d, bd := setupDirectories(t)
	cfg := setupConfig(d, bd)
	srv := setupMockServer(t, cfg)

	// GET handler
	getHandler := func(io HandlerIO, ctx context.Context) {
		err := io.Render(BaseTemplate, BaseTemplate, nil)
		assert.NoError(t, err)
	}

	// POST handler
	postHandler := func(io HandlerIO, ctx context.Context) {
		io.Redirect("/success", http.StatusSeeOther)
	}

	// PUT handler
	putHandler := func(io HandlerIO, ctx context.Context) {
		io.IssueError(IntErr())
	}

	// PATCH handler
	patchHandler := func(io HandlerIO, ctx context.Context) {
		io.IssueError(ExtErr(nil, http.StatusNotImplemented, "not implemented"))
	}

	// Custom error handler
	errorHandler := func(err HandlerError) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, fmt.Sprintf("Custom Error: %s", err.Error()), err.Status)
		}
	}

	// Register a controller with GET, POST handlers and a custom error handler
	controller := NewController("test.route", "/test",
		Get(getHandler),
		Post(postHandler),
		Put(putHandler),
		Patch(patchHandler),
		Error(errorHandler),
		WithTemplaters(srv.Templaters()),
	)

	redirectController := NewController("redirect.route", "/redirect",
		Get(func(io HandlerIO, ctx context.Context) {
			err := io.RedirectRoute("test.route", http.StatusSeeOther)
			assert.NoError(t, err)
		}),
	)

	srv.RegisterControllers(controller, redirectController)

	// Test GET handler
	t.Run("GET handler", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test", nil)
		response := executeRequest(req, srv)
		checkResponseCode(t, http.StatusOK, response.Code)
		assert.Contains(t, response.Body.String(), "Hello from index")
	})

	// Test POST handler
	t.Run("POST handler", func(t *testing.T) {
		req, _ := http.NewRequest("POST", "/test", nil)
		response := executeRequest(req, srv)
		checkResponseCode(t, http.StatusSeeOther, response.Code)
		assert.Equal(t, "/success", response.Header().Get("Location"))
	})

	// Test custom error handler
	t.Run("custom error handler", func(t *testing.T) {
		req, _ := http.NewRequest("DELETE", "/test", nil)
		response := executeRequest(req, srv)
		checkResponseCode(t, http.StatusMethodNotAllowed, response.Code)
		assert.Contains(t, response.Body.String(), "Custom Error: method not allowed")
	})

	// Test default error handler
	t.Run("default error handler", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/nonexistent", nil)
		response := executeRequest(req, srv)
		checkResponseCode(t, http.StatusNotFound, response.Code)
		assert.Contains(t, response.Body.String(), "page not found")
	})

	t.Run("issue internal error", func(t *testing.T) {
		req, _ := http.NewRequest("PUT", "/test", nil)
		response := executeRequest(req, srv)
		checkResponseCode(t, http.StatusInternalServerError, response.Code)
		assert.Contains(t, response.Body.String(), "internal server error")
	})

	t.Run("issue external error", func(t *testing.T) {
		req, _ := http.NewRequest("PATCH", "/test", nil)
		response := executeRequest(req, srv)
		checkResponseCode(t, http.StatusNotImplemented, response.Code)
		assert.Contains(t, response.Body.String(), "not implemented")
	})

	t.Run("redirect route", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/redirect", nil)
		response := executeRequest(req, srv)
		checkResponseCode(t, http.StatusSeeOther, response.Code)
		assert.Equal(t, fmt.Sprintf("%s/%s", cfg.Server.BaseURL, "test"), response.Header().Get("Location"))
	})
}

func TestMiddlewareApplication(t *testing.T) {
	d, bd := setupDirectories(t)
	cfg := setupConfig(d, bd)
	srv := setupMockServer(t, cfg)

	// Middleware that adds a custom header
	addCustomHeaderMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add("X-Custom-Header", "TestValue")
			next.ServeHTTP(w, r)
		})
	}

	// GET handler to test middleware
	getHandler := func(io HandlerIO, ctx context.Context) {
		err := io.Render(BaseTemplate, BaseTemplate, nil)
		assert.NoError(t, err)
	}

	// Register a controller with GET handler and middleware
	controller := NewController("test.route.middleware", "/test-middleware",
		Get(getHandler),
		WithMiddlewares(addCustomHeaderMiddleware),
		WithTemplaters(srv.Templaters()),
	)

	srv.RegisterControllers(controller)

	// Test middleware
	t.Run("middleware application", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/test-middleware", nil)
		response := executeRequest(req, srv)
		checkResponseCode(t, http.StatusOK, response.Code)
		assert.Contains(t, response.Body.String(), "Hello from index")
		assert.Equal(t, "TestValue", response.Header().Get("X-Custom-Header"))
	})
}

func TestServerErrorHandler(t *testing.T) {
	// Setup server with a custom server-wide error handler
	customServerErrorMsg := "Custom Server Error Handler Message"
	serverErrorHandler := func(err HandlerError) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, customServerErrorMsg, err.Status)
		}
	}

	cfg := setupConfig("", "")
	srv := NewServer(cfg, WithErrorHandler(serverErrorHandler))

	// Define a controller handler that triggers an error
	handlerThatErrors := func(io HandlerIO, ctx context.Context) {
		io.IssueError(ExtErr(errors.New("an error"), http.StatusInternalServerError, "should not see this message"))
	}

	controller := NewController("error.trigger", "/trigger-error", Get(handlerThatErrors))
	srv.RegisterControllers(controller)

	// Make a request to the handler that triggers an error
	req, _ := http.NewRequest("GET", "/trigger-error", nil)
	response := executeRequest(req, srv)
	checkResponseCode(t, http.StatusInternalServerError, response.Code)

	// Verify that the response matches the custom server error handler message
	assert.Contains(t, response.Body.String(), customServerErrorMsg)
}

func TestStdHandlerIO(t *testing.T) {
	dir, baseDir := setupDirectories(t)
	// This one needs to re-define index in order to print something because it inherits from base as it is loaded from
	// the BaseTemplate (index). All in all this templating stuff currently is rather unsatisfying.
	// It should be revisited and reworked in the future.
	tmplContent := "{{define \"index\"}}Sample Template Content{{end}}"
	err := os.WriteFile(filepath.Join(dir, "sample.go.html"), []byte(tmplContent), 0644)
	require.NoError(t, err)

	cfg := setupConfig(dir, baseDir)
	srv := setupMockServer(t, cfg)

	handler := func(io HandlerIO, ctx context.Context) {
		// Test Logger
		assert.NotNil(t, io.Logger())

		// Test Request
		assert.Equal(t, "GET", io.Request().Method)

		// Test SetHeader
		io.SetHeader("X-Test-Header", "TestValue")

		// Test Render
		err := io.Render("sample.go.html", BaseTemplate, nil)
		assert.NoError(t, err)
	}

	rawHandler := func(io HandlerIO, ctx context.Context) {
		io.Raw(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte("raw response"))
		})
	}

	failedRedirect := func(io HandlerIO, ctx context.Context) {
		err := io.RedirectRoute("non.existent", http.StatusSeeOther)
		assert.Error(t, err)

		err = io.RedirectRoute("raw.test", http.StatusSeeOther)
	}

	redirect := func(io HandlerIO, ctx context.Context) {
		err := io.RedirectRoute("raw.test", http.StatusSeeOther)
		assert.NoError(t, err)
	}

	controller := NewController("handlerio.test", "/handlerio", Get(handler), WithTemplaters(srv.Templaters()))
	rawController := NewController("raw.test", "/raw", Get(rawHandler))
	redirecter := NewController("redirect.test", "/redirect", Get(failedRedirect), Post(redirect))
	srv.RegisterControllers(controller, rawController, redirecter)

	t.Run("handlerio", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/handlerio", nil)
		response := executeRequest(req, srv)
		checkResponseCode(t, http.StatusOK, response.Code)

		assert.Equal(t, "TestValue", response.Header().Get("X-Test-Header"))
		assert.Contains(t, response.Body.String(), "Sample Template Content")
	})

	t.Run("raw", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/raw", nil)
		response := executeRequest(req, srv)
		checkResponseCode(t, http.StatusOK, response.Code)

		assert.Equal(t, "raw response", response.Body.String())
	})

	t.Run("redirect", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/redirect", nil)
		response := executeRequest(req, srv)
		checkResponseCode(t, http.StatusSeeOther, response.Code)

		req, _ = http.NewRequest("POST", "/redirect", nil)
		response = executeRequest(req, srv)
		checkResponseCode(t, http.StatusSeeOther, response.Code)
		assert.Equal(t, fmt.Sprintf("%s/%s", cfg.Server.BaseURL, "raw"), response.Header().Get("Location"))
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

// setupMockServer creates and returns a mock server.
func setupMockServer(t *testing.T, cfg *Cfg) *StdServer {
	mockTranslator := trans.NewTranslator()
	mockTemplater, err := NewTemplater(cfg.UI, mockTranslator, FromBaseTemplate())
	require.NoError(t, err)
	return NewServer(cfg, WithTemplater(mockTemplater, BaseTemplate))
}

// setupServerWithMocks creates and returns a server with mocks.
func setupServerWithMocks(t *testing.T, cfg *Cfg) *StdServer {
	mockRouter := chi.NewRouter()
	mockLogger := trace.NewTestLogger(t)
	mockTranslator := trans.NewTranslator()
	mockEventManger := event.NewEventManager(mockLogger)
	mockTemplater, err := NewTemplater(cfg.UI, mockTranslator, FromBaseTemplate())
	require.NoError(t, err)

	return NewServer(
		cfg,
		WithRouter(mockRouter),
		WithLogger(mockLogger),
		WithTranslator(mockTranslator),
		WithEventManger(mockEventManger),
		WithTemplater(mockTemplater, BaseTemplate),
	)
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

// setupAssetsFileServer sets up the assets directory with sample files and returns its path and a server configuration.
func setupAssetsFileServer(t *testing.T) *StdServer {
	assetsDir := setupAssetsDirectory(t)

	// Extend the existing setupConfig to include the AssetFsCfg
	cfg := setupConfig("", "")
	cfg.Server.AssetFsCfg = &FileServerCfg{
		Root:  assetsDir,
		Route: "/static",
	}

	return NewServer(cfg)
}

// executeRequest executes the request and returns the response recorder.
func executeRequest(req *http.Request, srv *StdServer) *httptest.ResponseRecorder {
	rr := httptest.NewRecorder()
	srv.router.ServeHTTP(rr, req)
	return rr
}

// checkResponseCode checks the response code and fails the test if it is not the expected code.
func checkResponseCode(t *testing.T, expected, actual int) {
	if expected != actual {
		t.Errorf("Expected response code %d. Got %d\n", expected, actual)
	}
}
