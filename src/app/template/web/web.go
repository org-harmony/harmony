package web

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/org-harmony/harmony/src/app/template"
	"github.com/org-harmony/harmony/src/app/user"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/validation"
	"github.com/org-harmony/harmony/src/core/web"
	"net/http"
)

var (
	// ErrTemplateConfigIncomplete is a validation error that is displayed to the user when the template config is incomplete.
	ErrTemplateConfigIncomplete = validation.Error{Msg: "template.new.config-incomplete"}
)

// TemplateCopyFormData is passed to the template copy modal to render the form that allows users to copy a template into another template set.
type TemplateCopyFormData struct {
	Name          string `hvalidate:"required"`
	TemplateSetID string `hvalidate:"required"`
	Template      *template.Template
	TemplateSets  []*template.Set
	Copied        bool
}

// TemplateSetListData is passed to the template set list and contains the additional paris version.
type TemplateSetListData struct {
	TemplateSets []*template.Set
	PARISVersion string
}

// RegisterController registers the controllers and navigation for the template module.
func RegisterController(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	registerNavigation(appCtx, webCtx)

	router := webCtx.Router.With(user.LoggedInMiddleware(appCtx))

	router.Get("/template-set/list", templateSetListController(appCtx, webCtx).ServeHTTP)
	router.Get("/template-set/new", templateSetNewController(appCtx, webCtx).ServeHTTP)
	router.Post("/template-set/new", templateSetNewSaveController(appCtx, webCtx).ServeHTTP)
	router.Get("/template-set/edit/{id}", templateSetEditFormController(appCtx, webCtx).ServeHTTP)
	router.Put("/template-set/{id}", templateSetEditController(appCtx, webCtx).ServeHTTP)
	router.Delete("/template-set/{id}", templateSetDeleteController(appCtx, webCtx).ServeHTTP)
	// TODO generalize this
	router.Post("/template-set/import/default-paris", templateSetImportDefaultParisController(appCtx, webCtx).ServeHTTP)

	router.Get("/template-set/{id}/list", templateListController(appCtx, webCtx).ServeHTTP)
	router.Get("/template-set/{id}/new", templateNewController(appCtx, webCtx).ServeHTTP)
	router.Post("/template-set/{id}/new", templateNewSaveController(appCtx, webCtx).ServeHTTP)
	router.Get("/template/{id}/edit", templateEditPageController(appCtx, webCtx).ServeHTTP)
	router.Put("/template/{id}", templateEditSaveController(appCtx, webCtx).ServeHTTP)
	router.Delete("/template/{id}", templateDeleteController(appCtx, webCtx).ServeHTTP)
	router.Get("/template/{id}/copy/modal", templateCopyModalController(appCtx, webCtx).ServeHTTP)
	router.Post("/template/{id}/copy", templateCopyController(appCtx, webCtx).ServeHTTP)
}

func registerNavigation(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	webCtx.Navigation.Add("template.set.list", web.NavItem{
		URL:      "/template-set/list",
		Name:     "harmony.menu.template-sets",
		Position: 150,
	})
}

func templateSetListController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[template.SetRepository](appCtx.Repository(template.SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()
		templateSets, err := templateSetRepository.FindByCreatedBy(ctx, user.MustCtxUser(ctx).ID)
		if err != nil && !errors.Is(err, persistence.ErrNotFound) {
			return io.Error(web.ErrInternal, err)
		}

		ver, err := LatestPARISVersion("docs/templates/paris")
		if err != nil {
			return io.Error(ErrDefaultTemplateDoesNotExist, err)
		}

		return io.Render(TemplateSetListData{
			TemplateSets: templateSets,
			PARISVersion: ver,
		}, "template.set.list.page", "template/set-list-page.go.html", "template/_list-set.go.html")
	})
}

func templateSetNewController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		return renderNewTemplateSetPage(io, &template.SetToCreate{}, nil)
	})
}

func templateSetNewSaveController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[template.SetRepository](appCtx.Repository(template.SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()

		toCreate := &template.SetToCreate{CreatedBy: user.MustCtxUser(ctx).ID}
		err, validationErrs := web.ReadForm(io.Request(), toCreate, appCtx.Validator)
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		if validationErrs != nil {
			return renderNewTemplateSetPage(io, toCreate, validationErrs)
		}

		_, err = templateSetRepository.Create(ctx, toCreate)
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		return io.Redirect("/template-set/list", http.StatusFound)
	})
}

