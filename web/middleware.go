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
		var user *structures.User
		var session *structures.Session

		cookie, err := c.Cookie(SessionCookieName)
		if err == nil {
			session, err = s.store.SessionGet(cookie.Value)
			if err == nil {
				user, err = s.store.UserGet(session.UserId)
				if err != nil {
					log.Warnf("unable to get user by session userId: %s", err)
					user = nil
				}
			} else {
				log.Warnf("unable to get session by sessionId: %s", err)
			}
		}

		ensembleContext := &EnsembleContext{
			Context: c,
			server:  s,
			session: session,
			user:    user,
		}
		return next(ensembleContext)
	}
}

func (s *Server) authenticationRequiredMiddleware(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		context := c.(*EnsembleContext)
		if context.user == nil {
			return c.Redirect(http.StatusFound, "/login")
		}
		return next(c)
	}
}

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
