package web

import (
	"ensemble/storage"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"net/http"
)

type Configuration struct {
	DebugTemplates bool
	Listen         string
}

type Server struct {
	e     *echo.Echo
	store *storage.Storage
}

///////////////////////////////////////////////////////////////////////////////

func New(configuration Configuration, store *storage.Storage) *Server {
	e := echo.New()

	e.HideBanner = true
	e.Renderer = Pongo2Renderer{Debug: configuration.DebugTemplates}
	e.HTTPErrorHandler = httpErrorHandler
	e.Static("/assets", "assets")

	server := &Server{
		e:     e,
		store: store,
	}

	//TODO handlers
	server.e.GET("/", server.index)

	return server
}

///////////////////////////////////////////////////////////////////////////////

func (s *Server) Start(listen string) error {
	return s.e.Start(listen)
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
