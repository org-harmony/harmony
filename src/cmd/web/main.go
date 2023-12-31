package main

import (
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/org-harmony/harmony/src/app/eiffel"
	homeWeb "github.com/org-harmony/harmony/src/app/home"
	"github.com/org-harmony/harmony/src/app/template"
	templateWeb "github.com/org-harmony/harmony/src/app/template/web"
	"github.com/org-harmony/harmony/src/app/user"
	userWeb "github.com/org-harmony/harmony/src/app/user/web"
	"github.com/org-harmony/harmony/src/core/config"
	"github.com/org-harmony/harmony/src/core/event"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/trace"
	"github.com/org-harmony/harmony/src/core/trans"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/validation"
	"github.com/org-harmony/harmony/src/core/web"
)

// TODO add larger integration/e2e tests for the web layer. Each controller and they're functions should be tested.
// TODO add module management to automatically register controllers and subscribe to events
// TODO evaluate events using code generation for type safety and performance
// TODO add extensive use of events for module management and all major application parts
// TODO add extensive debugging and tracing capabilities/tools/commands for events and modules to help with development
// TODO add more tests for the web layer
// TODO add utilities for easier testing of the web layer
// TODO improve UI/UX/Design (styling, css, scss)
// TODO add more loading indications, especially for loading body changes
// TODO improve user logged out handling during requests/responses and general site interaction/navigation
// TODO add cleanup task for expired sessions
// TODO add info for esfa about prozessbeschreibung being potentially long
// TODO add info for esfa about potentially complex <System> definition

func main() {
	logger := trace.NewLogger()
	validator := initValidator()
	eventManager := event.NewManager(logger)

	provider, db := initDB(validator)
	defer db.Close()

	appCtx := hctx.NewAppCtx(logger, validator, provider, eventManager)
	translatorProvider := initTrans(validator, logger)
	webCtx, r := initWeb(appCtx, validator, translatorProvider)

	homeWeb.RegisterController(appCtx, webCtx)
	userWeb.RegisterController(appCtx, webCtx)
	templateWeb.RegisterController(appCtx, webCtx)
	eiffel.RegisterController(appCtx, webCtx)

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
