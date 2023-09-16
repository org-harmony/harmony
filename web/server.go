package web

import (
	"context"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/org-harmony/harmony/trace"
)

const MOD = "sys.web.server"

type StdServer struct {
	r      chi.Router
	config *ServerConfig
}

type ServerConfig struct {
	Logger trace.Logger
	Addr   string
}

type Server interface {
	Serve(ctx context.Context) error
}

func NewServer(config *ServerConfig, ctx context.Context) *StdServer {
	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("hello world!"))
	})

	return &StdServer{
		r:      r,
		config: config,
	}
}

func (s *StdServer) Serve(ctx context.Context) error {
	s.config.Logger.Info(MOD, "starting web server...")

	return http.ListenAndServe(s.config.Addr, s.r)
}
