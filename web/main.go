package web

import (
	"ensemble/repository"
	"ensemble/runner"
	"ensemble/storage"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"net/http"
)

const (
	SessionCookieName = "ensemble-session-id"
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

///////////////////////////////////////////////////////////////////////////////

func New(configuration Configuration, store *storage.Storage, manager *repository.Manager, runner *runner.Runner) *Server {
	e := echo.New()

	e.HideBanner = true
	e.Renderer = Pongo2Renderer{Debug: configuration.DebugTemplates}
	e.HTTPErrorHandler = httpErrorHandler
	e.Static("/assets", "assets")

	s := &Server{
		e:       e,
		store:   store,
		manager: manager,
		runner:  runner,
	}

	s.e.Use(s.contextCustomizationMiddleware)

	s.e.GET("/", s.index)

	//authentication
	s.e.GET("/login", s.loginForm)
	s.e.POST("/login", s.loginSubmit)
	s.e.GET("/logout", s.logout)

	//projects
	projects := s.e.Group("/projects")
	projects.Use(s.authenticationRequiredMiddleware)
	projects.GET("", s.projects)

	projectNew := projects.Group("/new")
	projectNew.Use(s.projectCreateAccessRequiredMiddleware)
	projectNew.GET("", s.projectNewForm)
	projectNew.POST("", s.projectNewSubmit)

	projectEdit := projects.Group("/edit")
	projectEdit.Use(s.projectRequiredMiddleware)
	projectEdit.Use(s.projectWriteAccessRequiredMiddleware)
	projectEdit.GET("/:project_id", s.projectEditForm)
	projectEdit.POST("/:project_id", s.projectEditSubmit)

	projectDelete := projects.Group("/delete")
	projectDelete.Use(s.projectRequiredMiddleware)
	projectDelete.Use(s.projectWriteAccessRequiredMiddleware)
	projectDelete.GET("/:project_id", s.projectDeleteForm)
	projectDelete.POST("/:project_id", s.projectDeleteSubmit)

	projectUpdate := projects.Group("/update")
	projectUpdate.Use(s.projectRequiredMiddleware)
	projectUpdate.GET("/:project_id", s.projectUpdate)

	projectUpdates := projects.Group("/updates/:project_id")
	projectUpdates.Use(s.projectRequiredMiddleware)
	projectUpdates.Use(s.projectWriteAccessRequiredMiddleware)
	projectUpdates.GET("", s.projectUpdates)

	projectUpdateLog := projectUpdates.Group("/log")
	projectUpdateLog.Use(s.projectUpdateRequiredMiddleware)
	projectUpdateLog.GET("/:project_update_id", s.projectUpdateLog)

	projectUpdateDelete := projectUpdates.Group("/delete")
	projectUpdateDelete.Use(s.projectUpdateRequiredMiddleware)
	projectUpdateDelete.GET("/:project_update_id", s.projectUpdateDeleteForm)
	projectUpdateDelete.POST("/:project_update_id", s.projectUpdateDeleteSubmit)

	//users
	users := s.e.Group("/users")
	users.Use(s.authenticationRequiredMiddleware)
	users.Use(s.userControlAccessRequiredMiddleware)
	users.GET("", s.users)
	users.GET("/new", s.userNewForm)
	users.POST("/new", s.userNewSubmit)

	usersEdit := users.Group("/edit/:user_id")
	usersEdit.Use(s.userControlRequiredMiddleware)
	usersEdit.GET("", s.userEditForm)
	usersEdit.POST("", s.userEditSubmit)

	usersDelete := users.Group("/delete/:user_id")
	usersDelete.Use(s.userControlRequiredMiddleware)
	usersDelete.GET("", s.userDeleteForm)
	usersDelete.POST("", s.userDeleteSubmit)

	usersProjects := users.Group("/projects/:user_id")
	usersProjects.Use(s.userControlRequiredMiddleware)
	usersProjects.GET("", s.userProjectsForm)
	usersProjects.POST("", s.userProjectsSubmit)

	return s
}

func (s *Server) Start(listen string) error {
	return s.e.Start(listen)
}

///////////////////////////////////////////////////////////////////////////////

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
