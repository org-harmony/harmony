package trans

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMiddleware_SetsLocaleFromCookie(t *testing.T) {
	provider := NewMockProvider()
	middleware := Middleware(provider)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		translator := r.Context().Value(TranslatorContextKey).(Translator)
		assert.Equal(t, "de-DE", translator.Locale().Name)
	})

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: LocaleSessionKey, Value: "de-DE"})
	recorder := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestMiddleware_InvalidLocaleInCookie(t *testing.T) {
	provider := NewMockProvider()
	middleware := Middleware(provider)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		translator := r.Context().Value(TranslatorContextKey).(Translator)
		// Assuming "en-US" is the default locale set by the mock provider
		assert.Equal(t, "en-US", translator.Locale().Name)
	})

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	req.AddCookie(&http.Cookie{Name: LocaleSessionKey, Value: "invalid-locale"})
	recorder := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

func TestMiddleware_ErrorHandlingCookie(t *testing.T) {
	provider := NewMockProvider()
	middleware := Middleware(provider)

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Assuming "en-US" is the default locale set by the mock provider
		translator := r.Context().Value(TranslatorContextKey).(Translator)
		assert.Equal(t, "en-US", translator.Locale().Name)
	})

	wrappedHandler := middleware(handler)
	req := httptest.NewRequest("GET", "/", nil)
	// Simulate a cookie error by manipulating the request directly
	req.Header.Set("Cookie", "invalid-cookie")
	recorder := httptest.NewRecorder()
	wrappedHandler.ServeHTTP(recorder, req)

	assert.Equal(t, http.StatusOK, recorder.Code)
}

// NewMockProvider creates a mock translator provider with a predefined set of translators.
func NewMockProvider() TranslatorProvider {
	return &HTranslatorProvider{
		translators: map[string]Translator{
			"en-US": NewMockTranslator("en-US"),
			"de-DE": NewMockTranslator("de-DE"),
		},
		defaultTrans: NewMockTranslator("en-US"),
	}
}

// NewMockTranslator creates a mock translator for testing purposes.
func NewMockTranslator(localeName string) Translator {
	return &HTranslator{
		locale: &Locale{
			Name: localeName,
			Path: localeName,
		},
		translations: map[string]string{},
	}
}
