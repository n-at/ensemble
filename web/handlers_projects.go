package web

import (
	"ensemble/storage/structures"
	"errors"
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

//projects List of current user projects
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
		return err
	}

	var projectsWithInfo []*projectInfo
	for _, project := range projects {
		update, err := s.store.ProjectUpdateGetProjectLatest(project.Id)
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

//projectNewForm Form for new project creation
func (s *Server) projectNewForm(c echo.Context) error {
	context := c.(*EnsembleContext)

	if !context.user.CanCreateProjects() {
		return errors.New("forbidden to create new projects")
	}

	return c.Render(http.StatusOK, "templates/project_new.twig", pongo2.Context{
		"user": context.user,
	})
}

//projectNewSubmit Save and update new project
func (s *Server) projectNewSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)

	if !context.user.CanCreateProjects() {
		return errors.New("forbidden to create new projects")
	}

	project := &structures.Project{
		Name:               c.FormValue("name"),
		Description:        c.FormValue("description"),
		RepositoryUrl:      c.FormValue("repo_url"),
		RepositoryLogin:    c.FormValue("repo_login"),
		RepositoryPassword: c.FormValue("repo_password"),
		RepositoryBranch:   c.FormValue("repo_branch"),
		Inventory:          c.FormValue("inventory"),
		Variables:          c.FormValue("variables"),
		VaultPassword:      c.FormValue("vault_password"),
	}

	err := s.store.ProjectInsert(project)
	if err != nil {
		log.Warnf("projectNewSubmit unable to save project: %s", err)
		return c.Render(http.StatusOK, "templates/project_new.twig", pongo2.Context{
			"user":    context.user,
			"project": project,
			"error":   err,
		})
	}

	err = s.manager.Update(project)
	if err != nil {
		log.Warnf("projectNewSubmit unable to update project: %s", err)
	}

	return c.Redirect(http.StatusFound, "/projects/")
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

//projectUpdate Update project repository
func (s *Server) projectUpdate(c echo.Context) error {
	project, err := s.getProjectToRead(c)
	if err != nil {
		return err
	}

	err = s.manager.Update(project)
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, "/projects/")
}

///////////////////////////////////////////////////////////////////////////////

func (s *Server) getProjectToRead(c echo.Context) (*structures.Project, error) {
	context := c.(*EnsembleContext)

	projectId := c.Param("project_id")
	project, err := s.store.ProjectGet(projectId)
	if err != nil {
		return nil, err
	}
	if context.user.CanEditProjects() {
		return project, nil
	}

	if s.store.ProjectUserAccessExists(projectId, context.user.Id) {
		return project, nil
	} else {
		return nil, errors.New("forbidden to view project")
	}
}

func (s *Server) getProjectToWrite(c echo.Context) (*structures.Project, error) {
	context := c.(*EnsembleContext)

	projectId := c.Param("project_id")
	project, err := s.store.ProjectGet(projectId)
	if err != nil {
		return nil, err
	}

	if !context.user.CanEditProjects() {
		return nil, errors.New("forbidden to edit project")
	}

	return project, nil
}
