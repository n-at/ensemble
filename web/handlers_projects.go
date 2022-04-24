package web

import (
	"ensemble/storage/structures"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type projectInfo struct {
	Project *structures.Project
	Update  *structures.ProjectUpdate
}

///////////////////////////////////////////////////////////////////////////////

func (s *Server) projects(c echo.Context) error {
	context := c.(*EnsembleContext)

	var projects []*structures.Project
	var err error

	if context.user.CanViewAllProjects() {
		projects, err = s.store.ProjectGetAll()
	} else {
		projects, err = s.store.ProjectGetByUser(context.user.Id)
	}
	if err != nil {
		log.Warnf("unable to get projects: %s", err)
		return c.Render(http.StatusOK, "templates/error.twig", pongo2.Context{
			"user":  context.user,
			"error": err,
		})
	}

	var projectsWithInfo []*projectInfo
	for _, project := range projects {
		update, err := context.server.store.ProjectUpdateGetProjectLatest(project.Id)
		if err != nil {
			log.Warnf("unable to get latest project update %s: %s", project.Id, err)
			update = nil
		}
		projectsWithInfo = append(projectsWithInfo, &projectInfo{
			Project: project,
			Update:  update,
		})
	}

	return c.Render(http.StatusOK, "templates/projects.twig", pongo2.Context{
		"user":     context.user,
		"projects": projectsWithInfo,
	})
}

func (s *Server) projectNewForm(c echo.Context) error {
	return nil //TODO
}

func (s *Server) projectNewSubmit(c echo.Context) error {
	return nil //TODO
}

func (s *Server) projectEditForm(c echo.Context) error {
	return nil //TODO
}

func (s *Server) projectEditSubmit(c echo.Context) error {
	return nil //TODO
}

func (s *Server) projectDeleteForm(c echo.Context) error {
	return nil //TODO
}

func (s *Server) projectDeleteSubmit(c echo.Context) error {
	return nil //TODO
}

func (s *Server) projectUpdate(c echo.Context) error {
	return nil //TODO
}
