package main

import (
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/core/auth"
	"github.com/org-harmony/harmony/core/config"
	"github.com/org-harmony/harmony/core/hctx"
	"github.com/org-harmony/harmony/core/persistence"
	"github.com/org-harmony/harmony/core/trace"
	"github.com/org-harmony/harmony/core/trans"
	"github.com/org-harmony/harmony/core/util"
	"github.com/org-harmony/harmony/core/web"
)

func main() {
	l := trace.NewLogger()
	v := validator.New(validator.WithRequiredStructEnabled())
	t := trans.NewTranslator(trans.WithLogger(l))
	webCfg := &web.Cfg{}
	util.Ok(config.C(webCfg, config.From("web"), config.Validate(v)))
	store := util.Unwrap(web.SetupTemplaterStore(webCfg.UI, t))

	r := web.NewRouter()
	web.MountFileServer(r, webCfg.Server.AssetFsCfg)

	dbCfg := &persistence.Cfg{}
	util.Ok(config.C(dbCfg, config.From("persistence"), config.Validate(v)))
	db := util.Unwrap(persistence.NewDB(dbCfg.DB))
	defer db.Close()

	p := initRepositoryProvider(db)

	appCtx := hctx.NewAppContext(l, v, p)
	webCtx := web.NewContext(r, webCfg, store)

	web.RegisterHome(appCtx, webCtx)
	auth.RegisterAuth(appCtx, webCtx)

	util.Ok(web.Serve(r, webCfg.Server))
}

func initRepositoryProvider(db *pgxpool.Pool) persistence.RepositoryProvider {
	p := persistence.NewPGRepositoryProvider(db)

	util.Ok(p.RegisterRepository(func(db any) (persistence.Repository, error) {
		return auth.NewUserRepository(db.(*pgxpool.Pool)), nil
	}))

	return p
}
