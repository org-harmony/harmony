// Package web provides a web server implementation for HARMONY and declaring its interface.
// Also, the web package handles utility and required web functionality to allow domain packages
// to easily extend upon this package and allow web communication.
package web

import "context"

const Pkg = "sys.web"

// Cfg is the web packages configuration.
type Cfg struct {
	Server *ServerCfg `toml:"server" validate:"required"`
	UI     *UICfg     `toml:"ui" validate:"required"`
}

// RegisterHome is a convenience function to register a home controller.
func RegisterHome(s Server) {
	s.RegisterControllers(
		NewController(
			"sys.home",
			"/",
			Get(home),
		),
	)
}

func home(io HandlerIO, _ context.Context) {
	if err := io.Render("auth/home.go.html", LandingPageTemplate, nil); err != nil {
		io.Logger().Error(Pkg, "failed to render home template", err)
		io.IssueError(IntErr())
	}
}
