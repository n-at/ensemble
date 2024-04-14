package web

import (
	"ensemble/storage"
	"ensemble/storage/structures"
	"errors"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (s *Server) keys(c echo.Context) error {
	context := c.(*EnsembleContext)

	keys, err := s.store.KeyGetAll()
	if err != nil {
		log.Errorf("keys get all error: %s", err)
		return err
	}

	return c.Render(http.StatusOK, "templates/keys.twig", pongo2.Context{
		"_csrf_token": c.Get("csrf"),
		"user":        context.user,
		"keys":        keys,
	})
}

func (s *Server) keyNewForm(c echo.Context) error {
	context := c.(*EnsembleContext)

	return c.Render(http.StatusOK, "templates/key_new.twig", pongo2.Context{
		"_csrf_token": c.Get("csrf"),
		"user":        context.user,
	})
}

func (s *Server) keyNewSubmit(c echo.Context) error {
	key := &structures.Key{
		Id:       storage.NewId(),
		Name:     c.FormValue("name"),
		Password: c.FormValue("password"),
	}
	content := c.FormValue("content")

	if len(key.Name) == 0 {
		return errors.New("key name required")
	}
	if len(content) == 0 {
		return errors.New("key content required")
	}

	if err := s.keyManager.Save(key, content); err != nil {
		log.Errorf("keyNewSubmit save file error: %s", err)
		return err
	}
	if err := s.store.KeyInsert(key); err != nil {
		log.Errorf("keyNewSubmit insert error: %s", err)
		return err
	}

	return c.Redirect(http.StatusFound, "/keys")
}

func (s *Server) keyDeleteForm(c echo.Context) error {
	context := c.(*EnsembleContext)

	return c.Render(http.StatusOK, "templates/key_delete.twig", pongo2.Context{
		"_csrf_token": c.Get("csrf"),
		"user":        context.user,
		"key":         context.key,
	})
}

func (s *Server) keyDeleteSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)

	if err := s.store.KeyDelete(context.key.Id); err != nil {
		log.Errorf("keyDeleteSubmit key %s delete error: %s", context.key.Id, err)
		return err
	}

	return c.Redirect(http.StatusFound, "/keys")
}
