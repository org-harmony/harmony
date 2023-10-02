package web

import (
	"context"
	"github.com/go-chi/chi/v5"
	"github.com/org-harmony/harmony/core/event"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/trans"
	"html/template"
	"net/http"
	"path/filepath"
	"strings"
)

// ServerCfg contains the configuration for a web server.
type ServerCfg struct {
	AssetFsCfg *FileServerCfg `toml:"asset_fs" validate:"required"`
	Addr       string         `toml:"address" env:"ADDR"`
	Port       string         `toml:"port" env:"PORT" validate:"required"`
}

// FileServerCfg contains the configuration for a file server.
type FileServerCfg struct {
	Root  string `toml:"root" validate:"required"`
	Route string `toml:"route" validate:"required"`
}

// StdServer contains the configuration for a web server.
// It implements the Server interface.
type StdServer struct {
	config *ServerConfigs
}

// ServerConfigs contains the configuration for a web server.
type ServerConfigs struct {
	Router       chi.Router
	Logger       trace.Logger
	Addr         string
	EventManager *event.StdEventManager
	FileServer   *FileServerCfg
}

// ServerConfig is a function that configures a ServerConfigs.
// This follows the functional options pattern and is used to configure the web server
type ServerConfig func(*ServerConfigs)

// WithRouter configures the router for the web server.
func WithRouter(r chi.Router) ServerConfig {
	return func(cfg *ServerConfigs) {
		cfg.Router = r
	}
}

// WithLogger configures the logger for the web server.
func WithLogger(l trace.Logger) ServerConfig {
	return func(cfg *ServerConfigs) {
		cfg.Logger = l
	}
}

// WithAddr configures the address for the web server.
func WithAddr(addr string) ServerConfig {
	return func(cfg *ServerConfigs) {
		cfg.Addr = addr
	}
}

// WithEventManger configures the event manager for the web server.
func WithEventManger(em *event.StdEventManager) ServerConfig {
	return func(cfg *ServerConfigs) {
		cfg.EventManager = em
	}
}

// WithFileServer configures the file server for the web server.
func WithFileServer(cfg *FileServerCfg) ServerConfig {
	return func(c *ServerConfigs) {
		if cfg == nil {
			return
		}

		if c.Router == nil {
			c.Router = chi.NewRouter()
		}

		route := cfg.Route

		// Path Validation
		if strings.ContainsAny(route, "{}*") {
			panic("FileServer does not permit any URL parameters.")
		}

		// Path Adjustment and Redirection
		if route != "/" && route[len(route)-1] != '/' {
			c.Router.Get(route, http.RedirectHandler(route+"/", 301).ServeHTTP)
			route += "/"
		}

		// Adjust the route to include a wildcard
		routeWithWildcard := route + "*"

		// Handling of GET requests
		c.Router.Get(routeWithWildcard, func(w http.ResponseWriter, r *http.Request) {
			rctx := chi.RouteContext(r.Context())
			pathPrefix := strings.TrimSuffix(rctx.RoutePattern(), "/*")
			fs := http.StripPrefix(pathPrefix, http.FileServer(http.Dir(cfg.Root)))
			fs.ServeHTTP(w, r)
		})

		c.FileServer = cfg
	}
}

// defaultServerConfigs returns a new instance of ServerConfigs with default values.
func defaultServerConfigs() *ServerConfigs {
	return &ServerConfigs{
		Router: chi.NewRouter(),
		Logger: trace.NewLogger(),
		Addr:   ":8080",
	}
}

// NewServer creates a new instance of StdServer.
// The server is configured using the provided ServerConfig functions.
// If no ServerConfig functions are provided the server is configured with default values from defaultServerConfigs.
func NewServer(cfg ...ServerConfig) *StdServer {
	config := defaultServerConfigs()

	for _, f := range cfg {
		f(config)
	}

	return &StdServer{
		config: config,
	}
}

// Serve starts the web server.
// It blocks until the server is stopped or an error occurs.
// Serve can be run in a goroutine to prevent blocking.
func (s *StdServer) Serve(ctx context.Context) error {
	s.config.Logger.Info(Pkg, "starting web server...")

	return http.ListenAndServe(s.config.Addr, s.config.Router)
}

// RegisterController registers a controller with the web server.
func (s *StdServer) RegisterController(c ...Controller) {
	s.config.Router.Get("/", func(w http.ResponseWriter, r *http.Request) {
		tmpl := template.New("index.go.html")
		pathToAssets := s.config.FileServer.Route
		t := trans.NewTranslator()
		tmpl.Funcs(template.FuncMap{
			"t": func(s string, ctx context.Context) string {
				return t.T(s, ctx)
			},
			"tf": func(s string, ctx context.Context, args ...interface{}) string {
				return t.Tf(s, ctx, args...)
			},
			"html": func(s string) template.HTML {
				return template.HTML(s)
			},
			"asset": func(filename string) string {
				return filepath.Join(pathToAssets, filename)
			},
		})

		tmpl, err := tmpl.ParseFiles("core/web/tmpl/index.go.html")
		if err != nil {
			s.config.Logger.Error(Pkg, "failed to parse template", err)
		}

		w.Header().Set("Content-Type", "text/html; charset=utf-8")
		err = tmpl.Execute(w, nil)
		if err != nil {
			s.config.Logger.Error(Pkg, "failed to execute template", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	})
}

// RegisterMiddleware registers middleware with the web server.
func (s *StdServer) RegisterMiddleware(m ...func(http.Handler) http.Handler) {
	s.config.Router.Use(m...)
}
