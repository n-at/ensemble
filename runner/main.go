package runner

import (
	"bytes"
	"ensemble/storage"
	"ensemble/storage/structures"
	"errors"
	"fmt"
	"github.com/alessio/shellescape"
	log "github.com/sirupsen/logrus"
	"os"
	"os/exec"
	"strings"
	"syscall"
	"time"
)

type Runner struct {
	config    Configuration
	store     *storage.Storage
	processes map[string]*exec.Cmd
}

type Configuration struct {
	Path     string
	AuthSock string
}

///////////////////////////////////////////////////////////////////////////////

func New(config Configuration, store *storage.Storage) *Runner {
	return &Runner{
		config:    config,
		store:     store,
		processes: make(map[string]*exec.Cmd),
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

	var vaultPasswordFile *os.File
	if project.VariablesVault {
		var err error
		vaultPasswordFile, err = os.CreateTemp("", storage.NewId())
		if err != nil {
			return nil, err
		}
		if err := os.WriteFile(vaultPasswordFile.Name(), []byte(project.VaultPassword), 0600); err != nil {
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
			if project.VariablesVault {
				if err := os.Remove(vaultPasswordFile.Name()); err != nil {
					log.Warnf("vault password file remove error %s: %s", run.Id, err)
				}
			}
		}()

		stdout, stderr, err := r.executePlaybook(run.Id, project, playbook, mode, vaultPasswordFile)
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

func (r *Runner) TerminatePlaybook(runId string) error {
	cmd, ok := r.processes[runId]
	if !ok {
		return errors.New("playbook run process not found")
	}

	if err := cmd.Process.Signal(syscall.SIGINT); err != nil {
		return err
	}

	return nil
}

///////////////////////////////////////////////////////////////////////////////

func (r *Runner) installCollection(name string) error {
	command := fmt.Sprintf("ansible-galaxy collection install %s", shellescape.Quote(name))
	cmd := exec.Command("/bin/bash", "-c", command)
	output, err := cmd.CombinedOutput()
	log.Infof("collection %s installation log:\n%s", name, output)
	return err
}

func (r *Runner) executePlaybook(runId string, project *structures.Project, playbook *structures.Playbook, mode int, vaultPasswordFile *os.File) (string, string, error) {
	command := strings.Builder{}
	command.WriteString("ansible-playbook")

	if mode == structures.PlaybookRunModeCheck {
		command.WriteString(" --check --diff")
	}

	inventory := fmt.Sprintf("inventories/%s", project.Inventory)
	command.WriteString(fmt.Sprintf(" --inventory %s", shellescape.Quote(inventory)))

	if project.VariablesVault {
		command.WriteString(fmt.Sprintf(" --extra-vars %s", shellescape.Quote("@vars/vault.yml")))
		command.WriteString(fmt.Sprintf(" --vault-password-file %s", shellescape.Quote(vaultPasswordFile.Name())))
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

	cmd := exec.Command("/bin/bash", "-c", command.String())
	cmd.Dir = fmt.Sprintf("%s/%s", r.config.Path, project.Id)
	cmd.Env = append(cmd.Env, "ANSIBLE_STDOUT_CALLBACK=ansible.posix.json")
	r.sshAuthSock(cmd)

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	r.processes[runId] = cmd

	err := cmd.Run()

	delete(r.processes, runId)

	return stdout.String(), stderr.String(), err
}

func (r *Runner) sshAuthSock(cmd *exec.Cmd) {
	sock := ""

	if len(r.config.AuthSock) != 0 {
		sock = r.config.AuthSock
	} else if len(os.Getenv("SSH_AUTH_SOCK")) != 0 {
		sock = os.Getenv("SSH_AUTH_SOCK")
	}

	if len(sock) != 0 {
		sock = fmt.Sprintf("SSH_AUTH_SOCK=%s", shellescape.Quote(sock))
		cmd.Env = append(cmd.Env, sock)
	}
}
