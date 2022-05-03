package web

import (
	"ensemble/storage/structures"
	"errors"
	"fmt"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	"net/http"
)

type playbookInfo struct {
	Playbook *structures.Playbook
	Run      *structures.PlaybookRun
}

///////////////////////////////////////////////////////////////////////////////

func (s *Server) playbooks(c echo.Context) error {
	context := c.(*EnsembleContext)

	playbooks, err := s.store.PlaybookGetByProject(context.project.Id)
	if err != nil {
		return err
	}

	var info []*playbookInfo

	for _, playbook := range playbooks {
		run, _ := s.store.PlaybookRunGetLatest(playbook.Id)
		info = append(info, &playbookInfo{
			Playbook: playbook,
			Run:      run,
		})
	}

	return c.Render(http.StatusOK, "templates/playbooks.twig", pongo2.Context{
		"user":      context.user,
		"project":   context.project,
		"playbooks": info,
	})
}

func (s *Server) playbookRun(c echo.Context) error {
	context := c.(*EnsembleContext)

	mode := structures.PlaybookRunModeCheck

	operation := c.Param("operation")
	switch operation {
	case "execute":
		mode = structures.PlaybookRunModeExecute
	case "check":
		mode = structures.PlaybookRunModeCheck
	default:
		return errors.New("unknown run mode")
	}

	if context.playbook.Locked {
		return errors.New("playbook is locked")
	}

	run, err := s.runner.Run(context.project, context.playbook, mode, context.user.Id)
	if err != nil {
		return err
	}

	url := fmt.Sprintf("/projects/playbooks/%s/results/%s/log/%s", context.project.Id, context.playbook.Id, run.Id)

	return c.Redirect(http.StatusFound, url)
}

func (s *Server) playbookLock(c echo.Context) error {
	context := c.(*EnsembleContext)

	returnUrl := fmt.Sprintf("/projects/playbooks/%s", context.project.Id)
	lock := true

	operation := c.Param("operation")
	switch operation {
	case "lock":
		lock = true
	case "unlock":
		lock = false
	default:
		return errors.New("unknown lock operation")
	}

	if !context.user.CanLockPlaybooks() {
		return errors.New("cannot lock playbooks")
	}
	if err := s.store.PlaybookLock(context.playbook.Id, lock); err != nil {
		return err
	}

	return c.Redirect(http.StatusFound, returnUrl)
}
