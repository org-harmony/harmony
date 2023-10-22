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

// TODO Add translations
// TODO Migrate to Bootstrap 5

func main() {
	l := trace.NewLogger()
	v := validator.New(validator.WithRequiredStructEnabled())
	t := trans.NewTranslator(trans.WithLogger(l))

	webCtx, r := initWeb(v, t)
	p, db := initDB(v)
	defer db.Close()
	appCtx := hctx.NewAppContext(l, v, p)

	web.RegisterHome(appCtx, webCtx)
	auth.RegisterAuth(appCtx, webCtx)

	util.Ok(web.Serve(r, webCtx.Configuration().Server))
}

func initWeb(v *validator.Validate, t trans.Translator) (web.Context, web.Router) {
	webCfg := &web.Cfg{}
	util.Ok(config.C(webCfg, config.From("web"), config.Validate(v)))
	store := util.Unwrap(web.SetupTemplaterStore(webCfg.UI, t))

	r := web.NewRouter()
	registerMiddlewares(r)

	web.MountFileServer(r, webCfg.Server.AssetFsCfg)

	webCtx := web.NewContext(r, webCfg, store)

	return webCtx, r
}

func initDB(v *validator.Validate) (persistence.RepositoryProvider, *pgxpool.Pool) {
	dbCfg := &persistence.Cfg{}
	util.Ok(config.C(dbCfg, config.From("persistence"), config.Validate(v)))
	db := util.Unwrap(persistence.NewDB(dbCfg.DB))

	return initRepositoryProvider(db), db
}

func initRepositoryProvider(db *pgxpool.Pool) persistence.RepositoryProvider {
	p := persistence.NewPGRepositoryProvider(db)

	util.Ok(p.RegisterRepository(func(db any) (persistence.Repository, error) {
		return auth.NewUserRepository(db.(*pgxpool.Pool)), nil
	}))
	util.Ok(p.RegisterRepository(func(db any) (persistence.Repository, error) {
		return auth.NewPGUserSessionRepository(db.(*pgxpool.Pool)), nil
	}))

	return p
}

func registerMiddlewares(r web.Router) {
	r.Use(web.CleanPath, web.Heartbeat("/ping"), web.Recoverer)
}
