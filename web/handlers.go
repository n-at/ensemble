package web

import (
	"github.com/labstack/echo/v4"
	"net/http"
)

func (s *Server) index(c echo.Context) error {
	return c.Render(http.StatusOK, "templates/index.twig", nil)
}
