// Package web provides a web server implementation for HARMONY and declaring its interface.
// Also, the web package handles utility and required web functionality to allow domain packages
// to easily extend upon this package and allow web communication.
package web

import (
	"fmt"
	"github.com/org-harmony/harmony/core/trace"
	"net/http"
	"strings"
)

const Pkg = "sys.web"

// Cfg is the web packages configuration.
type Cfg struct {
	Server *ServerCfg `toml:"server" validate:"required"`
	UI     *UICfg     `toml:"ui" validate:"required"`
}

// ServerCfg contains the configuration for a web server.
type ServerCfg struct {
	AssetFsCfg *FileServerCfg `toml:"asset_fs" validate:"required"`
	Addr       string         `toml:"address" env:"ADDR"`
	Port       string         `toml:"port" env:"PORT" validate:"required"`
	BaseURL    string         `toml:"base_url" env:"BASE_URL" validate:"required"`
}

// FileServerCfg contains the configuration for a file server.
type FileServerCfg struct {
	Root  string `toml:"root" validate:"required"`
	Route string `toml:"route" validate:"required"`
}

// MountFileServer registers a file server with a config on a router.
func MountFileServer(r Router, cfg *FileServerCfg) {
	route := cfg.Route

	// Path Validation
	if strings.ContainsAny(route, "{}*") {
		panic("FileServer does not permit any URL parameters.")
	}

	// Path Adjustment and Redirection
	if route != "/" && route[len(route)-1] != '/' {
		r.Get(route, http.RedirectHandler(route+"/", 301).ServeHTTP)
		route += "/"
	}

	// Adjust the route to include a wildcard
	routeWithWildcard := route + "*"

	// Handling of GET requests
	r.Get(routeWithWildcard, func(w http.ResponseWriter, r *http.Request) {
		pathPrefix := strings.TrimSuffix(route, "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(http.Dir(cfg.Root)))
		fs.ServeHTTP(w, r)
	})
}

// Serve starts a web server with a router and config.
func Serve(r Router, cfg *ServerCfg) error {
	return http.ListenAndServe(fmt.Sprintf("%s:%s", cfg.Addr, cfg.Port), r)
}

// MaybeIntErr is a convenience function to log an error and return an internal server error.
// If the error is nil, it returns nil.
func MaybeIntErr(err error, l trace.Logger, w http.ResponseWriter, _ *http.Request) *HandlerError {
	if err == nil {
		return nil
	}

	l.Error(Pkg, "internal server error", err)
	e := IntErr()
	http.Error(w, e.Error(), e.Status)

	return &e
}

// RegisterHome registers the home page on a router.
func RegisterHome(r Router, store TemplaterStore, logger trace.Logger) {
	r.Get("/", home(store, logger))
}

// home returns a handler for the home page.
func home(store TemplaterStore, logger trace.Logger) http.HandlerFunc {
	lp, err := store.Templater(LandingPageTemplateName)
	if err != nil {
		panic(err)
	}

	t, err := lp.Template("home", "home.go.html")
	if err != nil {
		panic(err)
	}

	return func(w http.ResponseWriter, r *http.Request) {
		err := t.Execute(w, nil)
		if e := MaybeIntErr(err, logger, w, r); e != nil {
			return
		}
	}
}
