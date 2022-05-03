package web

import (
	"encoding/json"
	"ensemble/storage/structures"
	"fmt"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	"github.com/labstack/gommon/log"
	"net/http"
)

func (s *Server) playbookRuns(c echo.Context) error {
	context := c.(*EnsembleContext)

	runs, err := s.store.PlaybookRunGetByPlaybook(context.playbook.Id)
	if err != nil {
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
	if err == nil {
		if err := json.Unmarshal([]byte(runResult.Output), ansibleResult); err != nil {
			log.Warnf("unable to unmarshal run result for %s", context.playbookRun.Id)
		}
	} else {
		log.Warnf("unable to get run result for %s", context.playbookRun.Id)
	}

	runUser, err = s.store.UserGet(context.playbookRun.UserId)
	if err != nil {
		log.Warnf("unable to get run user for %s", context.playbookRun.Id)
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

	if err := s.store.PlaybookRunDelete(context.playbookRun.Id); err != nil {
		return err
	}

	url := fmt.Sprintf("/projects/playbooks/%s/runs/%s", context.project.Id, context.playbook.Id)

	return c.Redirect(http.StatusFound, url)
}

func (s *Server) playbookRunStatus(c echo.Context) error {
	context := c.(*EnsembleContext)
	return c.JSON(http.StatusOK, context.playbookRun.Result)
}
