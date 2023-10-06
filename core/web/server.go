package web

import (
	"context"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/org-harmony/harmony/core/event"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/trans"
	"html/template"
	"net/http"
	"strings"
	"sync"
)

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

// StdServer contains the configuration for a web server.
// It implements the Server interface.
type StdServer struct {
	router        chi.Router
	logger        trace.Logger
	translator    trans.Translator
	config        *ServerCfg        // config contains the web server's configuration.
	controllers   []*Controller     // controllers contains all registered entries of Controller.
	routeToPath   map[string]string // routeToPath maps a Controller route to a path.
	routeResolver RouteResolver     // routeResolver resolves a route to a path.
	errorHandler  ErrorHandler      // errorHandler handles a HandlerError.
	eventManager  *event.StdEventManager
	templaters    map[string]Templater // templaters allows using templates in the web server.
	lock          sync.RWMutex
}

// StdHandlerIO is passed to a Handler to allow handling of requests.
// See HandlerIO for more information.
type StdHandlerIO struct {
	logger        trace.Logger
	request       *http.Request
	header        http.Header
	handler       http.HandlerFunc
	errorHandler  ErrorHandler
	routeResolver RouteResolver
	templaters    map[string]Templater
}

// Controller contains the configuration for a Controller handling requests for a specific route
// with handlers for each HTTP verb (GET, POST, PUT, DELETE, PATCH).
// Controllers can not be nested. Earlier grouping functionalities were removed to favor simplicity.
//
// Controllers can also define middlewares and templates that are applied to all handlers in the Controller.
//
// Controller's seeks to provide a simple abstracted way to handle HTTP requests specific to the HARMONY project.
type Controller struct {
	Route       string                            // Route is the route for the Controller. Example: "auth.oauth.login"
	Path        string                            // Path is the path for the Controller. Example: "/oauth/login"
	get         Handler                           // Get handler for GET requests.
	post        Handler                           // Post handler for POST requests.
	put         Handler                           // Put handler for PUT requests.
	delete      Handler                           // Delete handler for DELETE requests.
	patch       Handler                           // Patch handler for PATCH requests.
	error       ErrorHandler                      // Error handler to handle a HandlerError route specific, default error page is used if not defined.
	middlewares []func(http.Handler) http.Handler // Middlewares are applied to all handlers in the Controller.
	templaters  map[string]Templater              // templaters allows using templates in the Controller.
}

// StdRouteResolver resolves a route to a path based on the registered Controllers on the referenced StdServer.
type StdRouteResolver struct {
	srv *StdServer
}

// ErrorHandler handles a HandlerError.
type ErrorHandler func(HandlerError) http.HandlerFunc

// Handler allows to handle a request via the Controller.
type Handler func(HandlerIO, context.Context)

// ServerOption is a function that configures an instance of StdServer.
type ServerOption func(*StdServer)

// ControllerOption configures a Controller.
type ControllerOption func(*Controller)

// Server is a web server working with middlewares, controllers and templates.
type Server interface {
	Serve(context.Context) error                           // Serve starts the web server.
	RegisterControllers(...*Controller)                    // RegisterControllers registers Controller's with the web server.
	RegisterErrorHandler(ErrorHandler)                     // RegisterErrorHandler registers an error handler.
	RegisterMiddleware(...func(http.Handler) http.Handler) // RegisterMiddleware registers route-specific middleware.
	ErrorHandler() ErrorHandler                            // ErrorHandler returns the ErrorHandler from the web server.
	Templaters() map[string]Templater                      // Templaters returns the instances of Templater associated with the web server.
}

// HandlerIO is passed to a Handler to allow handling of requests.
// It is an abstraction on top of the http.ResponseWriter and http.Request.
type HandlerIO interface {
	Logger() trace.Logger                                     // Logger returns the logger for the Handler.
	Request() *http.Request                                   // Request returns the request for the Handler.
	IssueError(HandlerError)                                  // IssueError issues a HandlerError to the client.
	Template(string, from string) (*template.Template, error) // Template returns a template by name and derived from another template.
	Render(name string, from string, data any) error          // Render renders a template by name with the provided data.
	RenderTemplate(tmpl *template.Template, data any)         // RenderTemplate renders a template with the provided data.
	SetHeader(key string, value string)                       // SetHeader sets a header on the response.
	Redirect(url string, code int)                            // Redirect redirects the client to the provided URL.
	RedirectRoute(route string, code int) error               // RedirectRoute redirects the client to the provided route.
	Raw(http.HandlerFunc)                                     // Raw allows for a raw http.HandlerFunc to be used, ignoring all other Handler functionality.
	handle() (http.HandlerFunc, error)                        // handle returns a http.HandlerFunc that handles the actual request.
}

