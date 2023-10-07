// Package web provides a web server implementation for HARMONY and declaring its interface.
// Also, the web package handles utility and required web functionality to allow domain packages
// to easily extend upon this package and allow web communication.
package web

import (
	"fmt"
	"github.com/org-harmony/harmony/core/ctx"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/util"
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

// Ctx is the web application context.
type Ctx struct {
	router Router
	cfg    *Cfg
	t      TemplaterStore
}

// Context is the web application context.
type Context interface {
	Router() Router                 // Router returns an instance of Router.
	Configuration() Cfg             // Configuration returns a copy of the web configuration.
	TemplaterStore() TemplaterStore // TemplaterStore returns an instance of TemplaterStore.
}

// NewContext returns a new Context.
func NewContext(router Router, cfg *Cfg, t TemplaterStore) Context {
	return &Ctx{
		router: router,
		cfg:    cfg,
		t:      t,
	}
}

// Router returns an instance of Router.
func (c *Ctx) Router() Router {
	return c.router
}

// Configuration returns a copy of the web configuration.
func (c *Ctx) Configuration() Cfg {
	return *c.cfg // return a copy
}

// TemplaterStore returns an instance of TemplaterStore.
func (c *Ctx) TemplaterStore() TemplaterStore {
	return c.t
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
func RegisterHome(app ctx.App, ctx Context) {
	ctx.Router().Get("/", home(ctx.TemplaterStore(), app.Logger()))
}

// home returns a handler for the home page.
func home(store TemplaterStore, logger trace.Logger) http.HandlerFunc {
	lp := util.Unwrap(store.Templater(LandingPageTemplateName))
	t := util.Unwrap(lp.Template("home", "home.go.html"))

	return func(w http.ResponseWriter, r *http.Request) {
		err := t.Execute(w, nil)
		_ = MaybeIntErr(err, logger, w, r)
	}
}
