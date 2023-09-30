package main

import (
	"context"
	"fmt"
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony"
	"github.com/org-harmony/harmony/config"
	"github.com/org-harmony/harmony/web"
	"os"
)

const WebMod = "sys.cmd.web"

func main() {
	ctx := context.Background()
	l := harmony.NewLogger()
	em := harmony.NewEventManager(l)
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

	web.LoadConfig(v)

	err = s.Serve(ctx)
	if err != nil {
		l.Error(WebMod, "failed to serve", err)
	}
}
