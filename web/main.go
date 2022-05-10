package web

import (
	"ensemble/privatekeys"
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
	e          *echo.Echo
	store      *storage.Storage
	manager    *repository.Manager
	runner     *runner.Runner
	keyManager *privatekeys.KeyManager
}

///////////////////////////////////////////////////////////////////////////////

func New(configuration Configuration, store *storage.Storage, manager *repository.Manager, runner *runner.Runner, keyManager *privatekeys.KeyManager) *Server {
	e := echo.New()

	e.HideBanner = true
	e.Renderer = Pongo2Renderer{Debug: configuration.DebugTemplates}
	e.HTTPErrorHandler = httpErrorHandler
	e.Static("/assets", "assets")

	s := &Server{
		e:          e,
		store:      store,
		manager:    manager,
		runner:     runner,
		keyManager: keyManager,
	}

	s.e.Use(s.contextCustomizationMiddleware)

	s.e.GET("/", s.index)

	//authentication
	s.e.GET("/login", s.loginForm)
	s.e.POST("/login", s.loginSubmit)
	s.e.GET("/logout", s.logout)

	//user profile
	profile := s.e.Group("/profile")
	profile.Use(s.authenticationRequiredMiddleware)
	profile.GET("", s.profileForm)
	profile.POST("", s.profileSubmit)

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

	playbooks := projects.Group("/playbooks/:project_id")
	playbooks.Use(s.projectRequiredMiddleware)
	playbooks.GET("", s.playbooks)

	playbookLock := playbooks.Group("/lock")
	playbookLock.Use(s.playbookRequiredMiddleware)
	playbookLock.Use(s.playbookLockAccessRequiredMiddleware)
	playbookLock.GET("/:playbook_id/:operation", s.playbookLock)

	playbookRun := playbooks.Group("/run")
	playbookRun.Use(s.playbookRequiredMiddleware)
	playbookRun.GET("/:playbook_id/:operation", s.playbookRun)

	playbookRuns := playbooks.Group("/runs/:playbook_id")
	playbookRuns.Use(s.playbookRequiredMiddleware)
	playbookRuns.GET("", s.playbookRuns)

	playbookRunResult := playbookRuns.Group("/result")
	playbookRunResult.Use(s.playbookRunRequiredMiddleware)
	playbookRunResult.GET("/:playbook_run_id", s.playbookRunResult)

	playbookRunDelete := playbookRuns.Group("/delete")
	playbookRunDelete.Use(s.playbookRunRequiredMiddleware)
	playbookRunDelete.Use(s.projectWriteAccessRequiredMiddleware)
	playbookRunDelete.GET("/:playbook_run_id", s.playbookRunDeleteForm)
	playbookRunDelete.POST("/:playbook_run_id", s.playbookRunDeleteSubmit)

	playbookRunStatus := playbookRuns.Group("/status")
	playbookRunStatus.Use(s.playbookRunRequiredMiddleware)
	playbookRunStatus.GET("/:playbook_run_id", s.playbookRunStatus)

	playbookRunTerminate := playbookRuns.Group("/terminate")
	playbookRunTerminate.Use(s.playbookRunRequiredMiddleware)
	playbookRunTerminate.POST("/:playbook_run_id", s.playbookRunTerminate)

	playbookRunDownload := playbookRuns.Group("/download")
	playbookRunDownload.Use(s.playbookRunRequiredMiddleware)
	playbookRunDownload.GET("/:playbook_run_id", s.playbookRunDownload)

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

	//keys
	keys := s.e.Group("/keys")
	keys.Use(s.authenticationRequiredMiddleware)
	keys.Use(s.keyAccessRequiredMiddleware)
	keys.GET("", s.keys)
	keys.GET("/new", s.keyNewForm)
	keys.POST("/new", s.keyNewSubmit)

	keysDelete := keys.Group("/delete")
	keysDelete.Use(s.keyRequiredMiddleware)
	keysDelete.GET("/:key_id", s.keyDeleteForm)
	keysDelete.POST("/:key_id", s.keyDeleteSubmit)

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
