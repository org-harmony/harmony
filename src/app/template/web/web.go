package web

import (
	"errors"
	"fmt"
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

func RegisterController(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	registerNavigation(appCtx, webCtx)

	router := webCtx.Router.With(user.LoggedInMiddleware(appCtx))

	router.Get("/template-set/list", templateSetListController(appCtx, webCtx).ServeHTTP)
	router.Get("/template-set/new", templateSetNewController(appCtx, webCtx).ServeHTTP)
	router.Post("/template-set/new", templateSetNewSaveController(appCtx, webCtx).ServeHTTP)
	router.Get("/template-set/edit/{id}", templateSetEditFormController(appCtx, webCtx).ServeHTTP)
	router.Put("/template-set/{id}", templateSetEditController(appCtx, webCtx).ServeHTTP)
	router.Delete("/template-set/{id}", templateSetDeleteController(appCtx, webCtx).ServeHTTP)

	router.Get("/template-set/{id}/list", templateListController(appCtx, webCtx).ServeHTTP)
	router.Get("/template-set/{id}/new", templateNewController(appCtx, webCtx).ServeHTTP)
	router.Post("/template-set/{id}/new", templateNewSaveController(appCtx, webCtx).ServeHTTP)

	router.Get("/template/{id}/edit", templateEditPageController(appCtx, webCtx).ServeHTTP)
	router.Put("/template/{id}", templateEditSaveController(appCtx, webCtx).ServeHTTP)
	router.Delete("/template/{id}", templateDeleteController(appCtx, webCtx).ServeHTTP)
}

func registerNavigation(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	webCtx.Navigation.Add("web", web.NavItem{
		URL:      "/template-set/list",
		Name:     "harmony.menu.template-sets",
		Position: 100,
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

		return io.Render(
			templateSets,
			"template.set.list.page",
			"template/set-list-page.go.html",
			"template/_list-set.go.html",
		)
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

		return io.Render(templateSets, "template.set.list", "template/_list-set.go.html")
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
			return io.Error(web.ErrInternal, err)
		}

		err = templateRepository.Delete(io.Context(), tmpl.ID)
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		templateSet, err := templateSetRepository.FindByID(io.Context(), tmpl.TemplateSet)
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		templates, err := templateRepository.FindByTemplateSetID(io.Context(), templateSet.ID)
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		return io.Render(templateListPageData{
			TemplateSet: templateSet,
			Templates:   templates,
		}, "template.list", "template/_list.go.html")
	})
}
