package main

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/core/auth"
	"github.com/org-harmony/harmony/core/config"
	"github.com/org-harmony/harmony/core/event"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/web"
)

const WebMod = "sys.cmd.web"

func main() {
	ctx := context.Background()
	l := trace.NewLogger()
	em := event.NewEventManager(l)
	v := validator.New(validator.WithRequiredStructEnabled())

	webCfg := &web.Cfg{}
	err := config.C(webCfg, config.From("web"), config.Validate(v))
	if err != nil {
		l.Error(WebMod, "failed to load config", err)
		return
	}
	s := web.NewServer(
		web.WithFileServer(webCfg.Server.AssetFsCfg),
		web.WithAddr(fmt.Sprintf("%s:%s", webCfg.Server.Addr, webCfg.Server.Port)),
		web.WithLogger(l),
		web.WithEventManger(em),
	)

	s.RegisterController(nil)

	auth.LoadConfig(v)

	err = s.Serve(ctx)
	if err != nil {
		l.Error(WebMod, "failed to serve", err)
	}
}
