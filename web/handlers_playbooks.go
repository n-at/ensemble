package web

import (
	"ensemble/storage/structures"
	"errors"
	"fmt"
	"github.com/flosch/pongo2/v4"
	"github.com/labstack/echo/v4"
	log "github.com/sirupsen/logrus"
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
		log.Errorf("playbooks project %s playbooks get error: %s", context.project.Id, err)
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

	log.Infof("playbookRun playbook %s run mode %s", context.playbook.Id, operation)

	run, err := s.runner.Run(context.project, context.playbook, mode, context.user.Id)
	if err != nil {
		log.Errorf("playbookRun playbook %s mode %s run error: %s", context.playbook.Id, operation, err)
		return err
	}

	returnUrl := fmt.Sprintf("/projects/playbooks/%s/runs/%s/result/%s", context.project.Id, context.playbook.Id, run.Id)

	return c.Redirect(http.StatusFound, returnUrl)
}

func (s *Server) playbookLock(c echo.Context) error {
	context := c.(*EnsembleContext)

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

	log.Infof("playbookLock playbook %s lock %s", context.playbook.Id, operation)

	if err := s.store.PlaybookLock(context.playbook.Id, lock); err != nil {
		log.Errorf("playbookLock playbook %s lock %s error: %s", context.playbook.Id, operation, err)
		return err
	}

	returnUrl := fmt.Sprintf("/projects/playbooks/%s", context.project.Id)

	return c.Redirect(http.StatusFound, returnUrl)
}
