package main

import (
	"context"
	"github.com/org-harmony/harmony"

	"github.com/org-harmony/harmony/web"
)

const WebMod = "sys.cmd.web"

func main() {
	ctx := context.Background()
	l := harmony.NewStdLogger()
	em := harmony.NewEventManager(l)

	s := web.NewServer(web.WithEventManger(em))

	s.RegisterController(nil)

	err := s.Serve(ctx)
	if err != nil {
		l.Error(WebMod, "failed to serve", err)
	}
}
