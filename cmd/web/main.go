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

	baseT, lpT, err := templater(webCfg, translator)
	if err != nil {
		l.Error(WebMod, "failed to load templates", err)
		return
	}

	s := web.NewServer(
		webCfg,
		web.WithTemplater(baseT, web.BaseTemplate),
		web.WithTemplater(lpT, web.LandingPageTemplate),
		web.WithLogger(l),
		web.WithEventManger(em),
	)

	web.RegisterHome(s)
	auth.Setup(v)

	err = s.Serve(ctx)
	if err != nil {
		l.Error(WebMod, "failed to serve", err)
	}
}

func templater(cfg *web.Cfg, t trans.Translator) (base web.Templater, landingPage web.Templater, err error) {
	base, err = web.NewTemplater(cfg.UI, t, web.FromBaseTemplate())
	if err != nil {
		return
	}
	landingPage, err = web.NewTemplater(cfg.UI, t, web.FromLandingPageTemplate())
	if err != nil {
		return
	}
	return
}
