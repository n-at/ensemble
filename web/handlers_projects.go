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

	return c.Render(http.StatusOK, "templates/project_new.twig", pongo2.Context{
		"user": context.user,
	})
}

//projectNewSubmit Save and update new project
func (s *Server) projectNewSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)

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

	var err error
	if len(project.Name) == 0 {
		err = errors.New("project name should not be empty")
	}
	if len(project.RepositoryUrl) == 0 {
		err = errors.New("repository URL should not be empty")
	}
	if len(project.RepositoryBranch) == 0 {
		project.RepositoryBranch = structures.ProjectDefaultBranchName
	}
	if err != nil {
		return c.Render(http.StatusOK, "templates/project_new.twig", pongo2.Context{
			"user":    context.user,
			"project": project,
			"error":   err,
		})
	}

	err = s.store.ProjectInsert(project)
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

	return c.Redirect(http.StatusFound, "/projects")
}

//projectEditForm Project edit form
func (s *Server) projectEditForm(c echo.Context) error {
	context := c.(*EnsembleContext)

	return c.Render(http.StatusOK, "templates/project_edit.twig", pongo2.Context{
		"user":    context.user,
		"project": context.project,
	})
}

//projectEditSubmit Save changed project data
func (s *Server) projectEditSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)

	project := context.project

	project.Name = c.FormValue("name")
	project.Description = c.FormValue("description")
	project.RepositoryUrl = c.FormValue("repo_url")
	project.RepositoryLogin = c.FormValue("repo_login")
	project.RepositoryPassword = c.FormValue("repo_password")
	project.RepositoryBranch = c.FormValue("repo_branch")
	project.Inventory = c.FormValue("inventory")
	project.Variables = c.FormValue("variables")
	project.VaultPassword = c.FormValue("vault_password")

	var err error

	if len(project.Name) == 0 {
		err = errors.New("project name should not be empty")
	}
	if len(project.RepositoryUrl) == 0 {
		err = errors.New("repository URL should not be empty")
	}
	if len(project.RepositoryBranch) == 0 {
		project.RepositoryBranch = structures.ProjectDefaultBranchName
	}
	if len(project.Inventory) == 0 {
		project.Inventory = structures.ProjectDefaultInventoryName
	}

	inventoryFound := false
	for _, inventory := range project.InventoryList {
		if inventory == project.Inventory {
			inventoryFound = true
			break
		}
	}
	if !inventoryFound {
		err = errors.New("selected inventory not found")
	}

	variablesFound := false
	for _, variables := range project.VariablesList {
		if variables == project.Variables {
			variablesFound = true
			break
		}
	}
	if len(project.Variables) != 0 && !variablesFound {
		err = errors.New("selected variables not found")
	}

	if err != nil {
		return c.Render(http.StatusOK, "templates/project_edit.twig", pongo2.Context{
			"user":    context.user,
			"project": project,
			"error":   err,
		})
	}

	err = s.store.ProjectUpdate(project)
	if err != nil {
		log.Warnf("projectEditSubmit unable to update project: %s", err)
		return c.Render(http.StatusOK, "templates/project_edit.twig", pongo2.Context{
			"user":    context.user,
			"project": project,
			"error":   err,
		})
	}

	return c.Redirect(http.StatusFound, "/projects")
}

//projectDeleteForm Project delete form
func (s *Server) projectDeleteForm(c echo.Context) error {
	context := c.(*EnsembleContext)

	return c.Render(http.StatusOK, "templates/project_delete.twig", pongo2.Context{
		"user":    context.user,
		"project": context.project,
	})
}

//projectDeleteSubmit Delete selected project
func (s *Server) projectDeleteSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)

	err := s.store.ProjectDelete(context.project.Id)
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, "/projects")
}

//projectUpdate Update project repository
func (s *Server) projectUpdate(c echo.Context) error {
	context := c.(*EnsembleContext)

	err := s.manager.Update(context.project)
	if err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, "/projects")
}
