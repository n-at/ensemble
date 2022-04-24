package web

import (
	"ensemble/storage"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"net/http"
)

///////////////////////////////////////////////////////////////////////////////
// index

func (s *Server) index(c echo.Context) error {
	context := c.(*EnsembleContext)

	if context.user != nil {
		return c.Redirect(http.StatusFound, "/projects/")
	} else {
		return c.Redirect(http.StatusFound, "/login")
	}
}

func (s *Server) loginForm(c echo.Context) error {
	context := c.(*EnsembleContext)
	if context.user != nil {
		return c.Redirect(http.StatusFound, "/")
	}

	return c.Render(http.StatusOK, "templates/login.twig", pongo2.Context{})
}

func (s *Server) loginSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)
	if context.user != nil {
		return c.Redirect(http.StatusFound, "/")
	}

	login := c.FormValue("login")
	password := c.FormValue("password")

	user, err := context.server.store.UserGetByLogin(login)
	if err != nil || user == nil {
		log.Warnf("loginSubmit get user error: %s", err)
		return c.Render(http.StatusOK, "templates/login.twig", pongo2.Context{
			"login": login,
			"error": "Incorrect login or password",
		})
	}

	if !storage.CheckPassword(password, user.Password) {
		return c.Render(http.StatusOK, "templates/login.twig", pongo2.Context{
			"login": login,
			"error": "Incorrect login or password",
		})
	}

	session, err := context.server.store.SessionCreate(user.Id)
	if err != nil {
		return c.Render(http.StatusOK, "templates/login.twig", pongo2.Context{
			"login": login,
			"error": err.Error(),
		})
	}

	context.SetSessionId(session.Id)

	return c.Redirect(http.StatusFound, "/")
}

func (s *Server) logout(c echo.Context) error {
	context := c.(*EnsembleContext)
	context.DeleteSessionId()
	return c.Redirect(http.StatusFound, "/login")
}
