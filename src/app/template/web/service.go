package web

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/google/uuid"
	"github.com/org-harmony/harmony/src/app/eiffel"
	"github.com/org-harmony/harmony/src/app/template"
	"github.com/org-harmony/harmony/src/app/user"
	"github.com/org-harmony/harmony/src/core/event"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/trace"
	"github.com/org-harmony/harmony/src/core/validation"
	"github.com/org-harmony/harmony/src/core/web"
	"os"
)

var (
	// ErrInvalidUUID is returned when the resource's id is not a valid uuid.
	ErrInvalidUUID = errors.New("invalid resource uuid")
	// ErrResourceNotFound is returned when the requested resource (e.g. template) is not found.
	ErrResourceNotFound = errors.New("resource not found")
	// ErrUserNotPermitted is returned when the user is not permitted to access the requested resource, e.g. a template set.
	ErrUserNotPermitted = errors.New("user not permitted")
	// ErrDefaultTemplateDoesNotExist is returned when the default template does not exist.
	ErrDefaultTemplateDoesNotExist = errors.New("default template does not exist")
)

// templateFormData is the data passed to the template form. It contains the template and information about the
// status of the form being an edit form or a new form.
type templateFormData struct {
	// Template is either a *template.ToCreate or a *template.ToUpdate. As Go does not support union types,
	// I don't see another comfortable way to do this.
	Template   any
	IsEditForm bool
}

// templateListPageData is the data for the template list page template.
type templateListPageData struct {
	TemplateSet *template.Set
	Templates   []*template.Template
}

// TemplateSetFromParams returns a template set from the given request parameters. It might return an error if
// the template set id is invalid (ErrInvalidUUID), the template set is not found (ErrResourceNotFound)
// or the user is not permitted to access the template set (ErrUserNotPermitted).
// In the latter case, the template set is still returned and the caller can decide whether to handle the user
// not being permitted to access this template set as an error or not.
func TemplateSetFromParams(io web.IO, repo template.SetRepository, param string) (*template.Set, error) {
	ctx := io.Context()
	u := user.MustCtxUser(ctx)

	templateSetID := web.URLParam(io.Request(), param)
	templateSetUUID, err := uuid.Parse(templateSetID)
	if templateSetID == "" || err != nil {
		return nil, ErrInvalidUUID
	}

	templateSet, err := repo.FindByID(ctx, templateSetUUID)
	if err != nil {
		return nil, errors.Join(ErrResourceNotFound, err)
	}

	if templateSet.CreatedBy != u.ID {
		return templateSet, ErrUserNotPermitted
	}

	return templateSet, nil
}

// TemplateFromParams returns a template from the given request parameters. It might return an error if
// the template id is invalid (ErrInvalidUUID), the template is not found (ErrResourceNotFound)
// or the user is not permitted to access the template (ErrUserNotPermitted).
// In the latter case, the template is still returned and the caller can decide whether to handle the user
// not being permitted to access this template as an error or not.
func TemplateFromParams(io web.IO, repo template.Repository, param string) (*template.Template, error) {
	ctx := io.Context()
	u := user.MustCtxUser(ctx)

	templateID := web.URLParam(io.Request(), param)
	templateUUID, err := uuid.Parse(templateID)
	if templateID == "" || err != nil {
		return nil, ErrInvalidUUID
	}

	tmpl, err := repo.FindByID(ctx, templateUUID)
	if err != nil {
		return nil, errors.Join(ErrResourceNotFound, err)
	}

	if tmpl.CreatedBy != u.ID {
		return tmpl, ErrUserNotPermitted
	}

	return tmpl, nil
}

