package repository

import (
	"ensemble/storage"
	"ensemble/storage/structures"
	"errors"
	"fmt"
	"github.com/alessio/shellescape"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"os/exec"
	"regexp"
	"sort"
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
		log.Infof("updating project %s...", project.Id)
		if err := m.Update(project); err != nil {
			log.Warnf("unable to update project %s: %s", project.Name, err)
		}
	}
}

func (m *Manager) Update(project *structures.Project) error {
	if m.store.ProjectHasLockedPlaybooks(project.Id) {
		return errors.New("project has locked playbooks")
	}

	output := strings.Builder{}
	success := false
	revision := "unknown revision"

	defer func() {
		update := structures.ProjectUpdate{
			ProjectId: project.Id,
			Date:      time.Now(),
			Success:   success,
			Revision:  revision,
			Log:       output.String(),
		}
		if err := m.store.ProjectUpdateInsert(&update); err != nil {
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

	if err := m.updateProjectInfo(project); err != nil {
		return err
	}
	if err := m.updatePlaybooksInfo(project); err != nil {
		return err
	}

	success = true

	return nil
}

///////////////////////////////////////////////////////////////////////////////

func (m *Manager) projectDirectory(p *structures.Project) string {
	return fmt.Sprintf("%s/%s", m.config.Path, p.Id)
}

func (m *Manager) projectDirectoryExists(p *structures.Project) (bool, error) {
	return directoryExists(m.projectDirectory(p))
}

func (m *Manager) projectRepositoryExists(p *structures.Project) (bool, error) {
	repoDir := fmt.Sprintf("%s/.git", m.projectDirectory(p))
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

func (m *Manager) updateProjectInfo(p *structures.Project) error {
	projectDirectory := m.projectDirectory(p)

	inventoryMainFileName := fmt.Sprintf("%s/inventories/main.yml", projectDirectory)
	exists, err := fileExists(inventoryMainFileName)
	if err != nil {
		return err
	}
	if !exists {
		return errors.New("main inventory file not found")
	}

	variablesMainFileName := fmt.Sprintf("%s/vars/main.yml", projectDirectory)
	exists, err = fileExists(variablesMainFileName)
	if err != nil {
		return err
	}
	p.VariablesMain = exists

	vaultFileName := fmt.Sprintf("%s/vars/vault.yml", projectDirectory)
	exists, err = fileExists(vaultFileName)
	if err != nil {
		return err
	}
	p.VariablesVault = exists

	inventories, err := m.projectInventories(p)
	if err != nil {
		return err
	}
	p.InventoryList = inventories

	inventoryFound := false
	for _, i := range inventories {
		if i == p.Inventory {
			inventoryFound = true
			break
		}
	}
	if !inventoryFound {
		p.Inventory = "main.yml"
	}

	variables, err := m.projectVariables(p)
	if err != nil {
		return err
	}
	p.VariablesList = variables

	variablesFound := false
	for _, v := range variables {
		if v == p.Variables {
			variablesFound = true
			break
		}
	}
	if !variablesFound {
		p.Variables = ""
	}

	collections, err := m.projectCollections(p)
	if err != nil {
		return err
	}
	p.CollectionsList = collections

	if err := m.store.ProjectUpdate(p); err != nil {
		return err
	}

	return nil
}

func (m *Manager) projectCollections(p *structures.Project) ([]string, error) {
	collectionsFile := fmt.Sprintf("%s/collections.txt", m.projectDirectory(p))
	exists, err := fileExists(collectionsFile)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []string{}, nil
	}

	bytes, err := os.ReadFile(collectionsFile)
	if err != nil {
		return nil, err
	}
	return strings.Split(string(bytes), "\n"), nil
}

func (m *Manager) projectVariables(p *structures.Project) ([]string, error) {
	variablesDirectory := fmt.Sprintf("%s/vars", m.projectDirectory(p))
	exists, err := directoryExists(variablesDirectory)
	if err != nil {
		return nil, err
	}
	if !exists {
		return []string{}, nil
	}

	pattern, err := regexp.Compile("^.+\\.yml$")
	if err != nil {
		return nil, err
	}

	entries, err := ioutil.ReadDir(variablesDirectory)
	if err != nil {
		return nil, err
	}

	var variables []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if pattern.MatchString(name) && name != "main.yml" && name != "vault.yml" {
			variables = append(variables, name)
		}
	}

	sort.Strings(variables)

	return variables, nil
}

func (m *Manager) projectInventories(p *structures.Project) ([]string, error) {
	inventoriesDirectory := fmt.Sprintf("%s/inventories", m.projectDirectory(p))
	exists, err := directoryExists(inventoriesDirectory)
	if err != nil {
		return nil, err
	}
	if !exists {
		return nil, errors.New("inventories directory not found")
	}

	entries, err := ioutil.ReadDir(inventoriesDirectory)
	if err != nil {
		return nil, err
	}

	var inventories []string

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		inventories = append(inventories, entry.Name())
	}

	sort.Strings(inventories)

	return inventories, nil
}

