package runner

import (
	"bytes"
	"ensemble/storage"
	"ensemble/storage/structures"
	"fmt"
	"github.com/alessio/shellescape"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
	"time"
)

type Runner struct {
	config Configuration
	store  *storage.Storage
}

type Configuration struct {
	Path string
}

///////////////////////////////////////////////////////////////////////////////

func New(config Configuration, store *storage.Storage) *Runner {
	return &Runner{
		config: config,
		store:  store,
	}
}

///////////////////////////////////////////////////////////////////////////////

func (r *Runner) Run(project *structures.Project, playbook *structures.Playbook, mode int, userId string) (*structures.PlaybookRun, error) {
	if err := r.installCollection("ansible.posix"); err != nil {
		return nil, err
	}
	for _, collection := range project.CollectionsList {
		if len(strings.TrimSpace(collection)) == 0 {
			continue
		}
		if err := r.installCollection(collection); err != nil {
			return nil, err
		}
	}
	if project.VariablesVault {
		vaultPasswordFilePath := fmt.Sprintf("%s/%s/__vault_password__", r.config.Path, project.Id)
		if err := os.WriteFile(vaultPasswordFilePath, []byte(project.VaultPassword), 0666); err != nil {
			return nil, err
		}
	}

	run := structures.PlaybookRun{
		PlaybookId:    playbook.Id,
		UserId:        userId,
		Mode:          mode,
		StartTime:     time.Now(),
		Result:        structures.PlaybookRunResultRunning,
		InventoryFile: project.Inventory,
		VariablesFile: project.Variables,
	}
	if err := r.store.PlaybookRunInsert(&run); err != nil {
		return nil, err
	}
	if err := r.store.PlaybookLock(playbook.Id, true); err != nil {
		return nil, err
	}

	go func() {
		defer func() {
			if err := r.store.PlaybookLock(playbook.Id, false); err != nil {
				log.Warnf("playbook %s unlock failed: %s", playbook.Id, err)
			}
		}()

		stdout, stderr, err := r.executePlaybook(project, playbook, mode)
		if err != nil {
			log.Warnf("playbook run %s failed: %s", run.Id, err)
			run.Result = structures.PlaybookRunResultFailure
		} else {
			run.Result = structures.PlaybookRunResultSuccess
		}
		run.FinishTime = time.Now()

		if err := r.store.PlaybookRunUpdate(&run); err != nil {
			log.Warnf("playbook run %s update failed: %s", run.Id, err)
			return
		}

		runResult := structures.RunResult{
			Id:     run.Id,
			RunId:  run.Id,
			Output: stdout,
			Error:  stderr,
		}
		if err := r.store.RunResultInsert(&runResult); err != nil {
			log.Warnf("playbook run result %s insert failed: %s", runResult.Id, err)
		}
	}()

	return &run, nil
}

///////////////////////////////////////////////////////////////////////////////

func (r *Runner) installCollection(name string) error {
	command := fmt.Sprintf("ansible-galaxy collection install %s", shellescape.Quote(name))
	cmd := exec.Command("sh", "-c", command)
	return cmd.Run()
}

func (r *Runner) executePlaybook(project *structures.Project, playbook *structures.Playbook, mode int) (string, string, error) {
	command := strings.Builder{}
	command.WriteString("ansible-playbook")

	if mode == structures.PlaybookRunModeCheck {
		command.WriteString(" --check --diff")
	}

	inventory := fmt.Sprintf("inventories/%s", project.Inventory)
	command.WriteString(fmt.Sprintf(" --inventory %s", shellescape.Quote(inventory)))

	if project.VariablesVault {
		command.WriteString(fmt.Sprintf(" --extra-vars %s", shellescape.Quote("@vars/vault.yml")))
		command.WriteString(fmt.Sprintf(" --vault-password-file __vault_password__"))
	}
	if project.VariablesMain {
		command.WriteString(fmt.Sprintf(" --extra-vars %s", shellescape.Quote("@vars/main.yml")))
	}
	if len(project.Variables) != 0 {
		variables := fmt.Sprintf("@vars/%s", project.Variables)
		command.WriteString(fmt.Sprintf(" --extra-vars %s", shellescape.Quote(variables)))
	}

	command.WriteString(" ")
	command.WriteString(shellescape.Quote(playbook.Filename))

	cmd := exec.Command("sh", "-c", command.String())
	cmd.Dir = fmt.Sprintf("%s/%s", r.config.Path, project.Id)
	cmd.Env = append(cmd.Env, "ANSIBLE_STDOUT_CALLBACK=ansible.posix.json")

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	err := cmd.Run()

	return stdout.String(), stderr.String(), err
}
