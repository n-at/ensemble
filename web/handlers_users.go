package web

import (
	"ensemble/storage"
	"ensemble/storage/structures"
	"errors"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"net/http"
	"strconv"
)

type projectAccess struct {
	Project *structures.Project
	Access  bool
}

///////////////////////////////////////////////////////////////////////////////

func (s *Server) users(c echo.Context) error {
	users, err := s.store.UserGetAll()
	if err != nil {
		log.Errorf("users get all error: %s", err)
		return err
	}

	context := c.(*EnsembleContext)

	return c.Render(http.StatusOK, "templates/users.twig", pongo2.Context{
		"user":  context.user,
		"users": users,
	})
}

func (s *Server) userNewForm(c echo.Context) error {
	context := c.(*EnsembleContext)
	return c.Render(http.StatusOK, "templates/user_new.twig", pongo2.Context{
		"user": context.user,
	})
}

func (s *Server) userNewSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)

	log.Infof("userNewSubmit")

	password, err := storage.EncryptPassword(c.FormValue("password"))
	if err != nil {
		log.Errorf("userNewSubmit password encryption error: %s", err)
		return err
	}
	role, err := strconv.Atoi(c.FormValue("role"))
	if err != nil {
		log.Errorf("userNewSubmit role read error: %s", err)
		return err
	}
	if role != structures.UserRoleAdmin && role != structures.UserRoleOperator {
		log.Errorf("userNewSubmit unknown user role")
		return errors.New("unknown user role")
	}

	user := structures.User{
		Login:    c.FormValue("login"),
		Password: password,
		Role:     role,
	}
	if err := s.store.UserInsert(&user); err != nil {
		log.Errorf("userNewSubmit user save error: %s", err)
		return c.Render(http.StatusOK, "templates/user_new.twig", pongo2.Context{
			"user":         context.user,
			"user_control": &user,
			"error":        err,
		})
	}

	return c.Redirect(http.StatusFound, "/users")
}

func (s *Server) userEditForm(c echo.Context) error {
	context := c.(*EnsembleContext)
	return c.Render(http.StatusOK, "templates/user_edit.twig", pongo2.Context{
		"user":         context.user,
		"user_control": context.userControl,
	})
}

func (s *Server) userEditSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)

	log.Infof("userEditSubmit %s", context.userControl.Id)

	user := context.userControl
	user.Login = c.FormValue("login")

	password := c.FormValue("password")
	if len(password) != 0 {
		password, err := storage.EncryptPassword(password)
		if err != nil {
			log.Errorf("userEditSubmit user %s password encryption error: %s", context.userControl.Id, err)
			return err
		}
		user.Password = password
	}

	role, err := strconv.Atoi(c.FormValue("role"))
	if err != nil {
		log.Errorf("userEditSubmit user %s role read error: %s", context.userControl.Id, err)
		return err
	}
	if role != structures.UserRoleAdmin && role != structures.UserRoleOperator {
		log.Errorf("userEditSubmit user %s unknown role", context.userControl.Id)
		return errors.New("unknown user role")
	}
	user.Role = role

	if err := s.store.UserUpdate(user); err != nil {
		log.Errorf("userEditSubmit user %s save error: %s", context.userControl.Id, err)
		return c.Render(http.StatusOK, "templates/user_edit.twig", pongo2.Context{
			"user":         context.user,
			"user_control": user,
			"error":        err,
		})
	}

	return c.Redirect(http.StatusFound, "/users")
}

func (s *Server) userDeleteForm(c echo.Context) error {
	context := c.(*EnsembleContext)

	if context.user.Id == context.userControl.Id {
		return errors.New("cannot delete self")
	}

	return c.Render(http.StatusOK, "templates/user_delete.twig", pongo2.Context{
		"user":         context.user,
		"user_control": context.userControl,
	})
}

func (s *Server) userDeleteSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)

	log.Infof("userDeleteSubmit delete %s", context.userControl.Id)

	if context.user.Id == context.userControl.Id {
		log.Errorf("userDeleteSubmit delete self")
		return errors.New("cannot delete self")
	}
	if err := s.store.UserDelete(context.userControl.Id); err != nil {
		log.Errorf("userDeleteSubmit user %s delete error: %s", context.userControl.Id, err)
		return err
	}

	return c.Redirect(http.StatusFound, "/users")
}

func (s *Server) userProjectsForm(c echo.Context) error {
	context := c.(*EnsembleContext)

	if context.userControl.CanViewAllProjects() {
		return errors.New("user already has access to all projects")
	}

	projects, err := s.store.ProjectGetAll()
	if err != nil {
		log.Errorf("userProjectsForm get all error: %s", err)
		return err
	}

	userProjects, err := s.store.ProjectGetByUser(context.userControl.Id)
	if err != nil {
		log.Errorf("userProjectsForm user %s get projects: %s", context.userControl.Id, err)
		return err
	}
	userProjectAccess := make(map[string]bool)
	for _, project := range userProjects {
		userProjectAccess[project.Id] = true
	}

	var info []*projectAccess
	for _, project := range projects {
		info = append(info, &projectAccess{
			Project: project,
			Access:  userProjectAccess[project.Id],
		})
	}

	return c.Render(http.StatusOK, "templates/user_projects.twig", pongo2.Context{
		"user":         context.user,
		"user_control": context.userControl,
		"projects":     info,
	})
}

func (s *Server) userProjectsSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)
	userId := context.userControl.Id

	log.Infof("userProjectsSubmit %s", userId)

	if context.userControl.CanViewAllProjects() {
		return errors.New("user already has access to all projects")
	}

	var projectIds []string
	if err := echo.FormFieldBinder(c).Strings("projects[]", &projectIds).BindError(); err != nil {
		log.Errorf("userProjectsSubmit user %s projects read error: %s", userId, err)
		return err
	}
	newUserProjectAccess := make(map[string]bool)
	for _, projectId := range projectIds {
		newUserProjectAccess[projectId] = true
	}

	userProjects, err := s.store.ProjectGetByUser(userId)
	if err != nil {
		log.Errorf("userProjectsSubmit user %s projects get by user error: %s", userId, err)
		return err
	}
	userProjectAccess := make(map[string]bool)
	for _, project := range userProjects {
		userProjectAccess[project.Id] = true
	}

	projects, err := s.store.ProjectGetAll()
	if err != nil {
		log.Errorf("userProjectsSubmit user %s projects get error: %s", userId, err)
		return err
	}
	for _, project := range projects {
		if !userProjectAccess[project.Id] && newUserProjectAccess[project.Id] {
			if err := s.store.ProjectUserAccessCreate(project.Id, userId); err != nil {
				log.Errorf("userProjectsSubmit user %s project %s grant access error: %s", userId, project.Id, err)
				return err
			}
		}
		if userProjectAccess[project.Id] && !newUserProjectAccess[project.Id] {
			if err := s.store.ProjectUserAccessDelete(project.Id, userId); err != nil {
				log.Errorf("userProjectsSubmit user %s project %s revoke access error: %s", userId, project.Id, err)
				return err
			}
		}
	}

	return c.Redirect(http.StatusFound, "/users")
}
