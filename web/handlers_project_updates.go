package web

import (
	"fmt"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (s *Server) projectUpdates(c echo.Context) error {
	context := c.(*EnsembleContext)

	updates, err := s.store.ProjectUpdateGetByProject(context.project.Id)
	if err != nil {
		log.Infof("projectUpdates project %s updates get error: %s", context.project.Id, err)
		return err
	}

	return c.Render(http.StatusOK, "templates/project_updates.twig", pongo2.Context{
		"_csrf_token": c.Get("csrf"),
		"user":        context.user,
		"project":     context.project,
		"updates":     updates,
	})
}

func (s *Server) projectUpdateLog(c echo.Context) error {
	context := c.(*EnsembleContext)

	return c.Render(http.StatusOK, "templates/project_update_log.twig", pongo2.Context{
		"_csrf_token": c.Get("csrf"),
		"user":        context.user,
		"project":     context.project,
		"update":      context.projectUpdate,
	})
}

func (s *Server) projectUpdateDeleteForm(c echo.Context) error {
	context := c.(*EnsembleContext)

	return c.Render(http.StatusOK, "templates/project_update_delete.twig", pongo2.Context{
		"_csrf_token": c.Get("csrf"),
		"user":        context.user,
		"project":     context.project,
		"update":      context.projectUpdate,
	})
}

func (s *Server) projectUpdateDeleteSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)

	log.Infof("projectUpdateDeleteSubmit %s", context.projectUpdate.Id)

	err := s.store.ProjectUpdateDelete(context.projectUpdate.Id)
	if err != nil {
		log.Errorf("projectUpdateDeleteSubmit project update %s delete error: %s", context.projectUpdate.Id, err)
		return err
	}

	returnUrl := fmt.Sprintf("/projects/updates/%s", context.project.Id)

	return c.Redirect(http.StatusFound, returnUrl)
}