///////////////////////////////////////////////////////////////////////////////

func (m *Manager) updatePlaybooksInfo(p *structures.Project) error {
	playbooks, err := m.store.PlaybookGetByProject(p.Id)
	if err != nil {
		return err
	}

	currentPlaybooks, err := m.playbooks(p)
	if err != nil {
		return err
	}

	var playbooksToDelete map[string]bool
	var playbooksByFileName map[string]string
	for _, playbook := range playbooks {
		playbooksToDelete[playbook.Id] = true
		playbooksByFileName[playbook.Filename] = playbook.Id
	}

	var playbooksToUpdate []*structures.Playbook
	var playbooksToInsert []*structures.Playbook

	for _, playbook := range currentPlaybooks {
		id, found := playbooksByFileName[playbook.Filename]
		if found {
			playbook.Id = id
			playbooksToUpdate = append(playbooksToUpdate, playbook)
			playbooksToDelete[id] = false
		} else {
			playbooksToInsert = append(playbooksToInsert, playbook)
		}
	}

	for _, playbook := range playbooksToInsert {
		if err := m.store.PlaybookInsert(playbook); err != nil {
			log.Errorf("unable to insert playbook %s: %s", playbook.Filename, err)
		}
	}
	for _, playbook := range playbooksToUpdate {
		if err := m.store.PlaybookUpdate(playbook); err != nil {
			log.Errorf("unable to update playbook %s: %s", playbook.Id, err)
		}
	}
	for playbookId, toDelete := range playbooksToDelete {
		if !toDelete {
			continue
		}
		if err := m.store.PlaybookDelete(playbookId); err != nil {
			log.Errorf("unable to delete playbook %s: %s", playbookId, err)
		}
	}

	return nil
}

func (m *Manager) playbooks(p *structures.Project) ([]*structures.Playbook, error) {
	pattern, err := regexp.Compile("^.+\\.yml$")
	if err != nil {
		return nil, err
	}

	projectDirectory := m.projectDirectory(p)

	entries, err := ioutil.ReadDir(projectDirectory)
	if err != nil {
		return nil, err
	}

	var playbooks []*structures.Playbook

	for _, entry := range entries {
		if entry.IsDir() || !pattern.MatchString(entry.Name()) {
			continue
		}
		path := fmt.Sprintf("%s/%s", projectDirectory, entry.Name())
		name, description, err := m.playbookInfo(path)
		if err != nil {
			log.Errorf("unable to read %s: %s", path, err)
			continue
		}
		playbook := structures.Playbook{
			ProjectId:   p.Id,
			Filename:    entry.Name(),
			Name:        name,
			Description: description,
			Locked:      false,
		}
		playbooks = append(playbooks, &playbook)
	}

	return playbooks, nil
}

func (m *Manager) playbookInfo(filePath string) (string, string, error) {
	content, err := os.ReadFile(filePath)
	if err != nil {
		return "", "", err
	}

	name := ""
	description := strings.Builder{}

	lines := strings.Split(string(content), "\n")

	for _, line := range lines {
		if line == "---" {
			break
		}
		if line[:1] != "#" {
			continue
		}
		if len(name) == 0 {
			name = line
		} else {
			description.WriteString(line)
			description.WriteString("\n")
		}
	}

	return name, description.String(), nil
}

///////////////////////////////////////////////////////////////////////////////

func directoryExists(dir string) (bool, error) {
	stat, err := os.Stat(dir)
	if err == nil {
		return stat.IsDir(), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, err
}

func fileExists(file string) (bool, error) {
	stat, err := os.Stat(file)
	if err == nil {
		return !stat.IsDir(), nil
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
