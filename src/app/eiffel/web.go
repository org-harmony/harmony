package eiffel

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/org-harmony/harmony/src/app/template"
	"github.com/org-harmony/harmony/src/app/template/parser"
	"github.com/org-harmony/harmony/src/app/user"
	"github.com/org-harmony/harmony/src/core/config"
	"github.com/org-harmony/harmony/src/core/event"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/validation"
	"github.com/org-harmony/harmony/src/core/web"
	"net/http"
	"path/filepath"
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
	// ErrInvalidFilepath will be displayed to the user if the filepath is invalid.
	ErrInvalidFilepath = errors.New("eiffel.elicitation.output.file.path-invalid")
	// ErrCouldNotCreateFile will be displayed to the user if the file could not be created.
	ErrCouldNotCreateFile = errors.New("eiffel.elicitation.output.file.create-failed")
)

// TemplateDisplayType specifies how a rule should be displayed in the UI.
type TemplateDisplayType string

// TemplateFormData is the data that is passed to the template rendering the elicitation form.
type TemplateFormData struct {
	Template *BasicTemplate
	// Variant is the currently selected variant. This might be through a specified variant name parameter
	// or as a default value because no variant was explicitly specified. However, Variant is expected to be filled.
	Variant *BasicVariant
	// VariantKey is the key through which the variant was selected. It is not the name of the variant.
	// If the variant was auto-selected as default the key will still be filled.
	VariantKey string
	// DisplayTypes is a map of rule names to display types. The rule names are the keys of the BasicTemplate.Rules map.
	// The display types are used to determine how the rule should be displayed in the UI.
	DisplayTypes map[string]TemplateDisplayType
	// TemplateID is the ID of the template that is currently being rendered.
	TemplateID uuid.UUID
	// CopyAfterParse is a flag indicating if the user wants to copy the parsed requirement to the clipboard.
	CopyAfterParse bool
	// ParsingResult is the result of the parsing process. This can be empty if no parsing was done yet.
	ParsingResult *parser.ParsingResult
	// SegmentMap is a map of rule names to their corresponding segment value.
	// This is used to fill the segments with their values after parsing.
	SegmentMap map[string]string
	// OutputFormData is the form data for the output directory and file.
	OutputFormData web.BaseTemplateData
}

// SearchTemplateData contains templates to render as search results and a flag indicating if the query was too short.
type SearchTemplateData struct {
	Templates     []*template.Template
	QueryTooShort bool
}

// OutputFormData is the form data for the output directory and file search form.
// It is used to fill the form with the previously selected values.
type OutputFormData struct {
	OutputDir  string
	OutputFile string
}

// RegisterController registers the controllers as well as the navigation and event listeners.
func RegisterController(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	cfg := Cfg{}
	util.Ok(config.C(&cfg, config.From("eiffel"), config.Validate(appCtx.Validator)))

	// TODO move this to module init when module manager is implemented (see subscribeEvents)
	subscribeEvents(appCtx)

	registerNavigation(appCtx, webCtx)

	router := webCtx.Router.With(user.LoggedInMiddleware(appCtx))

	router.Get("/eiffel", eiffelElicitationPage(appCtx, webCtx).ServeHTTP)
	router.Get("/eiffel/{templateID}", eiffelElicitationPage(appCtx, webCtx).ServeHTTP)
	router.Get("/eiffel/{templateID}/{variant}", eiffelElicitationPage(appCtx, webCtx).ServeHTTP)
	router.Get("/eiffel/elicitation/templates/search/modal", searchModal(appCtx, webCtx).ServeHTTP)
	router.Post("/eiffel/elicitation/templates/search", searchTemplate(appCtx, webCtx).ServeHTTP)
	router.Get("/eiffel/elicitation/{templateID}", elicitationTemplate(appCtx, webCtx, true).ServeHTTP)
	router.Get("/eiffel/elicitation/{templateID}/{variant}", elicitationTemplate(appCtx, webCtx, false).ServeHTTP)
	router.Post("/eiffel/elicitation/{templateID}/{variant}", parseRequirement(appCtx, webCtx, cfg).ServeHTTP)
	router.Post("/eiffel/elicitation/output/search", outputSearchForm(appCtx, webCtx, cfg).ServeHTTP)
	router.Post("/eiffel/elicitation/output/dir/search", outputSearchDir(appCtx, webCtx, cfg).ServeHTTP)
	router.Post("/eiffel/elicitation/output/file/search", outputFileSearch(appCtx, webCtx, cfg).ServeHTTP)
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
	templateRepository := util.UnwrapType[template.Repository](appCtx.Repository(template.RepositoryName))
	sessionStore := util.UnwrapType[user.SessionRepository](appCtx.Repository(user.SessionRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		templateID := web.URLParam(io.Request(), "templateID")
		variantKey := web.URLParam(io.Request(), "variant")
		if templateID == "" {
			return renderElicitationPage(io, TemplateFormData{}, nil, nil)
		}

		formData, err := TemplateFormFromRequest(
			io.Context(),
			templateID,
			variantKey,
			templateRepository,
			RuleParsers(),
			appCtx.Validator,
			true,
		)

		formData.CopyAfterParse = CopyAfterParseSetting(io.Request(), sessionStore, true)

		return renderElicitationPage(io, formData, nil, []error{err})
	})
}

