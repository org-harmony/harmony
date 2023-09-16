package main

import (
	"context"
	"fmt"

	_ "github.com/org-harmony/harmony/cmd"
	"github.com/org-harmony/harmony/core"
	"github.com/org-harmony/harmony/trace"
	"github.com/org-harmony/harmony/web"
)

const MOD = "sys.web.main"

func main() {
	ctx := context.Background()
	args := core.ModLifecycleArgs{
		Logger: trace.NewStdLogger(),
	}

	var err error
	ctx, err = core.Modules.Setup(&args, ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to setup modules: %v", err))
	}

	ctx, err = core.Modules.Start(&args, ctx)
	if err != nil {
		panic(fmt.Sprintf("Failed to start modules: %v", err))
	}
	defer core.Modules.Stop(&args, ctx)

	args.Logger.Info(MOD, "started all modules")

	s := web.NewServer(&web.ServerConfig{
		Logger: args.Logger,
		Addr:   ":8080",
	}, ctx)
	err = s.Serve(ctx)
	if err != nil {
		panic(fmt.Sprintf("Serve failed: %v", err))
	}
}
