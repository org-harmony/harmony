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

func RegisterController(appCtx *hctx.AppCtx, webCtx *web.Ctx) {
	registerNavigation(appCtx, webCtx)

	router := webCtx.Router.With(user.LoggedInMiddleware(appCtx))

	router.Get("/template-set/list", templateSetListController(appCtx, webCtx).ServeHTTP)
	router.Get("/template-set/new", templateSetNewController(appCtx, webCtx).ServeHTTP)
	router.Post("/template-set/new", templateSetNewSaveController(appCtx, webCtx).ServeHTTP)
	router.Get("/template-set/edit/{id}", templateSetEditFormController(appCtx, webCtx).ServeHTTP)
	router.Put("/template-set/{id}", templateSetEditController(appCtx, webCtx).ServeHTTP)
	router.Delete("/template-set/{id}", templateSetDeleteController(appCtx, webCtx).ServeHTTP)
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
		ctx := io.Context()
		u := user.MustCtxUser(ctx)

		templateSetID := web.URLParam(io.Request(), "id")
		templateSetUUID, err := uuid.Parse(templateSetID)
		if templateSetID == "" || err != nil {
			return io.InlineError(web.ErrInternal, fmt.Errorf("template set id %s invalid (during edit page)", templateSetID), err)
		}

		templateSet, err := templateSetRepository.FindByID(ctx, templateSetUUID)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		if templateSet.CreatedBy != u.ID {
			appCtx.Info("user %s tried to edit template set %s without permission", u.ID.String(), templateSetID)
			return io.InlineError(web.ErrInternal, fmt.Errorf("template set %s not found", templateSetID))
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
		u := user.MustCtxUser(ctx)

		templateSetID := web.URLParam(io.Request(), "id")
		templateSetUUID, err := uuid.Parse(templateSetID)
		if templateSetID == "" || err != nil {
			return io.InlineError(web.ErrInternal, fmt.Errorf("template set id %s invalid (during edit)", templateSetID), err)
		}

		templateSet, err := templateSetRepository.FindByID(ctx, templateSetUUID)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		if templateSet.CreatedBy != u.ID {
			appCtx.Info("user %s tried to edit template set %s without permission", u.ID.String(), templateSetID)
			return io.InlineError(web.ErrInternal, fmt.Errorf("template set %s not found", templateSetID))
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

		templateSetID := web.URLParam(io.Request(), "id")
		templateSetUUID, err := uuid.Parse(templateSetID)
		if templateSetID == "" || err != nil {
			return io.InlineError(web.ErrInternal, fmt.Errorf("template set id %s invalid (during deletion)", templateSetID), err)
		}

		templateSet, err := templateSetRepository.FindByID(ctx, templateSetUUID)
		if err != nil {
			return io.InlineError(web.ErrInternal, err)
		}

		if templateSet.CreatedBy != u.ID {
			appCtx.Info("user %s tried to delete template set %s without permission", u.ID.String(), templateSetID)
			return io.InlineError(web.ErrInternal, fmt.Errorf("template set %s not found", templateSetID))
		}

		err = templateSetRepository.Delete(ctx, templateSetUUID)
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
