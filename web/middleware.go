package web

import (
	"ensemble/storage/structures"
	"errors"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (s *Server) contextCustomizationMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		var session *structures.Session

		cookie, err := c.Cookie(SessionCookieName)
		if err == nil {
			session, err = s.store.SessionGet(cookie.Value)
			if err != nil {
				log.Warnf("unable to get session by sessionId: %s", err)
			}
		}

		ensembleContext := &EnsembleContext{
			Context: c,
			session: session,
		}
		return next(ensembleContext)
	}
}

///////////////////////////////////////////////////////////////////////////////

func (s *Server) authenticationRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		context := c.(*EnsembleContext)

		if context.session == nil || len(context.session.UserId) == 0 {
			return c.Redirect(http.StatusFound, "/login")
		}

		user, err := s.store.UserGet(context.session.UserId)
		if err != nil {
			log.Warnf("unable to get user by session userId: %s", err)
			return c.Redirect(http.StatusFound, "/login")
		}
		if user == nil {
			log.Warn("user by session not found")
			return c.Redirect(http.StatusFound, "/login")
		}

		context.user = user

		return next(context)
	}
}

///////////////////////////////////////////////////////////////////////////////

func (s *Server) projectRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		projectId := c.Param("project_id")
		if len(projectId) == 0 {
			return errors.New("project id required")
		}

		project, err := s.store.ProjectGet(projectId)
		if err != nil {
			return err
		}
		if project == nil {
			return errors.New("project not found")
		}

		context := c.(*EnsembleContext)

		if !context.user.CanEditProjects() && !s.store.ProjectUserAccessExists(project.Id, context.user.Id) {
			return errors.New("project access denied")
		}

		context.project = project

		return next(context)
	}
}

func (s *Server) projectCreateAccessRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		context := c.(*EnsembleContext)
		if !context.user.CanCreateProjects() {
			return errors.New("projects creation access denied")
		}
		return next(c)
	}
}

func (s *Server) projectWriteAccessRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		context := c.(*EnsembleContext)
		if !context.user.CanEditProjects() {
			return errors.New("project access denied")
		}
		return next(c)
	}
}

///////////////////////////////////////////////////////////////////////////////

func (s *Server) projectUpdateRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		projectUpdateId := c.Param("project_update_id")
		if len(projectUpdateId) == 0 {
			return errors.New("project update id required")
		}

		projectUpdate, err := s.store.ProjectUpdateGet(projectUpdateId)
		if err != nil {
			return err
		}
		if projectUpdate == nil {
			return errors.New("project update not found")
		}

		context := c.(*EnsembleContext)
		context.projectUpdate = projectUpdate

		return next(context)
	}
}

///////////////////////////////////////////////////////////////////////////////

func (s *Server) playbookRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		playbookId := c.Param("playbook_id")
		if len(playbookId) == 0 {
			return errors.New("playbook is required")
		}

		context := c.(*EnsembleContext)

		if context.project == nil {
			return errors.New("project access required")
		}

		playbook, err := s.store.PlaybookGet(playbookId)
		if err != nil {
			return err
		}
		if playbook == nil {
			return errors.New("playbook not found")
		}

		context.playbook = playbook

		return next(context)
	}
}

func (s *Server) playbookLockAccessRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		context := c.(*EnsembleContext)
		if !context.user.CanLockPlaybooks() {
			return errors.New("cannot lock playbooks")
		}
		return next(c)
	}
}

///////////////////////////////////////////////////////////////////////////////

func (s *Server) playbookRunRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		context := c.(*EnsembleContext)

		playbookRunId := c.Param("playbook_run_id")
		if len(playbookRunId) == 0 {
			return errors.New("playbook run id required")
		}

		playbookRun, err := s.store.PlaybookRunGet(playbookRunId)
		if err != nil {
			return err
		}
		if playbookRun == nil {
			return errors.New("playbook run not found")
		}
		if playbookRun.PlaybookId != context.playbook.Id {
			return errors.New("playbook run does not belong to project")
		}

		context.playbookRun = playbookRun

		return next(context)
	}
}

func (s *Server) playbookRunDeleteAccessRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		context := c.(*EnsembleContext)
		if !context.user.CanDeletePlaybookRuns() {
			return errors.New("playbook run delete denied")
		}
		return next(c)
	}
}

///////////////////////////////////////////////////////////////////////////////

func (s *Server) userControlAccessRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		context := c.(*EnsembleContext)

		if context.user == nil {
			return errors.New("auth required")
		}
		if !context.user.CanControlUsers() {
			return errors.New("user control access denied")
		}

		return next(c)
	}
}

func (s *Server) userControlRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		userId := c.Param("user_id")
		if len(userId) == 0 {
			return errors.New("user id required")
		}

		user, err := s.store.UserGet(userId)
		if err != nil {
			return err
		}
		if user == nil {
			return errors.New("user not found")
		}

		context := c.(*EnsembleContext)
		context.userControl = user

		return next(context)
	}
}

///////////////////////////////////////////////////////////////////////////////

func (s *Server) keyAccessRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		context := c.(*EnsembleContext)

		if context.user == nil {
			return errors.New("auth required")
		}
		if !context.user.CanControlKeys() {
			return errors.New("keys access denied")
		}

		return next(c)
	}
}

func (s *Server) keyRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		context := c.(*EnsembleContext)

		keyId := c.Param("key_id")
		if len(keyId) == 0 {
			return errors.New("key id required")
		}
		key, err := s.store.KeyGet(keyId)
		if err != nil {
			return err
		}
		if key == nil {
			return errors.New("key not found")
		}

		context.key = key

		return next(context)
	}
}
