package main

import (
	"context"
	"fmt"
	"github.com/org-harmony/harmony"
	"github.com/org-harmony/harmony/web"
	"os"
)

const WebMod = "sys.cmd.web"

func main() {
	ctx := context.Background()
	l := harmony.NewLogger()
	em := harmony.NewEventManager(l)

	err := harmony.LoadConfigToEnv("env")
	if err != nil {
		l.Error(WebMod, "failed to load config to env", err)
		return
	}

	s := web.NewServer(
		web.WithEventManger(em),
		web.WithAddr(fmt.Sprintf(":%s", os.Getenv("web.server.port"))),
	)

	s.RegisterController(nil)

	web.LoadConfig()

	err = s.Serve(ctx)
	if err != nil {
		l.Error(WebMod, "failed to serve", err)
	}
}
