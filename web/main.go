package web

import (
	"ensemble/repository"
	"ensemble/runner"
	"ensemble/storage"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"net/http"
	"time"
)

type Configuration struct {
	DebugTemplates bool
	Listen         string
}

type Server struct {
	e       *echo.Echo
	store   *storage.Storage
	manager *repository.Manager
	runner  *runner.Runner
}

type EnsembleContext struct {
	echo.Context
	server *Server
}

///////////////////////////////////////////////////////////////////////////////

func New(configuration Configuration, store *storage.Storage, manager *repository.Manager, runner *runner.Runner) *Server {
	e := echo.New()

	e.HideBanner = true
	e.Renderer = Pongo2Renderer{Debug: configuration.DebugTemplates}
	e.HTTPErrorHandler = httpErrorHandler
	e.Static("/assets", "assets")

	server := &Server{
		e:       e,
		store:   store,
		manager: manager,
		runner:  runner,
	}

	server.e.Use(server.contextCustomizationHandler)

	//TODO handlers
	server.e.GET("/", server.index)

	return server
}

///////////////////////////////////////////////////////////////////////////////

func (s *Server) Start(listen string) error {
	return s.e.Start(listen)
}

func (s *Server) contextCustomizationHandler(next echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		ensembleContext := &EnsembleContext{c, s}
		return next(ensembleContext)
	}
}

// httpErrorHandler Custom HTTP error handler
func httpErrorHandler(e error, c echo.Context) {
	code := http.StatusInternalServerError
	if httpError, ok := e.(*echo.HTTPError); ok {
		code = httpError.Code
	}

	log.Errorf("http error: %s, method=%s, url=%s", e, c.Request().Method, c.Request().URL)

	err := c.Render(code, "templates/error.twig", pongo2.Context{
		"error": e,
	})
	if err != nil {
		log.Errorf("error page render error: %s", err)
	}
}

///////////////////////////////////////////////////////////////////////////////

func (c *EnsembleContext) GetSessionId() string {
	cookie, err := c.Cookie("ensemble-session-id")
	if err != nil {
		return ""
	}
	return cookie.Value
}

func (c *EnsembleContext) SetSessionId(id string) {
	cookie := &http.Cookie{
		Name:    "ensemble-session-id",
		Value:   id,
		Expires: time.Now().Add(-24 * time.Hour),
	}
	c.SetCookie(cookie)
}

func (c *EnsembleContext) DeleteSessionId() {
	cookie := &http.Cookie{
		Name:    "ensemble-session-id",
		Value:   "",
		Expires: time.Now().Add(-24 * time.Hour),
	}
	c.SetCookie(cookie)
}