// RouteResolver resolves a route to a path.
type RouteResolver interface {
	Resolve(route string) (string, error)
}

// WithRouter configures the chi.Router for the web server.
func WithRouter(r chi.Router) ServerOption {
	return func(s *StdServer) {
		s.router = r
	}
}

// WithLogger configures the trace.Logger for the web server.
func WithLogger(l trace.Logger) ServerOption {
	return func(s *StdServer) {
		s.logger = l
	}
}

// WithTranslator configure the trans.Translator for the web server.
func WithTranslator(t trans.Translator) ServerOption {
	return func(s *StdServer) {
		s.translator = t
	}
}

// WithTemplater configures a Templater for the web server.
func WithTemplater(t Templater, f string) ServerOption {
	return func(s *StdServer) {
		if s.templaters == nil {
			s.templaters = make(map[string]Templater)
		}

		s.templaters[f] = t
	}
}

// WithEventManger configures the event manager for the web server.
func WithEventManger(em *event.StdEventManager) ServerOption {
	return func(s *StdServer) {
		s.eventManager = em
	}
}

// WithErrorHandler configures the default ErrorHandler for the web server.
func WithErrorHandler(h ErrorHandler) ServerOption {
	return func(s *StdServer) {
		s.errorHandler = h
	}
}

// NewServer creates a new instance of StdServer.
// The server is configured using the provided ServerOption functions.
// If no ServerOption functions are provided the server is configured with default values from defaultStdServer.
func NewServer(config *Cfg, opts ...ServerOption) *StdServer {
	srv := defaultStdServer(config.Server)

	if config.Server.AssetFsCfg != nil {
		newFileServer(srv.router, config.Server.AssetFsCfg)
	}

	for _, o := range opts {
		o(srv)
	}

	return srv
}

// Serve starts the web server.
// It blocks until the server is stopped or an error occurs.
// Serve can be run in a goroutine to prevent blocking.
func (s *StdServer) Serve(ctx context.Context) error {
	s.logger.Info(Pkg, "starting web server...")

	return http.ListenAndServe(fmt.Sprintf("%s:%s", s.config.Addr, s.config.Port), s.router)
}

// RegisterControllers registers Controller's with the web server.
func (s *StdServer) RegisterControllers(c ...*Controller) {
	s.lock.Lock()
	defer s.lock.Unlock()

	for _, controller := range c {
		s.controllers = append(s.controllers, controller) // add to server's Controller list
		s.routeToPath[controller.Route] = controller.Path // add path to routeToPath map on server

		subRouter := chi.NewRouter()             // create a sub-router for the Controller
		subRouter.Use(controller.middlewares...) // apply Controller specific middlewares

		if controller.templaters == nil {
			controller.templaters = s.templaters // overwrite Controller Templaters with server's Templaters
		}

		if controller.error == nil {
			controller.error = s.errorHandler // overwrite Controller ErrorHandler with server's ErrorHandler
		}

		subRouter.MethodNotAllowed(
			throughErrorHandler(controller.error, ExtErr(nil, http.StatusMethodNotAllowed, "method not allowed")),
		)
		subRouter.NotFound(
			throughErrorHandler(controller.error, ExtErr(nil, http.StatusNotFound, "page not found")),
		)

		// register all HTTP verbs on the sub-router
		registerControllerHTTPVerbsOnSubRouter(controller, subRouter, s.logger, s.routeResolver)

		// mount the sub-router on the main router
		s.router.Mount(controller.Path, subRouter)
	}
}

// RegisterErrorHandler registers an error handler with the web server.
func (s *StdServer) RegisterErrorHandler(h ErrorHandler) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.errorHandler = h
}

// RegisterMiddleware registers middleware with the web server.
func (s *StdServer) RegisterMiddleware(m ...func(http.Handler) http.Handler) {
	s.lock.Lock()
	defer s.lock.Unlock()

	s.router.Use(m...)
}

// ErrorHandler returns the ErrorHandler from the web server.
func (s *StdServer) ErrorHandler() ErrorHandler {
	return s.errorHandler
}

