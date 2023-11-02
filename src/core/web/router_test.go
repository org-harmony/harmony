package web

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"net/http/httptest"
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

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("GET", "/get", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "GET method", recorder.Body.String())

	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("POST", "/post", nil))
	assert.Equal(t, "POST method", recorder.Body.String())

	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("PUT", "/put", nil))
	assert.Equal(t, "PUT method", recorder.Body.String())

	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("DELETE", "/delete", nil))
	assert.Equal(t, "DELETE method", recorder.Body.String())

	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("PATCH", "/patch", nil))
	assert.Equal(t, "PATCH method", recorder.Body.String())
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
	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("GET", "/middleware", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "Applied", recorder.Header().Get("X-Middleware"))
	assert.Equal(t, "", recorder.Header().Get("X-Inline-Middleware"))

	// Test inline middleware
	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("GET", "/inline-middleware", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "Applied", recorder.Header().Get("X-Middleware"))
	assert.Equal(t, "InlineApplied", recorder.Header().Get("X-Inline-Middleware"))
}

func TestSubRouter(t *testing.T) {
	r, _ := setupMock(t)

	subRouter := NewRouter()
	subRouter.Handle("/subroute", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("SubRouter route"))
	})

	r.SubRouter("/sub", subRouter)

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("GET", "/sub/subroute", nil))
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "SubRouter route", recorder.Body.String())
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

	recorder := httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("GET", "/nonexistent", nil))
	assert.Equal(t, http.StatusNotFound, recorder.Code)
	assert.Equal(t, "Custom Not Found", recorder.Body.String())

	recorder = httptest.NewRecorder()
	r.ServeHTTP(recorder, httptest.NewRequest("POST", "/only-get", nil))
	assert.Equal(t, http.StatusMethodNotAllowed, recorder.Code)
	assert.Equal(t, "Custom Method Not Allowed", recorder.Body.String())
}
