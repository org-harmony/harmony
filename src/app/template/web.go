package template

import (
	"errors"
	"fmt"
	"github.com/google/uuid"
	"github.com/org-harmony/harmony/src/app/user"
	"github.com/org-harmony/harmony/src/core/hctx"
	"github.com/org-harmony/harmony/src/core/persistence"
	"github.com/org-harmony/harmony/src/core/util"
	"github.com/org-harmony/harmony/src/core/web"
	"net/http"
)

var (
	ErrInvalidTemplateSetID = errors.New("invalid template set id")
	ErrTemplateSetNotFound  = errors.New("template set not found")
	ErrUserNotPermitted     = errors.New("user not permitted")
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
}

// SetFromParams returns a template set from the given request parameters. It might return an error if
// the template set id is invalid (ErrInvalidTemplateSetID), the template set is not found (ErrTemplateSetNotFound)
// or the user is not permitted to access the template set (ErrUserNotPermitted).
// In the latter case, the template set is still returned and the caller can decide whether to handle the user
// not being permitted to access this template set as an error or not.
func SetFromParams(io web.IO, repo SetRepository, param string) (*Set, error) {
	ctx := io.Context()
	u := user.MustCtxUser(ctx)

	templateSetID := web.URLParam(io.Request(), param)
	templateSetUUID, err := uuid.Parse(templateSetID)
	if templateSetID == "" || err != nil {
		return nil, ErrInvalidTemplateSetID
	}

	templateSet, err := repo.FindByID(ctx, templateSetUUID)
	if err != nil {
		return nil, errors.Join(ErrTemplateSetNotFound, err)
	}

	if templateSet.CreatedBy != u.ID {
		return templateSet, ErrUserNotPermitted
	}

	return templateSet, nil
}

func registerNavigation(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	webCtx.Navigation.Add("web", web.NavItem{
		URL:      "/template-set/list",
		Name:     "harmony.menu.template-sets",
		Position: 100,
	})
}

func templateSetListController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[SetRepository](appCtx.Repository(SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()
		templateSets, err := templateSetRepository.FindByCreatedBy(ctx, user.MustCtxUser(ctx).ID)
		if err != nil && !errors.Is(err, persistence.ErrNotFound) {
			return io.Error(web.ErrInternal, err)
		}

		return io.RenderJoined(templateSets, "template.set.list.page", "template/set-list-page.go.html", "template/_list-set.go.html")
	})
}

func templateSetNewController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		return io.RenderJoined(
			web.NewFormData(&SetToCreate{}, nil),
			"template.set.new.page",
			"template/set-new-page.go.html",
			"template/_form-set-new.go.html",
		)
	})
}

func templateSetNewSaveController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[SetRepository](appCtx.Repository(SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()

		toCreate := &SetToCreate{CreatedBy: user.MustCtxUser(ctx).ID}
		err, validationErrs := web.ReadForm(io.Request(), toCreate, appCtx.Validator)
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		if validationErrs != nil {
			return io.RenderJoined(
				web.NewFormData(toCreate, nil, validationErrs...),
				"template.set.new.page",
				"template/set-new-page.go.html",
				"template/_form-set-new.go.html",
			)
		}

		_, err = templateSetRepository.Create(ctx, toCreate)
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		return io.Redirect("/template-set/list", http.StatusFound)
	})
}

func templateSetEditFormController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[SetRepository](appCtx.Repository(SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		templateSet, err := SetFromParams(io, templateSetRepository, "id")
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		return io.RenderJoined(
			web.NewFormData(templateSet.ToUpdate(), nil),
			"template.set.edit.form",
			"template/_form-set-edit.go.html",
		)
	})
}

func templateSetEditController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[SetRepository](appCtx.Repository(SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()

		templateSet, err := SetFromParams(io, templateSetRepository, "id")
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		toUpdate := templateSet.ToUpdate()
		err, validationErrs := web.ReadForm(io.Request(), toUpdate, appCtx.Validator)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		if validationErrs != nil {
			return io.Render("template.set.edit.form", "template/_form-set-edit.go.html", web.NewFormData(toUpdate, nil, validationErrs...))
		}

		templateSet, err = templateSetRepository.Update(ctx, toUpdate)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		return io.Render("template.set.edit.form", "template/_form-set-edit.go.html", web.NewFormData(templateSet.ToUpdate(), []string{"template.set.edit.updated"}))
	})
}

func templateSetDeleteController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[SetRepository](appCtx.Repository(SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()
		u := user.MustCtxUser(ctx)

		templateSet, err := SetFromParams(io, templateSetRepository, "id")
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		err = templateSetRepository.Delete(ctx, templateSet.ID)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		templateSets, err := templateSetRepository.FindByCreatedBy(ctx, u.ID)
		if err != nil && !errors.Is(err, persistence.ErrNotFound) {
			return io.InlineError(web.ErrInternal, err)
		}

		return io.Render("template.set.list", "template/_list-set.go.html", templateSets)
	})
}

func templateListController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[SetRepository](appCtx.Repository(SetRepositoryName))
	templateRepository := util.UnwrapType[Repository](appCtx.Repository(RepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()

		templateSet, err := SetFromParams(io, templateSetRepository, "id")
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		templates, err := templateRepository.FindByTemplateSetID(ctx, templateSet.ID)
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		return io.RenderJoined(struct {
			TemplateSet *Set
			Templates   []*Template
		}{
			TemplateSet: templateSet,
			Templates:   templates,
		}, "template.list.page", "template/list-page.go.html", "template/_list.go.html")
	})
}

func templateNewController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[SetRepository](appCtx.Repository(SetRepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		templateSet, err := SetFromParams(io, templateSetRepository, "id")
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		return io.RenderJoined(
			web.NewFormData(struct {
				TemplateSet *Set
				Template    *ToCreate
			}{
				TemplateSet: templateSet,
				Template:    &ToCreate{TemplateSet: templateSet.ID},
			}, nil),
			"template.new.page",
			"template/new-page.go.html",
			"template/_form-new.go.html",
		)
	})
}

func templateNewSaveController(appCtx *hctx.AppCtx, webCtx *web.Ctx) http.Handler {
	templateSetRepository := util.UnwrapType[SetRepository](appCtx.Repository(SetRepositoryName))
	templateRepository := util.UnwrapType[Repository](appCtx.Repository(RepositoryName))

	return web.NewController(appCtx, webCtx, func(io web.IO) error {
		ctx := io.Context()
		u := user.MustCtxUser(ctx)

		templateSet, err := SetFromParams(io, templateSetRepository, "id")
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		// todo add validation and transformation based on template type => e.g. EBT (EIFFEL Basic Template)

		toCreate := &ToCreate{TemplateSet: templateSet.ID, CreatedBy: u.ID}
		err, validationErrs := web.ReadForm(io.Request(), toCreate, appCtx.Validator)
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		if validationErrs != nil {
			return io.RenderJoined(
				web.NewFormData(struct {
					TemplateSet *Set
					Template    *ToCreate
				}{
					TemplateSet: templateSet,
					Template:    toCreate,
				}, nil, validationErrs...),
				"template.new.page",
				"template/new-page.go.html",
				"template/_form-new.go.html",
			)
		}

		_, err = templateRepository.Create(ctx, toCreate)
		if err != nil {
			return io.Error(web.ErrInternal, err)
		}

		return io.Redirect(fmt.Sprintf("/template-set/%s/list", templateSet.ID), http.StatusFound)
	})
}
