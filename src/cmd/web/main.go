package main

import (
	"github.com/jackc/pgx/v5/pgxpool"
	homeWeb "github.com/org-harmony/harmony/src/app/home"
	"github.com/org-harmony/harmony/src/app/template"
	"github.com/org-harmony/harmony/src/app/user"
	userWeb "github.com/org-harmony/harmony/src/app/user/web"
	"github.com/org-harmony/harmony/src/core/config"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/trace"
	"github.com/org-harmony/harmony/src/core/trans"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/validation"
	"github.com/org-harmony/harmony/src/core/web"
)

// TODO add comments for at least each exported function/method/type to follow go convention, if the element would not need a comment, it should not be exported(/existing)

func main() {
	logger := trace.NewLogger()
	validator := initValidator()

	provider, db := initDB(validator)
	defer db.Close()

	appCtx := hctx.NewAppCtx(logger, validator, provider)
	translatorProvider := initTrans(validator, logger)
	webCtx, r := initWeb(appCtx, validator, translatorProvider)

	homeWeb.RegisterController(appCtx, webCtx)
	userWeb.RegisterController(appCtx, webCtx)
	template.RegisterController(appCtx, webCtx)

	util.Ok(web.Serve(r, webCtx.Config.Server))
}

func initValidator() validation.V {
	return validation.New()
}

func initWeb(appCtx *hctx.AppCtx, v validation.V, tp trans.TranslatorProvider) (*web.Ctx, web.Router) {
	webCfg := &web.Cfg{}
	util.Ok(config.C(webCfg, config.From("web"), config.Validate(v)))
	store := util.Unwrap(web.SetupTemplaterStore(webCfg.UI))

	r := web.NewRouter()
	registerMiddlewares(appCtx, r, tp)

	web.MountFileServer(r, webCfg.Server.AssetFsCfg)

	webCtx := web.NewContext(r, webCfg, store)

	return webCtx, r
}

func initDB(v validation.V) (persistence.RepositoryProvider, *pgxpool.Pool) {
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
	util.Ok(p.RegisterRepository(func(db any) (persistence.Repository, error) {
		return template.NewRepository(db.(*pgxpool.Pool)), nil
	}))
	util.Ok(p.RegisterRepository(func(db any) (persistence.Repository, error) {
		return template.NewSetRepository(db.(*pgxpool.Pool)), nil
	}))

	return p
}

func initTrans(v validation.V, logger trace.Logger) trans.TranslatorProvider {
	transCfg := &trans.Cfg{}
	util.Ok(config.C(transCfg, config.From("trans"), config.Validate(v)))
	provider := util.Unwrap(trans.FromCfg(transCfg, logger))

	return provider
}

func registerMiddlewares(appCtx *hctx.AppCtx, r web.Router, translatorProvider trans.TranslatorProvider) {
	r.Use(
		web.Recoverer,
		web.Heartbeat("/ping"),
		web.CleanPath,
		user.LoggedInMiddleware(appCtx, user.AllowAnonymous),
		trans.Middleware(translatorProvider),
	)
}
