// Package web provides a web server implementation for HARMONY and declaring its interface.
// Also, the web package handles utility and required web functionality to allow domain packages
// to easily extend upon this package and allow web communication.
package web

import (
	"fmt"
	"github.com/org-harmony/harmony/core/ctx"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/util"
	"html/template"
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

// Controller is convenience struct for handling web requests.
// The Controller is aware of the application context and the web context.
// The Controller implements the http.Handler interface and can therefore be used as a handler.
type Controller struct {
	app     ctx.App
	ctx     Context
	handler func(io IO) error
}

// HIO is the implementation of the IO interface.
type HIO struct {
	w  http.ResponseWriter
	r  *http.Request
	l  trace.Logger
	t  TemplaterStore
	rt Router
}

// Context is the web application context.
type Context interface {
	Router() Router                 // Router returns an instance of Router.
	Configuration() Cfg             // Configuration returns a copy of the web configuration.
	TemplaterStore() TemplaterStore // TemplaterStore returns an instance of TemplaterStore.
}

// IO allows for simplified access to the http.ResponseWriter and http.Request.
// IO is passed to a Controller's handler function allowing the handler to interact with the http.ResponseWriter and http.Request.
// At the same time, IO allows the handler to interact with frequently used functionality such as logging and rendering.
type IO interface {
	Writer() http.ResponseWriter           // Writer returns the http.ResponseWriter.
	Request() *http.Request                // Request returns the http.Request.
	Logger() trace.Logger                  // Logger returns the application logger.
	TemplaterStore() TemplaterStore        // TemplaterStore returns an instance of TemplaterStore.
	Router() Router                        // Router returns an instance of Router.
	Render(*template.Template, any) error  // Render renders a template with data.
	Error(*template.Template, error) error // Error renders an error template with an error.
	Redirect(string, int) error            // Redirect redirects the client to a URL with a status code.
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

// NewController returns a new Controller.
func NewController(app ctx.App, ctx Context, handler func(io IO) error) http.Handler {
	if app == nil || ctx == nil || handler == nil {
		panic("nil contexts or handler")
	}

	return &Controller{
		app:     app,
		ctx:     ctx,
		handler: handler,
	}
}

// ServeHTTP implements the http.Handler interface. It executes the handler function and handles any errors.
// If an error occurs, the error is logged and an internal server error is returned to the client.
func (c *Controller) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io := &HIO{
		w:  w,
		r:  r,
		l:  c.app.Logger(),
		t:  c.ctx.TemplaterStore(),
		rt: c.ctx.Router(),
	}

	err := c.handler(io)
	if err != nil {
		c.app.Logger().Error(Pkg, "internal server error executing handler", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

// Writer returns the http.ResponseWriter.
func (h *HIO) Writer() http.ResponseWriter {
	return h.w
}

// Request returns the http.Request.
func (h *HIO) Request() *http.Request {
	return h.r
}

// Logger returns the application logger.
func (h *HIO) Logger() trace.Logger {
	return h.l
}

// TemplaterStore returns an instance of TemplaterStore.
func (h *HIO) TemplaterStore() TemplaterStore {
	return h.t
}

// Router returns an instance of Router.
func (h *HIO) Router() Router {
	return h.rt
}

// Render renders a template with data. If an error occurs, the error is returned.
func (h *HIO) Render(t *template.Template, data any) error {
	return util.Wrap(t.Execute(h.w, data), "failed to render template")
}

// Error renders an error template with an error. If an error occurs, the error is returned.
func (h *HIO) Error(t *template.Template, e error) error {
	return util.Wrap(t.Execute(h.w, NewErrorTemplateData(h.r.Context(), e.Error())), "failed to render error template")
}

// Redirect redirects the client to a URL with a status code.
func (h *HIO) Redirect(url string, code int) error {
	http.Redirect(h.w, h.r, url, code)
	return nil
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

// RegisterHome registers the home page on a router.
func RegisterHome(app ctx.App, ctx Context) {
	lp := util.Unwrap(ctx.TemplaterStore().Templater(LandingPageTemplateName))
	t := util.Unwrap(lp.Template("home", "home.go.html"))

	ctx.Router().Get("/", NewController(app, ctx, func(io IO) error {
		return io.Render(t, nil)
	}).ServeHTTP)
}