func renderElicitationPage(io web.IO, data TemplateFormData, success []string, errs []error) error {
	return io.Render(
		web.NewFormData(data, success, errs...),
		"eiffel.elicitation.page",
		"eiffel/elicitation-page.go.html",
		"eiffel/_elicitation-template.go.html",
		"eiffel/_form-elicitation.go.html",
		"eiffel/_form-output-file.go.html",
	)
}

func searchModal(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.Render(nil, "eiffel.template.search.modal", "eiffel/_modal-template-search.go.html")
	})
}

func searchTemplate(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateRepository := util.UnwrapType[template.Repository](appCtx.Repository(template.RepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		request := io.Request()
		err := request.ParseForm()
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		query := request.FormValue("search")
		if len(query) < 3 {
			return io.Render(
				&SearchTemplateData{QueryTooShort: true},
				"eiffel.template.search.result",
				"eiffel/_template-search-result.go.html",
			)
		}

		templates, err := templateRepository.FindByQueryForType(io.Context(), query, BasicTemplateType)
		if err != nil && !errors.Is(err, persistence.ErrNotFound) {
			return io.InlineError(web.ErrInternal, err)
		}

		return io.Render(
			&SearchTemplateData{Templates: templates},
			"eiffel.template.search.result",
			"eiffel/_template-search-result.go.html",
		)
	})
}

func elicitationTemplate(appCtx *hctx.AppCtx, webCtx *web.Ctx, defaultFirstVariant bool) http.Handler {
	templateRepository := util.UnwrapType[template.Repository](appCtx.Repository(template.RepositoryName))
	sessionStore := util.UnwrapType[user.SessionRepository](appCtx.Repository(user.SessionRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		templateID := web.URLParam(io.Request(), "templateID")
		variant := web.URLParam(io.Request(), "variant")

		io.Response().Header().Set("HX-Push-URL", fmt.Sprintf("/eiffel/%s", templateID))

		formData, err := TemplateFormFromRequest(
			io.Context(),
			templateID,
			variant,
			templateRepository,
			RuleParsers(),
			appCtx.Validator,
			defaultFirstVariant,
		)
		if err != nil {
			return io.InlineError(err)
		}

		formData.CopyAfterParse = CopyAfterParseSetting(io.Request(), sessionStore, true)

		io.Response().Header().Set("HX-Push-URL", fmt.Sprintf("/eiffel/%s/%s", templateID, formData.VariantKey))

		return io.Render(
			web.NewFormData(formData, nil),
			"eiffel.elicitation.template",
			"eiffel/_elicitation-template.go.html",
			"eiffel/_form-elicitation.go.html",
		)
	})
}

