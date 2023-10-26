package main

import (
	"github.com/go-playground/validator/v10"
	"github.com/jackc/pgx/v5/pgxpool"
	homeWeb "github.com/org-harmony/harmony/src/app/home"
	"github.com/org-harmony/harmony/src/app/user"
	userWeb "github.com/org-harmony/harmony/src/app/user/web"
	"github.com/org-harmony/harmony/src/core/config"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/trace"
	"github.com/org-harmony/harmony/src/core/trans"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/web"
)

// TODO move user to app and decouple from auth
// TODO Add translations
// TODO Add tests for new translation stuff
// TODO Migrate to Bootstrap 5

func main() {
	l := trace.NewLogger()
	v := validator.New(validator.WithRequiredStructEnabled())

	translatorProvider := initTrans(v, l)
	webCtx, r := initWeb(v, translatorProvider)
	p, db := initDB(v)
	defer db.Close()
	appCtx := hctx.NewAppCtx(l, v, p)

	homeWeb.RegisterController(appCtx, webCtx)
	userWeb.RegisterController(appCtx, webCtx)

	util.Ok(web.Serve(r, webCtx.Config.Server))
}

func initWeb(v *validator.Validate, tp trans.TranslatorProvider) (*web.Ctx, web.Router) {
	webCfg := &web.Cfg{}
	util.Ok(config.C(webCfg, config.From("web"), config.Validate(v)))
	store := util.Unwrap(web.SetupTemplaterStore(webCfg.UI))

	r := web.NewRouter()
	registerMiddlewares(r, tp)

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
		return user.NewUserRepository(db.(*pgxpool.Pool)), nil
	}))
	util.Ok(p.RegisterRepository(func(db any) (persistence.Repository, error) {
		return user.NewPGUserSessionRepository(db.(*pgxpool.Pool)), nil
	}))

	return p
}

func initTrans(v *validator.Validate, logger trace.Logger) trans.TranslatorProvider {
	transCfg := &trans.Cfg{}
	util.Ok(config.C(transCfg, config.From("trans"), config.Validate(v)))
	provider := util.Unwrap(trans.FromCfg(transCfg, logger))

	return provider
}

func registerMiddlewares(r web.Router, translatorProvider trans.TranslatorProvider) {
	r.Use(
		web.Recoverer,
		web.Heartbeat("/ping"),
		web.CleanPath,
		trans.Middleware(translatorProvider),
	)
}
