package main

import (
	"context"
	"fmt"

	_ "github.com/org-harmony/harmony/cmd"
	"github.com/org-harmony/harmony/core"
	"github.com/org-harmony/harmony/trace"
	"github.com/org-harmony/harmony/web"
)

const WebMod = "sys.cmd.web"

func main() {
	var errs []error
	ctx := context.Background()
	args := core.ModLifecycleArgs{
		Logger: trace.NewStdLogger(),
	}
	m := core.Manager()
	em := core.NewStdEventManager(args.Logger)

	errs = m.Setup(&args, ctx)
	if errs != nil {
		args.Logger.Error(WebMod, "Failed to setup modules: %v", errs)
	}

	errs = m.Start(&args, ctx)
	if errs != nil {
		args.Logger.Error(WebMod, "Failed to start modules: %v", errs)
	}
	defer m.Stop(&args)

	s := web.NewStdServer(web.WithEventManger(em))

	s.RegisterController(nil)

	err := s.Setup(ctx)
	if errs != nil {
		panic(fmt.Sprintf("Setup failed: %v", errs))
	}

	err = s.Serve(ctx)
	if err != nil {
		panic(fmt.Sprintf("Serve failed: %v", errs))
	}
}