// CopyTemplate copies the given template into the given template set. It returns the copied template.
// The name of the template is set to the given name, the user id is set as the created by user id of the template.
// Errors are returned transparently.
func CopyTemplate(ctx context.Context, tmpl *template.Template, tmplSetID, usrID uuid.UUID, name string, repo template.Repository) (*template.Template, error) {
	newTmpl, err := repo.CopyInto(ctx, tmpl.ID, tmplSetID, usrID)
	if err != nil {
		return nil, err
	}

	toUpdate := newTmpl.ToUpdate()

	configMap := make(map[string]any)
	err = json.Unmarshal([]byte(tmpl.Config), &configMap)
	if err != nil {
		return nil, err
	}

	configMap["name"] = name

	config, err := json.Marshal(configMap)
	if err != nil {
		return nil, err
	}

	toUpdate.Config = string(config)

	newTmpl, err = repo.Update(ctx, toUpdate)
	if err != nil {
		return nil, err
	}

	return newTmpl, nil
}

func ImportDefaultParisTemplates(ctx context.Context, tmplSetRepo template.SetRepository, tmplRepo template.Repository, usrID uuid.UUID) error {
	defaultAK, err := os.ReadFile("templates/default/paris/ak.json")
	if err != nil {
		return ErrDefaultTemplateDoesNotExist
	}
	defaultESFA, err := os.ReadFile("templates/default/paris/esfa.json")
	if err != nil {
		return ErrDefaultTemplateDoesNotExist
	}
	defaultESQUA, err := os.ReadFile("templates/default/paris/esqua.json")
	if err != nil {
		return ErrDefaultTemplateDoesNotExist
	}
	defaultGlossar, err := os.ReadFile("templates/default/paris/glossar.json")
	if err != nil {
		return ErrDefaultTemplateDoesNotExist
	}

	tmplSet, err := tmplSetRepo.Create(ctx, &template.SetToCreate{
		Name:        "PARIS",
		Version:     "0.6.2",
		CreatedBy:   usrID,
		Description: "Default PARIS templates. Change description and templates as needed.",
	})
	if err != nil {
		return web.ErrInternal
	}

	_, err = tmplRepo.Create(ctx, &template.ToCreate{
		TemplateSet: tmplSet.ID,
		Type:        eiffel.BasicTemplateType,
		Config:      string(defaultAK),
		CreatedBy:   usrID,
	})

	_, err = tmplRepo.Create(ctx, &template.ToCreate{
		TemplateSet: tmplSet.ID,
		Type:        eiffel.BasicTemplateType,
		Config:      string(defaultESFA),
		CreatedBy:   usrID,
	})

	_, err = tmplRepo.Create(ctx, &template.ToCreate{
		TemplateSet: tmplSet.ID,
		Type:        eiffel.BasicTemplateType,
		Config:      string(defaultESQUA),
		CreatedBy:   usrID,
	})

	_, err = tmplRepo.Create(ctx, &template.ToCreate{
		TemplateSet: tmplSet.ID,
		Type:        eiffel.BasicTemplateType,
		Config:      string(defaultGlossar),
		CreatedBy:   usrID,
	})

	return nil
}

// templateSetInlineDelete reads the template set id from the request 'id' parameter and deletes the template set.
// Errors are reported to the IO as inline errors. Errors are returned as internal errors safe to show to the user.
func templateSetInlineDelete(io web.IO, repo template.SetRepository) error {
	templateSet, err := TemplateSetFromParams(io, repo, "id")
	if err != nil {
		return io.InlineError(web.ErrInternal, err)
	}

	err = repo.Delete(io.Context(), templateSet.ID)
	if err != nil {
		return io.InlineError(web.ErrInternal, err)
	}

	return nil
}

// templateSetsForList reads the current user from the context and returns all template sets created by the user.
// It reports errors to the IO as inline errors. Errors are returned as internal errors safe to show to the user.
func templateSetsForList(io web.IO, repo template.SetRepository) ([]*template.Set, error) {
	usr := user.MustCtxUser(io.Context())

	templateSets, err := repo.FindByCreatedBy(io.Context(), usr.ID)
	if err != nil && !errors.Is(err, persistence.ErrNotFound) {
		return nil, io.InlineError(web.ErrInternal, err)
	}

	return templateSets, nil
}

