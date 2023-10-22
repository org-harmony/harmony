package web

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

func TestBasicRouting(t *testing.T) {
	r, _ := setupMock(t)

	r.Get("/get", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GET method"))
	})
	r.Post("/post", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("POST method"))
	})
	r.Put("/put", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PUT method"))
	})
	r.Delete("/delete", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("DELETE method"))
	})
	r.Patch("/patch", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("PATCH method"))
	})

	req, _ := http.NewRequest("GET", "/get", nil)
	resp := executeRequest(req, r)
	checkResponseCode(t, http.StatusOK, resp.Code)
	assert.Equal(t, "GET method", resp.Body.String())

	req, _ = http.NewRequest("POST", "/post", nil)
	resp = executeRequest(req, r)
	checkResponseCode(t, http.StatusOK, resp.Code)
	assert.Equal(t, "POST method", resp.Body.String())

	req, _ = http.NewRequest("PUT", "/put", nil)
	resp = executeRequest(req, r)
	checkResponseCode(t, http.StatusOK, resp.Code)
	assert.Equal(t, "PUT method", resp.Body.String())

	req, _ = http.NewRequest("DELETE", "/delete", nil)
	resp = executeRequest(req, r)
	checkResponseCode(t, http.StatusOK, resp.Code)
	assert.Equal(t, "DELETE method", resp.Body.String())

	req, _ = http.NewRequest("PATCH", "/patch", nil)
	resp = executeRequest(req, r)
	checkResponseCode(t, http.StatusOK, resp.Code)
	assert.Equal(t, "PATCH method", resp.Body.String())
}

func TestMiddlewareApplication(t *testing.T) {
	r, _ := setupMock(t)

	// Global middleware
	middleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Middleware", "Applied")
			next.ServeHTTP(w, r)
		})
	}
	r.Use(middleware)
	r.Get("/middleware", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Middleware route"))
	})

	// Inline middleware using With(...)
	inlineMiddleware := func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-Inline-Middleware", "InlineApplied")
			next.ServeHTTP(w, r)
		})
	}
	r.With(inlineMiddleware).Get("/inline-middleware", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("Inline middleware route"))
	})

	// Test global middleware
	req, _ := http.NewRequest("GET", "/middleware", nil)
	resp := executeRequest(req, r)
	checkResponseCode(t, http.StatusOK, resp.Code)
	assert.Equal(t, "Applied", resp.Header().Get("X-Middleware"))
	assert.Equal(t, "", resp.Header().Get("X-Inline-Middleware"))

	// Test inline middleware
	req, _ = http.NewRequest("GET", "/inline-middleware", nil)
	resp = executeRequest(req, r)
	checkResponseCode(t, http.StatusOK, resp.Code)
	assert.Equal(t, "InlineApplied", resp.Header().Get("X-Inline-Middleware"))
}

func TestSubRouter(t *testing.T) {
	r, _ := setupMock(t)

	subRouter := NewRouter()
	subRouter.Handle("/subroute", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("SubRouter route"))
	})

	r.SubRouter("/sub", subRouter)

	req, _ := http.NewRequest("GET", "/sub/subroute", nil)
	resp := executeRequest(req, r)
	checkResponseCode(t, http.StatusOK, resp.Code)
	assert.Equal(t, "SubRouter route", resp.Body.String())
}

func TestNotFoundAndMethodNotAllowed(t *testing.T) {
	r, _ := setupMock(t)

	r.NotFound(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Custom Not Found"))
	})

	r.Get("/only-get", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("GET method"))
	})
	r.MethodNotAllowed(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusMethodNotAllowed)
		w.Write([]byte("Custom Method Not Allowed"))
	})

	req, _ := http.NewRequest("GET", "/nonexistent", nil)
	resp := executeRequest(req, r)
	checkResponseCode(t, http.StatusNotFound, resp.Code)
	assert.Equal(t, "Custom Not Found", resp.Body.String())

	req, _ = http.NewRequest("PATCH", "/only-get", nil)
	resp = executeRequest(req, r)
	checkResponseCode(t, http.StatusMethodNotAllowed, resp.Code)
	assert.Equal(t, "Custom Method Not Allowed", resp.Body.String())
}
