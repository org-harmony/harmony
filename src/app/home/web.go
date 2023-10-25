package home

import (
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/web"
)

// RegisterController registers the home page on a router.
func RegisterController(appCtx *hctx.AppCtx, webCtx web.Context) {
	lp := util.Unwrap(webCtx.TemplaterStore().Templater(web.LandingPageTemplateName))
	t := util.Unwrap(lp.Template("home", "home.go.html"))

	webCtx.Router().Get("/", web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.Render(t, nil)
	}).ServeHTTP)
}
