package main

import (
	"github.com/go-playground/validator/v10"
	"github.com/org-harmony/harmony/core/auth"
	"github.com/org-harmony/harmony/core/config"
	"github.com/org-harmony/harmony/core/ctx"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/trans"
	"github.com/org-harmony/harmony/core/util"
	"github.com/org-harmony/harmony/core/web"
)

const WebMod = "sys.cmd.web"

func main() {
	l := trace.NewLogger()
	v := validator.New(validator.WithRequiredStructEnabled())
	t := trans.NewTranslator()
	webCfg := &web.Cfg{}
	util.Ok(config.C(webCfg, config.From("web"), config.Validate(v)))
	store := util.Unwrap(web.SetupTemplaterStore(webCfg.UI, t))

	r := web.NewRouter()
	web.MountFileServer(r, webCfg.Server.AssetFsCfg)

	app := ctx.NewApp(l, t, v)
	ctx := web.NewContext(r, webCfg, store)

	web.RegisterHome(app, ctx)
	auth.RegisterAuth(app, ctx)

	util.Ok(web.Serve(r, webCfg.Server))
}
