package web

import (
	"encoding/json"
	"ensemble/storage/structures"
	"errors"
	"fmt"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
	"net/http"
)

func (s *Server) playbookRuns(c echo.Context) error {
	context := c.(*EnsembleContext)

	runs, err := s.store.PlaybookRunGetByPlaybook(context.playbook.Id)
	if err != nil {
		log.Errorf("playbookRuns playbook %s runs get error: %s", context.playbook.Id, err)
		return err
	}

	return c.Render(http.StatusOK, "templates/playbook_runs.twig", pongo2.Context{
		"user":     context.user,
		"project":  context.project,
		"playbook": context.playbook,
		"runs":     runs,
	})
}

func (s *Server) playbookRunResult(c echo.Context) error {
	context := c.(*EnsembleContext)

	var ansibleResult *structures.AnsibleExecution
	var runResult *structures.RunResult
	var runUser *structures.User

	runResult, err := s.store.RunResultGet(context.playbookRun.Id)
	if err != nil {
		log.Warnf("playbookRunResult playbook run %s get result error: %s", context.playbookRun.Id, err)
	} else if context.playbookRun.Mode != structures.PlaybookRunModeSyntax {
		ansibleResult = &structures.AnsibleExecution{}
		if err := json.Unmarshal([]byte(runResult.Output), ansibleResult); err != nil {
			log.Warnf("playbookRunResult playbook run %s unmarshal error: %s", context.playbookRun.Id, err)
			ansibleResult = nil
		}
	}

	runUser, err = s.store.UserGet(context.playbookRun.UserId)
	if err != nil {
		log.Warnf("playbookRunResult playbook run %s get user error: %s", context.playbookRun.Id, err)
	}

	return c.Render(http.StatusOK, "templates/playbook_run_result.twig", pongo2.Context{
		"user":               context.user,
		"project":            context.project,
		"playbook":           context.playbook,
		"run":                context.playbookRun,
		"run_result":         runResult,
		"run_result_ansible": ansibleResult,
		"run_user":           runUser,
	})
}

func (s *Server) playbookRunDeleteForm(c echo.Context) error {
	context := c.(*EnsembleContext)

	return c.Render(http.StatusOK, "templates/playbook_run_delete.twig", pongo2.Context{
		"user":     context.user,
		"project":  context.project,
		"playbook": context.playbook,
		"run":      context.playbookRun,
	})
}

func (s *Server) playbookRunDeleteSubmit(c echo.Context) error {
	context := c.(*EnsembleContext)

	log.Infof("playbookRunDeleteSubmit %s", context.playbookRun.Id)

	if err := s.store.PlaybookRunDelete(context.playbookRun.Id); err != nil {
		log.Errorf("playbookRunDeleteSubmit playbook run %s delete error: %s", context.playbookRun.Id, err)
		return err
	}

	returnUrl := fmt.Sprintf("/projects/playbooks/%s/runs/%s", context.project.Id, context.playbook.Id)

	return c.Redirect(http.StatusFound, returnUrl)
}

func (s *Server) playbookRunStatus(c echo.Context) error {
	context := c.(*EnsembleContext)
	return c.JSON(http.StatusOK, context.playbookRun.Result)
}

func (s *Server) playbookRunTerminate(c echo.Context) error {
	context := c.(*EnsembleContext)

	log.Infof("playbookRunTerminate %s", context.playbookRun.Id)

	if err := s.runner.TerminatePlaybook(context.playbookRun.Id); err != nil {
		log.Errorf("playbookRunTerminate playbook run %s termination error: %s", context.playbookRun.Id, err)
		return err
	}

	returnUrl := fmt.Sprintf("/projects/playbooks/%s/runs/%s/result/%s", context.project.Id, context.playbook.Id, context.playbookRun.Id)

	return c.Redirect(http.StatusFound, returnUrl)
}

func (s *Server) playbookRunDownload(c echo.Context) error {
	context := c.(*EnsembleContext)

	log.Infof("playbookRunDownload %s", context.playbookRun.Id)

	result, err := s.store.RunResultGet(context.playbookRun.Id)
	if err != nil {
		log.Errorf("playbookRunDownload run result %s get error: %s", context.playbookRun.Id, err)
		return err
	}
	if result == nil {
		log.Warnf("playbookRunDownload run result %s not found", context.playbookRun.Id)
		return errors.New("playbook run result not found")
	}
	if len(result.Output) == 0 {
		log.Warnf("playbookRunDownload run result %s output is empty", context.playbookRun.Id)
		return errors.New("playbook run result not found")
	}

	return c.JSONBlob(http.StatusOK, []byte(result.Output))
}