func templateSetEditFormController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[template.SetRepository](appCtx.Repository(template.SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		templateSet, err := TemplateSetFromParams(io, templateSetRepository, "id")
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		return renderEditTemplateSetForm(io, templateSet.ToUpdate(), nil, nil)
	})
}

func templateSetEditController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[template.SetRepository](appCtx.Repository(template.SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()

		templateSet, err := TemplateSetFromParams(io, templateSetRepository, "id")
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		toUpdate := templateSet.ToUpdate()
		err, validationErrs := web.ReadForm(io.Request(), toUpdate, appCtx.Validator)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		if validationErrs != nil {
			return renderEditTemplateSetForm(io, toUpdate, nil, validationErrs)
		}

		templateSet, err = templateSetRepository.Update(ctx, toUpdate)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		return renderEditTemplateSetForm(io, templateSet.ToUpdate(), []string{"template.set.edit.updated"}, nil)
	})
}

func templateSetDeleteController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[template.SetRepository](appCtx.Repository(template.SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		err := templateSetInlineDelete(io, templateSetRepository)
		if err != nil {
			return err
		}

		templateSets, err := templateSetsForList(io, templateSetRepository)
		if err != nil {
			return err
		}

		ver, err := LatestPARISVersion("docs/templates/paris")
		if err != nil {
			return io.InlineError(ErrDefaultTemplateDoesNotExist, err)
		}

		return io.Render(TemplateSetListData{
			TemplateSets: templateSets,
			PARISVersion: ver,
		}, "template.set.list", "template/_list-set.go.html")
	})
}

func templateListController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[template.SetRepository](appCtx.Repository(template.SetRepositoryName))
	templateRepository := util.UnwrapType[template.Repository](appCtx.Repository(template.RepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()

		templateSet, err := TemplateSetFromParams(io, templateSetRepository, "id")
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		templates, err := templateRepository.FindByTemplateSetID(ctx, templateSet.ID)
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		return io.Render(templateListPageData{
			TemplateSet: templateSet,
			Templates:   templates,
		}, "template.list.page", "template/list-page.go.html", "template/_list.go.html")
	})
}

func templateNewController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[template.SetRepository](appCtx.Repository(template.SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		templateSet, err := TemplateSetFromParams(io, templateSetRepository, "id")
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		return renderNewTemplatePage(io, &template.ToCreate{TemplateSet: templateSet.ID}, nil)
	})
}

func templateNewSaveController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[template.SetRepository](appCtx.Repository(template.SetRepositoryName))
	templateRepository := util.UnwrapType[template.Repository](appCtx.Repository(template.RepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()

		templateSet, err := TemplateSetFromParams(io, templateSetRepository, "id")
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		toCreate, validationErrs, err := readValidTemplateForm(io, templateSet, appCtx.Validator, appCtx.EventManager, appCtx.Logger)
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		if validationErrs != nil {
			return renderNewTemplatePage(io, toCreate, validationErrs)
		}

		_, err = templateRepository.Create(ctx, toCreate)
		if err != nil && errors.Is(err, template.ErrTemplateConfigMissingInfo) {
			return renderNewTemplatePage(io, toCreate, []error{ErrTemplateConfigIncomplete})
		} else if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		return io.Redirect(fmt.Sprintf("/template-set/%s/list", templateSet.ID), http.StatusFound)
	})
}

func templateEditPageController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateRepository := util.UnwrapType[template.Repository](appCtx.Repository(template.RepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		tmpl, err := TemplateFromParams(io, templateRepository, "id")
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		return renderEditTemplatePage(io, tmpl.ToUpdate(), nil, nil)
	})
}

func templateEditSaveController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateRepository := util.UnwrapType[template.Repository](appCtx.Repository(template.RepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()

		tmpl, err := TemplateFromParams(io, templateRepository, "id")
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		toUpdate, validationErrs, err := readValidTemplateUpdateForm(io, tmpl, appCtx.Validator, appCtx.EventManager, appCtx.Logger)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		if validationErrs != nil {
			return renderEditTemplateForm(io, toUpdate, nil, validationErrs)
		}

		tmpl, err = templateRepository.Update(ctx, toUpdate)
		if err != nil && errors.Is(err, template.ErrTemplateConfigMissingInfo) {
			return renderEditTemplateForm(io, toUpdate, nil, []error{ErrTemplateConfigIncomplete})
		} else if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		return renderEditTemplateForm(io, tmpl.ToUpdate(), []string{"template.edit.updated"}, nil)
	})
}