// Templaters returns the instances of Templater associated with the web server.
func (s *StdServer) Templaters() map[string]Templater {
	return s.templaters
}

// WithTemplaters configures the templates for the Controller.
func WithTemplaters(templaters map[string]Templater) ControllerOption {
	return func(c *Controller) {
		c.templaters = templaters
	}
}

// WithMiddlewares configures the middleware for the Controller.
func WithMiddlewares(m ...func(http.Handler) http.Handler) ControllerOption {
	return func(c *Controller) {
		c.middlewares = append(c.middlewares, m...)
	}
}

// Get configures the GET handler for the Controller.
func Get(handler Handler) ControllerOption {
	return func(c *Controller) {
		c.get = handler
	}
}

// Post configures the POST handler for the Controller.
func Post(handler Handler) ControllerOption {
	return func(c *Controller) {
		c.post = handler
	}
}

// Put configures the PUT handler for the Controller.
func Put(handler Handler) ControllerOption {
	return func(c *Controller) {
		c.put = handler
	}
}

// Delete configures the DELETE handler for the Controller.
func Delete(handler Handler) ControllerOption {
	return func(c *Controller) {
		c.delete = handler
	}
}

// Patch configures the PATCH handler for the Controller.
func Patch(handler Handler) ControllerOption {
	return func(c *Controller) {
		c.patch = handler
	}
}

// Error configures the error handler for the Controller.
func Error(handler ErrorHandler) ControllerOption {
	return func(c *Controller) {
		c.error = handler
	}
}

// NewController creates a new instance of Controller.
// The Handler functions per HTTP verb should be configured using the ControllerOption functions:
// Get, Post, Put, Delete, Patch, Error (though not an HTTP verb).
//
// The templaters can directly be passed in from the server using the WithTemplaters ControllerOption function.
func NewController(route string, path string, opts ...ControllerOption) *Controller {
	c := &Controller{
		Route: route,
		Path:  path,
	}

	for _, opt := range opts {
		opt(c)
	}

	return c
}

// Logger returns the logger for the Controller.
func (io *StdHandlerIO) Logger() trace.Logger {
	return io.logger
}

// Request returns the request for the Controller.
func (io *StdHandlerIO) Request() *http.Request {
	return io.request
}

// IssueError issues a HandlerError to the client.
func (io *StdHandlerIO) IssueError(e HandlerError) {
	if io.errorHandler == nil {
		io.handler = func(w http.ResponseWriter, r *http.Request) {
			http.Error(w, e.Error(), e.Status)
		}
	}

	io.handler = io.errorHandler(e)
}

// Template returns a template (usually cloned) by name from the corresponding Templater if found.
//
// This template.Template is usually cloned within the Templater.Template() call.
func (io *StdHandlerIO) Template(name string, from string) (*template.Template, error) {
	templater, ok := io.templaters[from]
	if !ok {
		return nil, fmt.Errorf("templater %s not found", from)
	}

	tmpl, err := templater.Template(name)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve template: %w", err)
	}

	return tmpl, nil
}

// Render renders a template by name from the specified templater with the provided data.
func (io *StdHandlerIO) Render(name string, from string, data any) error {
	tmpl, err := io.Template(name, from)
	if err != nil {
		return err
	}

	io.RenderTemplate(tmpl, data)

	return nil
}

// RenderTemplate renders a template with the provided data.
func (io *StdHandlerIO) RenderTemplate(tmpl *template.Template, data any) {
	io.handler = func(w http.ResponseWriter, r *http.Request) {
		err := tmpl.Execute(w, data)
		if err == nil {
			w.Header().Set("Content-Type", "text/html; charset=utf-8")
			return
		}

		io.logger.Error(Pkg, "failed to render template", err)
		io.IssueError(IntErr())
	}
}

// SetHeader sets a header on the response.
func (io *StdHandlerIO) SetHeader(key string, value string) {
	io.header.Set(key, value)
}

// Redirect redirects the client to the provided URL.
func (io *StdHandlerIO) Redirect(url string, code int) {
	io.handler = func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, url, code)
	}
}

// RedirectRoute redirects the client to the provided route.
func (io *StdHandlerIO) RedirectRoute(route string, code int) error {
	url, err := io.routeResolver.Resolve(route)
	if err != nil {
		return err
	}

	io.Redirect(url, code)

	return nil
}

