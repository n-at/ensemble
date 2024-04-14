package web

import (
	"ensemble/storage"
	"errors"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (s *Server) index(c echo.Context) error {
	return c.Redirect(http.StatusFound, "/projects")
}

func (s *Server) loginForm(c echo.Context) error {
	context := c.(*EnsembleContext)
	if context.user != nil {
		return c.Redirect(http.StatusFound, "/")
	}

	return c.Render(http.StatusOK, "templates/login.twig", pongo2.Context{
		"_csrf_token": c.Get("csrf"),
	})
}

func (s *Server) loginSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)
	if context.user != nil {
		return c.Redirect(http.StatusFound, "/")
	}

	login := c.FormValue("login")
	password := c.FormValue("password")

	log.Infof("loginSubmit: %s", login)

	user, err := s.store.UserGetByLogin(login)
	if err != nil || user == nil {
		log.Errorf("loginSubmit user %s get user error: %s", login, err)
		return c.Render(http.StatusOK, "templates/login.twig", pongo2.Context{
			"_csrf_token": c.Get("csrf"),
			"login":       login,
			"error":       "Incorrect login or password",
		})
	}

	if !storage.CheckPassword(password, user.Password) {
		log.Warnf("loginSubmit user %s check password error", login)
		return c.Render(http.StatusOK, "templates/login.twig", pongo2.Context{
			"_csrf_token": c.Get("csrf"),
			"login":       login,
			"error":       "Incorrect login or password",
		})
	}

	session, err := s.store.SessionCreate(user.Id)
	if err != nil {
		log.Errorf("loginSubmit user %s session creation error: %s", login, err)
		return c.Render(http.StatusOK, "templates/login.twig", pongo2.Context{
			"_csrf_token": c.Get("csrf"),
			"login":       login,
			"error":       err.Error(),
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

///////////////////////////////////////////////////////////////////////////////

func (s *Server) profileForm(c echo.Context) error {
	context := c.(*EnsembleContext)

	return c.Render(http.StatusOK, "templates/profile.twig", pongo2.Context{
		"_csrf_token": c.Get("csrf"),
		"user":        context.user,
		"done":        c.QueryParam("done"),
	})
}

func (s *Server) profileSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)

	log.Infof("profileSubmit for %s", context.user.Login)

	currentPassword := c.FormValue("password")
	if !storage.CheckPassword(currentPassword, context.user.Password) {
		log.Warnf("profileSubmit user %s check password error", context.user.Login)
		return errors.New("wrong current password")
	}

	password1 := c.FormValue("password1")
	password2 := c.FormValue("password2")
	if password1 != password2 {
		return errors.New("passwords does not match")
	}
	if len(password1) == 0 {
		return errors.New("password should not be empty")
	}

	passwordEncrypted, err := storage.EncryptPassword(password1)
	if err != nil {
		log.Errorf("profileSubmit user %s password encryption error: %s", context.user.Login, err)
		return err
	}

	user := context.user
	user.Password = passwordEncrypted
	if err := s.store.UserUpdate(user); err != nil {
		log.Errorf("profileSubmit user %s update error: %s", context.user.Login, err)
		return err
	}

	return c.Redirect(http.StatusFound, "/profile?done=1")
}
