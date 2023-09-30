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
	"os"
)

const WebMod = "sys.cmd.web"

func main() {
	ctx := context.Background()
	l := trace.NewLogger()
	em := event.NewEventManager(l)
	v := validator.New(validator.WithRequiredStructEnabled())

	err := config.ToEnv(config.From("env"))
	if err != nil {
		l.Error(WebMod, "failed to load config to env", err)
		return
	}

	s := web.NewServer(
		web.WithEventManger(em),
		web.WithAddr(fmt.Sprintf(":%s", os.Getenv("WEB_SERVER_PORT"))),
	)

	s.RegisterController(nil)

	auth.LoadConfig(v)

	err = s.Serve(ctx)
	if err != nil {
		l.Error(WebMod, "failed to serve", err)
	}
}