// readValidTemplateForm reads the template form from the request and validates it. It returns the template to create
// and a slice of validation errors. If the validation errors slice is not empty, the template to create is not valid.
// Errors are returned as internal errors they are not safe to show to the user.
func readValidTemplateForm(
	io web.IO,
	templateSet *template.Set,
	validator validation.V,
	em event.Manager,
	logger trace.Logger,
) (*template.ToCreate, []error, error) {
	usr := user.MustCtxUser(io.Context())

	request := io.Request()
	err := request.ParseForm()
	if err != nil {
		return nil, nil, err
	}

	cfg := request.FormValue("Config")
	toCreate, err := template.ToCreateFromConfig(cfg)
	if err != nil {
		empty := &template.ToCreate{Config: cfg, TemplateSet: templateSet.ID}
		return empty, []error{validation.Error{Msg: "template.new.invalid-json"}}, nil
	}

	toCreate.TemplateSet = templateSet.ID
	toCreate.CreatedBy = usr.ID

	validationErrs, err := template.ValidateTemplateToCreate(toCreate, validator, em, logger)

	return toCreate, validationErrs, err
}

// readValidTemplateUpdateForm reads the template form from the request and validates it. It returns the template to update
// and a slice of validation errors. If the validation errors slice is not empty, the template to update is not valid.
// Errors are returned as internal errors they are not safe to show to the user.
func readValidTemplateUpdateForm(
	io web.IO,
	tmpl *template.Template,
	validator validation.V,
	em event.Manager,
	logger trace.Logger,
) (*template.ToUpdate, []error, error) {
	request := io.Request()
	err := request.ParseForm()
	if err != nil {
		return nil, nil, err
	}

	toUpdate := tmpl.ToUpdate()
	cfg := request.FormValue("Config")
	toCreate, err := template.ToCreateFromConfig(cfg)
	if err != nil {
		toUpdate.Config = cfg
		return toUpdate, []error{validation.Error{Msg: "template.new.invalid-json"}}, nil
	}

	toUpdate.Config = toCreate.Config
	toUpdate.Type = toCreate.Type

	validationErrs, err := template.ValidateTemplateToUpdate(toUpdate, validator, em, logger)

	return toUpdate, validationErrs, err
}

// renderNewTemplatePage renders the template set new page template.
func renderNewTemplatePage(io web.IO, toCreate *template.ToCreate, validationErrs []error) error {
	return io.Render(
		web.NewFormData(&templateFormData{Template: toCreate, IsEditForm: false}, nil, validationErrs...),
		"template.form.page",
		"template/form-page.go.html",
		"template/_form.go.html",
	)
}

// renderEditTemplatePage renders the template set edit page template.
func renderEditTemplatePage(io web.IO, toUpdate *template.ToUpdate, success []string, validationErrs []error) error {
	return io.Render(
		web.NewFormData(&templateFormData{Template: toUpdate, IsEditForm: true}, success, validationErrs...),
		"template.form.page",
		"template/form-page.go.html",
		"template/_form.go.html",
	)
}

// renderEditTemplateForm renders the template set edit form template.
func renderEditTemplateForm(io web.IO, toUpdate *template.ToUpdate, success []string, validationErrs []error) error {
	return io.Render(
		web.NewFormData(&templateFormData{Template: toUpdate, IsEditForm: true}, success, validationErrs...),
		"template.form",
		"template/_form.go.html",
	)
}

// renderNewTemplateSetPage renders the template set new page template.
func renderNewTemplateSetPage(io web.IO, toCreate *template.SetToCreate, validationErrs []error) error {
	return io.Render(
		web.NewFormData(toCreate, nil, validationErrs...),
		"template.set.new.page",
		"template/set-new-page.go.html",
		"template/_form-set-new.go.html",
	)
}

// renderEditTemplateSetForm renders the template set edit form template.
func renderEditTemplateSetForm(io web.IO, templateSet *template.SetToUpdate, success []string, validationErrs []error) error {
	return io.Render(
		web.NewFormData(templateSet, success, validationErrs...),
		"template.set.edit.form",
		"template/_form-set-edit.go.html",
	)
}

// TODO do for other things that happen in the controllers
// TODO for stuff that is done in controllers but not specific to web layer, move to the service layer above (not template/web but template)