func templateDeleteController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateRepository := util.UnwrapType[template.Repository](appCtx.Repository(template.RepositoryName))
	templateSetRepository := util.UnwrapType[template.SetRepository](appCtx.Repository(template.SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		tmpl, err := TemplateFromParams(io, templateRepository, "id")
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		err = templateRepository.Delete(io.Context(), tmpl.ID)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		templateSet, err := templateSetRepository.FindByID(io.Context(), tmpl.TemplateSet)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		templates, err := templateRepository.FindByTemplateSetID(io.Context(), templateSet.ID)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		return io.Render(templateListPageData{
			TemplateSet: templateSet,
			Templates:   templates,
		}, "template.list", "template/_list.go.html")
	})
}

func templateCopyModalController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateRepository := util.UnwrapType[template.Repository](appCtx.Repository(template.RepositoryName))
	templateSetRepository := util.UnwrapType[template.SetRepository](appCtx.Repository(template.SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		tmpl, err := TemplateFromParams(io, templateRepository, "id")
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		tmplSets, err := templateSetRepository.FindByCreatedBy(io.Context(), user.MustCtxUser(io.Context()).ID)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		return io.Render(web.NewFormData(TemplateCopyFormData{
			Template:     tmpl,
			TemplateSets: tmplSets,
		}, nil), "template.copy.modal", "template/_modal-copy.go.html")
	})
}

func templateCopyController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateRepository := util.UnwrapType[template.Repository](appCtx.Repository(template.RepositoryName))
	templateSetRepository := util.UnwrapType[template.SetRepository](appCtx.Repository(template.SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()
		usr := user.MustCtxUser(ctx)

		tmpl, err := TemplateFromParams(io, templateRepository, "id")
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		tmplSets, err := templateSetRepository.FindByCreatedBy(ctx, usr.ID)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		formData := &TemplateCopyFormData{Template: tmpl, TemplateSets: tmplSets}
		err, validationErrs := web.ReadForm(io.Request(), formData, appCtx.Validator)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		tmplSetUUID, err := uuid.Parse(formData.TemplateSetID)
		if err != nil {
			validationErrs = append(validationErrs, validation.Error{Msg: "template.copy.invalid-template-set-id"})
		}

		var intoTmplSet *template.Set
		if err == nil {
			intoTmplSet, err = templateSetRepository.FindByID(ctx, tmplSetUUID)
			if err != nil && !errors.Is(err, persistence.ErrNotFound) {
				return io.InlineError(web.ErrInternal, err)
			}

			if err != nil || intoTmplSet.CreatedBy != usr.ID {
				validationErrs = append(validationErrs, validation.Error{Msg: "template.copy.template-set-not-found"})
			}
		}

		if validationErrs != nil {
			return io.Render(web.NewFormData(formData, nil, validationErrs...), "template.copy.modal", "template/_modal-copy.go.html")
		}

		_, err = CopyTemplate(ctx, tmpl, tmplSetUUID, usr.ID, formData.Name, templateRepository)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		formData.Copied = true

		return io.Render(web.NewFormData(formData, []string{"template.copy.success"}), "template.copy.modal", "template/_modal-copy.go.html")
	})
}

func templateSetImportDefaultParisController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[template.SetRepository](appCtx.Repository(template.SetRepositoryName))
	templateRepository := util.UnwrapType[template.Repository](appCtx.Repository(template.RepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()

		err := ImportDefaultPARISTemplates(ctx, "docs/templates/paris", templateSetRepository, templateRepository, user.MustCtxUser(ctx).ID)
		if err != nil {
			return io.InlineError(err)
		}

		templateSets, err := templateSetRepository.FindByCreatedBy(ctx, user.MustCtxUser(ctx).ID)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		ver, err := LatestPARISVersion("docs/templates/paris")
		if err != nil {
			return io.InlineError(ErrDefaultTemplateDoesNotExist, err)
		}

		return io.Render(TemplateSetListData{
			TemplateSets: templateSets,
			PARISVersion: ver,
		}, "template.set.list", "template/_list-set.go.html")
	})
}
