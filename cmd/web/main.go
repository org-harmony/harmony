package main

import (
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/core/config"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/trans"
	"github.com/org-harmony/harmony/core/web"
)

const WebMod = "sys.cmd.web"

func main() {
	l := trace.NewLogger()
	v := validator.New(validator.WithRequiredStructEnabled())
	t := trans.NewTranslator()
	webCfg := &web.Cfg{}

	err := config.C(webCfg, config.From("web"), config.Validate(v))
	if err != nil {
		l.Error(WebMod, "failed to load config", err)
		return
	}

	store, err := web.SetupTemplaterStore(webCfg.UI, t)
	if err != nil {
		l.Error(WebMod, "failed to setup templater store", err)
		return
	}

	r := web.NewRouter()
	web.MountFileServer(r, webCfg.Server.AssetFsCfg)
	web.RegisterHome(r, store, l)

	err = web.Serve(r, webCfg.Server)
	if err != nil {
		l.Error(WebMod, "failed to serve web server", err)
		return
	}
}
