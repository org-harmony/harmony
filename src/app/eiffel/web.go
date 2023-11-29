package eiffel

import (
	"encoding/json"
	"github.com/org-harmony/harmony/src/app/template"
	"github.com/org-harmony/harmony/src/app/user"
	"github.com/org-harmony/harmony/src/core/event"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/validation"
	"github.com/org-harmony/harmony/src/core/web"
	"net/http"
	"strings"
)

func RegisterController(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	// TODO move this to module init when module manager is implemented (see subscribeEvents)
	subscribeEvents(appCtx)

	registerNavigation(appCtx, webCtx)

	router := webCtx.Router.With(user.LoggedInMiddleware(appCtx))

	router.Get("/eiffel", eiffelElicitationPage(appCtx, webCtx).ServeHTTP)
	router.Get("/eiffel/elicitation/templates/search/modal", searchModal(appCtx, webCtx).ServeHTTP)
	router.Get("/eiffel/elicitation/{templateID}/{variant}", elicitationTemplate(appCtx, webCtx).ServeHTTP)
	router.Post("/eiffel/elicitation/{templateID}/{variant}", parseRequirement(appCtx, webCtx).ServeHTTP)
	router.Get("/eiffel/elicitation/output/file/form", outputFileForm(appCtx, webCtx).ServeHTTP)
}

func subscribeEvents(appCtx *hctx.AppCtx) {
	// TODO remove this with module manager
	appCtx.EventManager.Subscribe("template.config.validate", func(event event.Event, args *event.PublishArgs) error {
		validateEvent, ok := event.Payload().(*template.ValidateTemplateConfigEvent)
		if !ok {
			return nil
		}
		if strings.ToLower(validateEvent.TemplateType) != BasicTemplateType {
			return nil
		}
		if validateEvent.DidValidate {
			return nil
		}
		validateEvent.DidValidate = true

		ebt := &BasicTemplate{}
		// Important notice: Unmarshalling is always case-insensitive if no other match could be found.
		// Therefore, NAME will be unmarshalled to Name. Keep this in mind.
		err := json.Unmarshal([]byte(validateEvent.Config), ebt)
		if err != nil {
			return err
		}

		validationErrs := ebt.Validate(appCtx.Validator, RuleParsers())
		if len(validationErrs) > 0 {
			validateEvent.AddErrors(validationErrs...)
			validateEvent.AddErrors(validation.Error{Msg: "eiffel.parser.error.invalid-template"})
			return nil
		}

		return nil
	}, event.DefaultEventPriority)
}

func registerNavigation(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	webCtx.Navigation.Add("eiffel.elicitation", web.NavItem{
		URL:  "/eiffel",
		Name: "harmony.menu.eiffel",
		Display: func(io web.IO) (bool, error) {
			return true, nil
		},
		Position: 100,
	})
}

func eiffelElicitationPage(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.Render(
			nil,
			"eiffel.elicitation.page",
			"eiffel/elicitation-page.go.html",
			"eiffel/elicitation-template.go.html",
			"eiffel/_form-elicitation.go.html",
			"eiffel/_form-output-file.go.html",
		)
	})
}

func searchModal(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.Render(nil, "eiffel.template.search.modal", "eiffel/_modal-template-search.go.html")
	})
}

func elicitationTemplate(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.Render(
			nil,
			"eiffel.elicitation.template",
			"eiffel/elicitation-template.go.html",
			"eiffel/_form-elicitation.go.html",
		)
	})
}

func parseRequirement(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.Render(nil, "eiffel.elicitation.form", "eiffel/_form-elicitation.go.html")
	})
}

func outputFileForm(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.Render(nil, "eiffel.elicitation.output-file.form", "eiffel/_form-output-file.go.html")
	})
}