func parseRequirement(appCtx *hctx.AppCtx, webCtx *web.Ctx, cfg Cfg) http.Handler {
	templateRepository := util.UnwrapType[template.Repository](appCtx.Repository(template.RepositoryName))
	sessionStore := util.UnwrapType[user.SessionRepository](appCtx.Repository(user.SessionRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		request := io.Request()
		ctx := request.Context()
		parsers := RuleParsers()

		templateID := web.URLParam(request, "templateID")
		variant := web.URLParam(request, "variant")
		outputDir := BuildDirPath(cfg.Output.BaseDir, request.FormValue("elicitationOutputDir"))
		outputFileRaw := request.FormValue("elicitationOutputFile")

		formData, err := TemplateFormFromRequest(
			ctx,
			templateID,
			variant,
			templateRepository,
			parsers,
			appCtx.Validator,
			false,
		)
		if err != nil {
			return io.InlineError(err)
		}

		if outputFileRaw == "" {
			outputFileRaw = formData.Template.Name
		}
		outputFile := BuildFilename(outputFileRaw)

		segmentMap, err := SegmentMapFromRequest(request, len(formData.Variant.Rules))
		if err != nil {
			return io.InlineError(web.ErrInternal)
		}
		formData.SegmentMap = segmentMap

		parsingResult, err := formData.Template.Parse(ctx, parsers, formData.VariantKey, SegmentMapToSegments(segmentMap)...)
		formData.ParsingResult = &parsingResult

		var s []string
		if parsingResult.Flawless() {
			s = []string{"eiffel.elicitation.parse.flawless-success"}
		} else if parsingResult.Ok() {
			s = []string{"eiffel.elicitation.parse.success"}
		}

		if parsingResult.Ok() {
			usr := user.MustCtxUser(ctx)
			csv, created, err := CreateIfNotExists(outputDir, outputFile, 0750)
			if err != nil {
				return io.InlineError(ErrCouldNotCreateFile, err)
			}

			writer := CSVWriter{file: csv}
			if created {
				err := writer.WriteHeaderRow()
				if err != nil {
					return io.InlineError(web.ErrInternal, err)
				}
			}

			err = writer.WriteRow(parsingResult, usr)
			if err != nil {
				return io.InlineError(web.ErrInternal, err)
			}
		}

		formData.CopyAfterParse = CopyAfterParseSetting(request, sessionStore, false)

		return io.Render(web.NewFormData(formData, s, err), "eiffel.elicitation.form", "eiffel/_form-elicitation.go.html")
	})
}

func outputSearchForm(appCtx *hctx.AppCtx, webCtx *web.Ctx, cfg Cfg) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		request := io.Request()
		err := request.ParseForm()
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		rawDir := request.FormValue("output-dir")
		dir := BuildDirPath(cfg.Output.BaseDir, rawDir)
		rawFile := request.FormValue("output-file")
		file := BuildFilename(rawFile)

		path := filepath.Join(dir, file)
		if path == "" {
			return io.InlineError(ErrInvalidFilepath)
		}

		csv, created, err := CreateIfNotExists(dir, file, 0750)
		if err != nil {
			return io.InlineError(ErrCouldNotCreateFile, err)
		}

		if created {
			writer := CSVWriter{file: csv}
			err = writer.WriteHeaderRow()
			if err != nil {
				return io.InlineError(web.ErrInternal, err)
			}
		}

		return io.Render(web.NewFormData(OutputFormData{
			OutputDir:  rawDir,
			OutputFile: rawFile,
		}, []string{"eiffel.elicitation.output.file.success"}), "eiffel.elicitation.output-file.form", "eiffel/_form-output-file.go.html")
	})
}

func outputSearchDir(appCtx *hctx.AppCtx, webCtx *web.Ctx, cfg Cfg) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		request := io.Request()
		err := request.ParseForm()
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		query := request.FormValue("output-dir")
		dirs, err := DirSearch(cfg.Output.BaseDir, query)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		return io.Render(dirs, "eiffel.elicitation.output.dir.search-result", "eiffel/_output-dir-search-result.go.html")
	})
}

func outputFileSearch(appCtx *hctx.AppCtx, webCtx *web.Ctx, cfg Cfg) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		request := io.Request()
		err := request.ParseForm()
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		query := request.FormValue("output-file")
		dir := request.FormValue("output-dir")
		files, err := FileSearch(BuildDirPath(cfg.Output.BaseDir, dir), query)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		return io.Render(files, "eiffel.elicitation.output.file.search-result", "eiffel/_output-file-search-result.go.html")
	})
}
