package trans

import (
	"context"
	"github.com/org-harmony/harmony/src/core/util"
	"net/http"
)

// TODO add tests

// Middleware is part of the trans package and sets the locale in the request context.
// It requires a TranslatorProvider to be passed and uses it to choose the actual locale after checking the request.
// The locale is extracted from the request's cookie. If no cookie is present or the cookie is empty, the default locale is used.
func Middleware(provider TranslatorProvider) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		f := func(w http.ResponseWriter, r *http.Request) {
			locale := util.Unwrap(provider.Default())
			localeCookie, err := r.Cookie(LocaleSessionKey)

			if localeCookie != nil && err == nil && localeCookie.Value != "" {
				given, err := provider.Translator(localeCookie.Value)
				if err == nil {
					locale = given
				}
			}

			withLocale := context.WithValue(r.Context(), TranslatorContextKey, locale)
			r = r.WithContext(withLocale)

			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(f)
	}
}
