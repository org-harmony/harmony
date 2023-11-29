package eiffel

import (
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/org-harmony/harmony/src/app/template"
	"github.com/org-harmony/harmony/src/app/user"
	"github.com/org-harmony/harmony/src/core/event"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/validation"
	"github.com/org-harmony/harmony/src/core/web"
	"net/http"
	"strings"
)

const (
	// TemplateDisplayString will display the rule value as text.
	TemplateDisplayString TemplateDisplayType = "text"
	// TemplateDisplayInputTypeText will display the rule value as a text input.
	TemplateDisplayInputTypeText TemplateDisplayType = "input-text"
	// TemplateDisplayInputTypeTextarea will display the rule value as a textarea.
	TemplateDisplayInputTypeTextarea TemplateDisplayType = "input-textarea"
	// TemplateDisplayInputTypeSingleSelect will display the rule value as an input field with datalist and single select.
	TemplateDisplayInputTypeSingleSelect TemplateDisplayType = "input-single-select"
)

var (
	// ErrTemplateNotFound will be displayed to the user if the template could not be found.
	ErrTemplateNotFound = errors.New("eiffel.elicitation.template.not-found")
	// ErrTemplateVariantNotFound will be displayed to the user if the template variant could not be found.
	ErrTemplateVariantNotFound = errors.New("eiffel.elicitation.template.variant.not-found")
)

// TemplateDisplayType specifies how a rule should be displayed in the UI.
type TemplateDisplayType string

// TemplateFormData is the data that is passed to the template rendering the elicitation form.
type TemplateFormData struct {
	Template *BasicTemplate
	// DisplayTypes is a map of rule names to display types. The rule names are the keys of the BasicTemplate.Rules map.
	// The display types are used to determine how the rule should be displayed in the UI.
	DisplayTypes map[string]TemplateDisplayType
}

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
			web.NewFormData(TemplateFormData{}, nil),
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
	templateRepository := util.UnwrapType[template.Repository](appCtx.Repository(template.RepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		templateID := web.URLParam(io.Request(), "templateID")
		variant := web.URLParam(io.Request(), "variant")

		templateUUID, err := uuid.Parse(templateID)
		if err != nil {
			return io.InlineError(ErrTemplateNotFound, err)
		}

		tmpl, err := templateRepository.FindByID(io.Context(), templateUUID)
		if err != nil {
			return io.InlineError(ErrTemplateNotFound, err)
		}

		bt, err := TemplateIntoBasicTemplate(tmpl, appCtx.Validator, RuleParsers())
		if err != nil {
			return io.InlineError(err)
		}

		if _, ok := bt.Variants[variant]; !ok {
			return io.InlineError(ErrTemplateVariantNotFound)
		}

		displayTypes := TemplateDisplayTypes(bt, RuleParsers())

		return io.Render(
			web.NewFormData(TemplateFormData{Template: bt, DisplayTypes: displayTypes}, nil),
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
