package main

import (
	"context"
	"fmt"

	_ "github.com/org-harmony/harmony/cmd"
	"github.com/org-harmony/harmony/core"
	"github.com/org-harmony/harmony/trace"
	"github.com/org-harmony/harmony/web"
)

const WEB_MOD = "sys.cmd.web"

func main() {
	var errs []error
	ctx := context.Background()
	args := core.ModLifecycleArgs{
		Logger: trace.NewStdLogger(),
	}
	m := core.Manager()

	errs = m.Setup(&args, ctx)
	if errs != nil {
		args.Logger.Error(WEB_MOD, "Failed to setup modules: %v", errs)
	}

	errs = m.Start(&args, ctx)
	if errs != nil {
		args.Logger.Error(WEB_MOD, "Failed to start modules: %v", errs)
	}
	defer m.Stop(&args)

	s := web.NewServer(&web.ServerConfig{
		Logger: args.Logger,
		Addr:   ":8080",
	}, ctx)

	err := s.Serve(ctx)
	if err != nil {
		panic(fmt.Sprintf("Serve failed: %v", errs))
	}
}
