// Package web provides a web server implementation for HARMONY and declaring its interface.
// Also, the web package handles utility and required web functionality to allow domain packages
// to easily extend upon this package and allow web communication.
package web

import (
	"context"
	"fmt"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/trace"
	"github.com/org-harmony/harmony/src/core/util"
	"html/template"
	"net/http"
	"strings"
)

const Pkg = "sys.web"

type Cfg struct {
	Server *ServerCfg `toml:"server" validate:"required"`
	UI     *UICfg     `toml:"ui" validate:"required"`
}

type ServerCfg struct {
	AssetFsCfg *FileServerCfg `toml:"asset_fs" validate:"required"`
	Addr       string         `toml:"address" env:"ADDR"`
	Port       string         `toml:"port" env:"PORT" validate:"required"`
	BaseURL    string         `toml:"base_url" env:"BASE_URL" validate:"required"`
}

type FileServerCfg struct {
	Root  string `toml:"root" validate:"required"`
	Route string `toml:"route" validate:"required"`
}

type Ctx struct {
	Router         Router
	Config         *Cfg
	TemplaterStore TemplaterStore
}

// Controller is convenience struct for handling web requests.
// The Controller is aware of the application context and the web context.
// The Controller implements the http.Handler interface and can therefore be used as a handler.
type Controller struct {
	app     *hctx.AppCtx
	ctx     *Ctx
	handler func(io IO) error
}

// HIO is a web.IO allowing for simplified access to the http.ResponseWriter and http.Request.
type HIO struct {
	w  http.ResponseWriter
	r  *http.Request
	l  trace.Logger
	t  TemplaterStore
	rt Router
}

// IO allows for simplified access to the http.ResponseWriter and http.Request.
// IO is passed to a Controller's handler function allowing the handler to interact with the http.ResponseWriter and http.Request.
// At the same time, IO allows the handler to interact with frequently used functionality such as logging and rendering.
type IO interface {
	Response() http.ResponseWriter
	Request() *http.Request
	Context() context.Context
	Logger() trace.Logger
	TemplaterStore() TemplaterStore
	Router() Router
	Render(*template.Template, any) error
	Error(error) error
	Redirect(string, int) error
	// Template returns a template by a name from the TemplateStore.
	// Template just wraps the call to TemplaterStore and consecutive Template call.
	Template(store, template, path string) (*template.Template, error)
}

func NewContext(router Router, cfg *Cfg, ts TemplaterStore) *Ctx {
	return &Ctx{
		Router:         router,
		Config:         cfg,
		TemplaterStore: ts,
	}
}

func NewController(app *hctx.AppCtx, ctx *Ctx, handler func(io IO) error) http.Handler {
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
		l:  c.app,
		t:  c.ctx.TemplaterStore,
		rt: c.ctx.Router,
	}

	err := c.handler(io)
	if err != nil {
		c.app.Error(Pkg, "internal server error executing handler", err)
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}
}

func (h *HIO) Response() http.ResponseWriter {
	return h.w
}

func (h *HIO) Request() *http.Request {
	return h.r
}

func (h *HIO) Context() context.Context {
	return h.r.Context()
}

func (h *HIO) Logger() trace.Logger {
	return h.l
}

func (h *HIO) TemplaterStore() TemplaterStore {
	return h.t
}

func (h *HIO) Router() Router {
	return h.rt
}

func (h *HIO) Render(t *template.Template, data any) error {
	if err := makeTemplateTranslatable(h.r.Context(), t); err != nil {
		h.l.Warn(Pkg, "failed to make template translatable, likely context does not contain translator", "error", err)
	}

	return util.Wrap(t.Execute(h.w, data), "failed to render template")
}

func (h *HIO) Error(e error) error {
	errTemplater, err := h.t.Templater(ErrorTemplateName)
	if err != nil {
		return err
	}

	errTemplate, err := errTemplater.Template("error", "error.go.html")
	if err != nil {
		return err
	}

	if err = makeTemplateTranslatable(h.r.Context(), errTemplate); err != nil {
		h.l.Warn(Pkg, "failed to make template translatable, likely context does not contain translator", "error", err)
	}

	return errTemplate.Execute(h.w, map[string]string{"Err": e.Error()})
}

func (h *HIO) Redirect(url string, code int) error {
	http.Redirect(h.w, h.r, url, code)
	return nil
}

// Template returns a template by a name from the TemplateStore.
// Template just wraps the call to TemplaterStore and consecutive Template call.
func (h *HIO) Template(store, template, path string) (*template.Template, error) {
	templaterStore, err := h.t.Templater(store)
	if err != nil {
		return nil, err
	}

	return templaterStore.Template(template, path)
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

func Serve(r Router, cfg *ServerCfg) error {
	return http.ListenAndServe(fmt.Sprintf("%s:%s", cfg.Addr, cfg.Port), r)
}
