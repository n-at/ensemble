package web

import (
	"fmt"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	"net/http"
)

func (s *Server) projectUpdates(c echo.Context) error {
	context := c.(*EnsembleContext)

	updates, err := s.store.ProjectUpdateGetByProject(context.project.Id)
	if err != nil {
		return err
	}

	return c.Render(http.StatusOK, "templates/project_updates.twig", pongo2.Context{
		"user":    context.user,
		"project": context.project,
		"updates": updates,
	})
}

func (s *Server) projectUpdateLog(c echo.Context) error {
	context := c.(*EnsembleContext)

	return c.Render(http.StatusOK, "templates/project_update_log.twig", pongo2.Context{
		"user":    context.user,
		"project": context.project,
		"update":  context.projectUpdate,
	})
}

func (s *Server) projectUpdateDeleteForm(c echo.Context) error {
	context := c.(*EnsembleContext)

	return c.Render(http.StatusOK, "templates/project_update_delete.twig", pongo2.Context{
		"user":    context.user,
		"project": context.project,
		"update":  context.projectUpdate,
	})
}

func (s *Server) projectUpdateDeleteSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)

	err := s.store.ProjectUpdateDelete(context.projectUpdate.Id)
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, fmt.Sprintf("/projects/updates/%s", context.project.Id))
}