// Raw allows for raw a http.HandlerFunc to be used, ignoring all other Handler functionality.
// Raw should be used with caution as it bypasses all other Handler functionality.
func (io *StdHandlerIO) Raw(handlerFunc http.HandlerFunc) {
	io.handler = handlerFunc
}

// handle returns a http.HandlerFunc that handles the actual request.
func (io *StdHandlerIO) handle() (http.HandlerFunc, error) {
	if io.handler == nil {
		return nil, fmt.Errorf("no handler defined")
	}

	return io.handler, nil
}

// Resolve resolves a route to a path.
func (r *StdRouteResolver) Resolve(route string) (string, error) {
	r.srv.lock.RLock()
	defer r.srv.lock.RUnlock()

	path, ok := r.srv.routeToPath[route]
	if !ok {
		return "", fmt.Errorf("route %s not found", route)
	}

	base := strings.TrimSuffix(r.srv.config.BaseURL, "/")
	path = strings.TrimSuffix(path, "/")

	return base + path, nil
}

// defaultStdServer returns a new instance of ServerConfigs with default values.
func defaultStdServer(cfg *ServerCfg) *StdServer {
	srv := &StdServer{
		router:      chi.NewRouter(),
		logger:      trace.NewLogger(),
		translator:  trans.NewTranslator(),
		config:      cfg,
		routeToPath: make(map[string]string),
		errorHandler: func(handlerError HandlerError) http.HandlerFunc {
			return func(w http.ResponseWriter, r *http.Request) {
				http.Error(w, handlerError.Error(), handlerError.Status)
			}
		},
	}

	srv.routeResolver = &StdRouteResolver{srv: srv}

	return srv
}

// newFileServer registers a file server with a config on a router.
func newFileServer(r chi.Router, cfg *FileServerCfg) {
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
		rctx := chi.RouteContext(r.Context())
		pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
		fs := http.StripPrefix(pathPrefix, http.FileServer(http.Dir(cfg.Root)))
		fs.ServeHTTP(w, r)
	})
}

// registerControllerHTTPVerbsOnSubRouter registers the HTTP verbs from a Controller on a sub-router.
// This allows for the Controller specific chi.Router to be mounted on the main chi.Router.
// Thereby, allowing for custom middlewares along all handlers in the Controller.
func registerControllerHTTPVerbsOnSubRouter(c *Controller, sr chi.Router, logger trace.Logger, resolver RouteResolver) {
	if c.get != nil {
		sr.Get("/", func(w http.ResponseWriter, r *http.Request) {
			throughHandler(c.get, c.error, logger, resolver, c.templaters)(w, r)
		})
	}

	if c.post != nil {
		sr.Post("/", func(w http.ResponseWriter, r *http.Request) {
			throughHandler(c.post, c.error, logger, resolver, c.templaters)(w, r)
		})
	}

	if c.put != nil {
		sr.Put("/", func(w http.ResponseWriter, r *http.Request) {
			throughHandler(c.put, c.error, logger, resolver, c.templaters)(w, r)
		})
	}

	if c.delete != nil {
		sr.Delete("/", func(w http.ResponseWriter, r *http.Request) {
			throughHandler(c.delete, c.error, logger, resolver, c.templaters)(w, r)
		})
	}

	if c.patch != nil {
		sr.Patch("/", func(w http.ResponseWriter, r *http.Request) {
			throughHandler(c.patch, c.error, logger, resolver, c.templaters)(w, r)
		})
	}
}

// throughHandler wraps a Handler with the HandlerIO and returns a http.HandlerFunc.
func throughHandler(
	handler Handler,
	eHandler ErrorHandler,
	logger trace.Logger,
	resolver RouteResolver,
	templaters map[string]Templater,
) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handlerIO := &StdHandlerIO{
			logger:        logger,
			request:       r,
			header:        w.Header(),
			errorHandler:  eHandler,
			routeResolver: resolver,
			templaters:    templaters,
		}

		handler(handlerIO, r.Context())

		handler, err := handlerIO.handle()
		if err != nil {
			logger.Error(Pkg, "failed to handle request", err)
			http.Error(w, "internal server error - please review the logs", http.StatusInternalServerError)
			return
		}

		handler(w, r)
	}
}

// throughErrorHandler wraps an ErrorHandler with a HandlerError and returns a http.HandlerFunc.
func throughErrorHandler(handler ErrorHandler, error HandlerError) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		handler(error)(w, r)
	}
}
