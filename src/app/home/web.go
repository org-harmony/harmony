package home

import (
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/web"
)

// RegisterController registers the home controller and navigation.
func RegisterController(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	registerNavigation(appCtx, webCtx)

	webCtx.Router.Get("/", web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.Render(nil, "home", "home.go.html")
	}).ServeHTTP)
}

func registerNavigation(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	webCtx.Navigation.Add("home", web.NavItem{
		URL:  "/",
		Name: "harmony.menu.home",
		Display: func(io web.IO) (bool, error) {
			return true, nil
		},
		Position: 0,
	})
}
