package repository

import (
	"ensemble/storage"
	"ensemble/storage/structures"
	"errors"
	"fmt"
	"github.com/alessio/shellescape"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Manager struct {
	config Configuration
	store  *storage.Storage
}

type Configuration struct {
	Path string
}

type result struct {
	success bool
	output  string
}

///////////////////////////////////////////////////////////////////////////////

func New(config Configuration, store *storage.Storage) *Manager {
	return &Manager{
		config: config,
		store:  store,
	}
}

///////////////////////////////////////////////////////////////////////////////

// UpdateAll Updates all projects in storage
func (m *Manager) UpdateAll() {
	projects, err := m.store.ProjectGetAll()
	if err != nil {
		log.Warnf("unable to get projects: %s", err)
		return
	}

	for _, project := range projects {
		if err := m.Update(project); err != nil {
			log.Warnf("unable to update project %s: %s", project.Name, err)
		}
	}
}

func (m *Manager) Update(project *structures.Project) error {
	output := strings.Builder{}
	success := false
	revision := "unknown revision"

	defer func() {
		if err := m.saveProjectUpdate(project, revision, success, output.String()); err != nil {
			log.Errorf("unable to save project update %s: %s", project.Id, err)
		}
	}()

	if err := m.ensureProjectDirectoryExists(project); err != nil {
		return err
	}

	exists, err := m.projectRepositoryExists(project)
	if err != nil {
		return err
	}
	if !exists {
		cloneResult, err := m.clone(project)
		if err != nil {
			return err
		}
		output.WriteString(fmt.Sprintf("> git clone\n\n%s\n\n", cloneResult.output))
		if !cloneResult.success {
			return errors.New("unable to execute git clone")
		}
	}

	originResult, err := m.origin(project)
	if err != nil {
		return err
	}
	output.WriteString(fmt.Sprintf("> git remote set-url\n\n%s\n\n", originResult.output))
	if !originResult.success {
		return errors.New("unable to execute git remote set-url")
	}

	resetResult, err := m.reset(project)
	if err != nil {
		return err
	}
	output.WriteString(fmt.Sprintf("> git reset\n\n%s\n\n", resetResult.output))
	if !resetResult.success {
		return errors.New("unable to execute git reset")
	}

	pullResult, err := m.pull(project)
	if err != nil {
		return err
	}
	output.WriteString(fmt.Sprintf("> git pull\n\n%s\n\n", pullResult.output))
	if !pullResult.success {
		return errors.New("unable to execute git pull")
	}

	checkoutResult, err := m.checkout(project)
	if err != nil {
		return err
	}
	output.WriteString(fmt.Sprintf("> git checkout\n\n%s\n\n", checkoutResult.output))
	if !checkoutResult.success {
		return errors.New("unable to execute git checkout")
	}

	revision, err = m.revision(project)
	if err != nil {
		return err
	}
	output.WriteString(fmt.Sprintf("> current revision: %s\n", revision))

	success = true

	return nil
}

///////////////////////////////////////////////////////////////////////////////

func (m *Manager) projectDirectory(p *structures.Project) string {
	return fmt.Sprintf("%s%c%s", m.config.Path, os.PathSeparator, p.Id)
}

func (m *Manager) projectDirectoryExists(p *structures.Project) (bool, error) {
	return directoryExists(m.projectDirectory(p))
}

func (m *Manager) projectRepositoryExists(p *structures.Project) (bool, error) {
	repoDir := fmt.Sprintf("%s%c%s", m.projectDirectory(p), os.PathSeparator, ".git")
	return directoryExists(repoDir)
}

func (m *Manager) ensureProjectDirectoryExists(p *structures.Project) error {
	exists, err := m.projectDirectoryExists(p)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	if err := os.Mkdir(m.projectDirectory(p), 0777); err != nil {
		return err
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////

func (m *Manager) clone(p *structures.Project) (*result, error) {
	command := fmt.Sprintf("git clone --branch %s %s .", shellescape.Quote(p.RepositoryBranch), shellescape.Quote(p.RepositoryUrl))
	return m.executeCommand(command, m.projectDirectory(p))
}

func (m *Manager) revision(p *structures.Project) (string, error) {
	result, err := m.executeCommand("git log --oneline -n 1", m.projectDirectory(p))
	if err != nil {
		return "", err
	}
	if !result.success {
		return "", errors.New("unable to execute git log")
	}
	return result.output[:structures.ProjectUpdateRevisionMaxLength], nil
}

func (m *Manager) origin(p *structures.Project) (*result, error) {
	command := fmt.Sprintf("git remote set-url origin %s", shellescape.Quote(p.RepositoryUrl))
	return m.executeCommand(command, m.projectDirectory(p))
}

func (m *Manager) reset(p *structures.Project) (*result, error) {
	return m.executeCommand("git reset --hard", m.projectDirectory(p))
}

func (m *Manager) checkout(p *structures.Project) (*result, error) {
	command := fmt.Sprintf("git checkout %s", shellescape.Quote(p.RepositoryBranch))
	return m.executeCommand(command, m.projectDirectory(p))
}

func (m *Manager) pull(p *structures.Project) (*result, error) {
	return m.executeCommand("git pull", m.projectDirectory(p))
}

///////////////////////////////////////////////////////////////////////////////

func (m *Manager) saveProjectUpdate(p *structures.Project, revision string, success bool, output string) error {
	update := structures.ProjectUpdate{
		ProjectId: p.Id,
		Date:      time.Now(),
		Success:   success,
		Revision:  revision,
		Log:       output,
	}
	return m.store.ProjectUpdateInsert(&update)
}

///////////////////////////////////////////////////////////////////////////////

func directoryExists(dir string) (bool, error) {
	_, err := os.Stat(dir)
	if err == nil {
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func (m *Manager) executeCommand(command, directory string) (*result, error) {
	cmd := exec.Command("sh", "-c", command)
	cmd.Dir = directory

	output, err := cmd.CombinedOutput()
	if err != nil {
		if errors.Is(err, &exec.ExitError{}) {
			return &result{
				success: false,
				output:  string(output),
			}, nil
		} else {
			return nil, err
		}
	}
	return &result{
		success: true,
		output:  string(output),
	}, nil
}
