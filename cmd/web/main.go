package main

import (
	"context"
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/core/auth"
	"github.com/org-harmony/harmony/core/config"
	"github.com/org-harmony/harmony/core/event"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/trans"
	"github.com/org-harmony/harmony/core/web"
)

const WebMod = "sys.cmd.web"

func main() {
	ctx := context.Background()
	l := trace.NewLogger()
	em := event.NewEventManager(l)
	v := validator.New(validator.WithRequiredStructEnabled())
	translator := trans.NewTranslator()

	webCfg := &web.Cfg{}
	err := config.C(webCfg, config.From("web"), config.Validate(v))
	if err != nil {
		l.Error(WebMod, "failed to load config", err)
		return
	}

	baseT, err := web.NewTemplater(webCfg.UI, translator, web.FromBaseTemplate())
	if err != nil {
		l.Error(WebMod, "failed to create base templater", err)
		return
	}
	lpT, err := web.NewTemplater(webCfg.UI, translator, web.FromLandingPageTemplate())
	if err != nil {
		l.Error(WebMod, "failed to create landing page templater", err)
		return
	}

	s := web.NewServer(
		webCfg,
		web.WithTemplater(baseT, web.BaseTemplate),
		web.WithTemplater(lpT, web.LandingPageTemplate),
		web.WithLogger(l),
		web.WithEventManger(em),
	)

	s.RegisterControllers(
		web.NewController(
			"sys.home",
			"/",
			web.WithTemplaters(s.Templaters()),
			web.Get(func(io web.HandlerIO, ctx context.Context) {
				if err := io.Render("auth/login.go.html", web.LandingPageTemplate, nil); err != nil {
					l.Error(WebMod, "failed to render home template", err)
					io.IssueError(web.IntErr())
				}
			}),
		),
	)

	auth.LoadConfig(v)

	err = s.Serve(ctx)
	if err != nil {
		l.Error(WebMod, "failed to serve", err)
	}
}
