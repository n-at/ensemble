package storage

import (
	"ensemble/storage/structures"
	"errors"
	"github.com/jmoiron/sqlx"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"time"
)

type Configuration struct {
	Url    string
	Secret string
}

type Storage struct {
	config Configuration
	db     *sqlx.DB
}

type Scanner interface {
	Scan(dest ...any) error
}

///////////////////////////////////////////////////////////////////////////////

func New(configuration Configuration) (*Storage, error) {
	db, err := sqlx.Open("postgres", configuration.Url)
	if err != nil {
		return nil, err
	}

	migrator, err := NewMigrator(db)
	if err != nil {
		return nil, err
	}
	if err := migrator.migrate(); err != nil {
		return nil, err
	}

	return &Storage{
		config: configuration,
		db:     db,
	}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func (s *Storage) queryExists(query string, args ...any) bool {
	count := 0

	if err := s.db.Get(&count, query, args...); err != nil {
		log.Warnf("exists error: %s", err)
	}

	return count > 0
}

///////////////////////////////////////////////////////////////////////////////
//Session
///////////////////////////////////////////////////////////////////////////////

func (s *Storage) SessionCreate(userId string) (*structures.Session, error) {
	session := structures.Session{
		Id:      NewId(),
		UserId:  userId,
		Created: time.Now(),
	}

	query := `insert into sessions (id, user_id, created) values (:id, :user_id, :created)`
	if _, err := s.db.NamedExec(query, session); err != nil {
		return nil, err
	}
	return &session, nil
}

func (s *Storage) SessionGet(id string) (*structures.Session, error) {
	var session structures.Session
	query := `select id, user_id, created from sessions where id = $1`
	if err := s.db.Get(&session, query, id); err != nil {
		return nil, err
	}
	return &session, nil
}

///////////////////////////////////////////////////////////////////////////////
//User
///////////////////////////////////////////////////////////////////////////////

func (s *Storage) UserAnyExists() bool {
	return s.queryExists(`select count(1) from users`)
}

func (s *Storage) UserExists(id string) bool {
	query := `select count(1) 
	          from users 
	          where id = $1 
	            and not coalesce(deleted, false)`
	return s.queryExists(query, id)
}

func (s *Storage) UserExistsByLogin(login string) bool {
	query := `select count(1) 
              from users 
              where login = $1 
                and not coalesce(deleted, false)`
	return s.queryExists(query, login)
}

func (s *Storage) UserGet(id string) (*structures.User, error) {
	query := `select id, login, password, role 
              from users 
              where id = $1 
                and not coalesce(deleted, false)`
	var user structures.User
	if err := s.db.Get(&user, query, id); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Storage) UserGetByLogin(login string) (*structures.User, error) {
	query := `select id, login, password, role 
              from users 
              where login = $1 
                and not coalesce(deleted, false)
	          limit 1`
	var user structures.User
	if err := s.db.Get(&user, query, login); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Storage) UserGetAll() ([]*structures.User, error) {
	query := `select id, login, password, role 
              from users 
              where not coalesce(deleted, false) 
              order by login`
	var users []*structures.User
	if err := s.db.Select(&users, query); err != nil {
		return nil, err
	}
	return users, nil
}

func (s *Storage) UserInsert(user *structures.User) error {
	if user == nil {
		return errors.New("insert nil user")
	}
	if len(user.Login) == 0 {
		return errors.New("insert user without login")
	}
	if s.UserExistsByLogin(user.Login) {
		return errors.New("insert user with duplicate login")
	}
	if len(user.Password) == 0 {
		return errors.New("insert user without password")
	}
	if len(user.Id) == 0 {
		user.Id = NewId()
	}
	if user.Role == 0 {
		user.Role = structures.UserRoleOperator
	}

	query := `insert into users (id, login, password, role) 
              values (:id, :login, :password, :role)`
	_, err := s.db.NamedExec(query, user)
	return err
}

func (s *Storage) UserUpdate(user *structures.User) error {
	if user == nil {
		return errors.New("update nil user")
	}
	if len(user.Id) == 0 {
		return errors.New("update user without id")
	}
	if len(user.Login) == 0 {
		return errors.New("update user without login")
	}
	if len(user.Password) == 0 {
		return errors.New("update user without password")
	}

	existingUser, err := s.UserGet(user.Id)
	if err != nil {
		return err
	}
	if existingUser == nil {
		return errors.New("update user existing not found")
	}
	if user.Login != existingUser.Login && s.UserExistsByLogin(user.Login) {
		return errors.New("update user new login exists")
	}

	query := `update users 
              set login = :login, password = :password, role = :role, deleted = false 
              where id = :id`
	_, err = s.db.NamedExec(query, user)
	return err
}

func (s *Storage) UserDelete(id string) error {
	query := `update users set deleted=true where id = $1`
	_, err := s.db.Exec(query, id)
	return err
}

func (s *Storage) UserEnsureAdminExists() error {
	if s.UserAnyExists() {
		return nil
	}

	adminPassword := GenerateRandomString(12)
	adminPasswordEncrypted, err := EncryptPassword(adminPassword)
	if err != nil {
		return err
	}

	admin := &structures.User{
		Login:    "admin",
		Password: adminPasswordEncrypted,
		Role:     structures.UserRoleAdmin,
	}
	if err := s.UserInsert(admin); err != nil {
		return err
	}

	log.Infof("CREATED ADMIN, login: %s, password: %s", admin.Login, adminPassword)

	return nil
}

///////////////////////////////////////////////////////////////////////////////
//Project
///////////////////////////////////////////////////////////////////////////////

func (s *Storage) projectDecrypt(p *structures.Project) (*structures.Project, error) {
	repositoryPasswordDecrypted, err := DecryptString(s.config.Secret, p.RepositoryPassword)
	if err != nil {
		return nil, err
	}

	vaultPasswordDecrypted, err := DecryptString(s.config.Secret, p.VaultPassword)
	if err != nil {
		return nil, err
	}

	p.RepositoryPassword = repositoryPasswordDecrypted
	p.VaultPassword = vaultPasswordDecrypted
	return p, nil
}

func (s *Storage) projectDecryptAll(projects []*structures.Project) []*structures.Project {
	var decryptedProjects []*structures.Project

	for _, project := range projects {
		decrypted, err := s.projectDecrypt(project)
		if err != nil {
			log.Warnf("unable to decrypt project: %s", err)
			continue
		}
		decryptedProjects = append(decryptedProjects, decrypted)
	}

	return decryptedProjects
}

func (s *Storage) projectEncrypt(p *structures.Project) (*structures.Project, error) {
	repositoryPasswordEncrypted, err := EncryptString(s.config.Secret, p.RepositoryPassword)
	if err != nil {
		return nil, err
	}

	vaultPasswordEncrypted, err := EncryptString(s.config.Secret, p.VaultPassword)
	if err != nil {
		return nil, err
	}

	p.RepositoryPassword = repositoryPasswordEncrypted
	p.VaultPassword = vaultPasswordEncrypted

	return p, nil
}

func (s *Storage) ProjectExists(id string) bool {
	query := `select count(1) 
              from projects 
              where id = $1 
                and not coalesce(deleted, false)`
	return s.queryExists(query, id)
}

func (s *Storage) ProjectExistsByName(name string) bool {
	query := `select count(1) 
              from projects 
              where name = $1 
                and not coalesce(deleted, false)`
	return s.queryExists(query, name)
}

func (s *Storage) ProjectGet(id string) (*structures.Project, error) {
	query := `select id, name, description, 
                     repo_url, repo_login, repo_password, repo_branch, 
                     inventory, inventory_list, 
                     collections_list,
                     variables, variables_list, variables_main, variables_vault,
                     vault_password
              from projects
              where id = $1 
                and not coalesce(deleted, false)`

	var project structures.Project
	if err := s.db.Get(&project, query, id); err != nil {
		return nil, err
	}
	return s.projectDecrypt(&project)
}

func (s *Storage) ProjectGetAll() ([]*structures.Project, error) {
	query := `select id, name, description, 
                     repo_url, repo_login, repo_password, repo_branch, 
                     inventory, inventory_list, 
                     collections_list,
                     variables, variables_list, variables_main, variables_vault,
                     vault_password
              from projects
              where not coalesce(deleted, false)
              order by name`

	var projects []*structures.Project
	if err := s.db.Select(&projects, query); err != nil {
		return nil, err
	}
	return s.projectDecryptAll(projects), nil
}

func (s *Storage) ProjectGetByUser(userId string) ([]*structures.Project, error) {
	query := `select id, name, description, 
                     repo_url, repo_login, repo_password, repo_branch, 
                     inventory, inventory_list, 
                     collections_list,
                     variables, variables_list, variables_main, variables_vault,
                     vault_password
              from projects 
                left join projects_users_access on (projects_users_access.project_id = projects.id) 
              where not coalesce(deleted, false) 
                and projects_users_access.user_id = $1
              order by name`

	var projects []*structures.Project
	if err := s.db.Select(&projects, query, userId); err != nil {
		return nil, err
	}
	return s.projectDecryptAll(projects), nil
}

func (s *Storage) ProjectInsert(project *structures.Project) error {
	if project == nil {
		return errors.New("project insert nil")
	}
	if len(project.Name) == 0 {
		return errors.New("project insert empty name")
	}
	if len(project.RepositoryUrl) == 0 {
		return errors.New("project insert empty repository url")
	}
	if s.ProjectExistsByName(project.Name) {
		return errors.New("project insert name exists")
	}
	if len(project.Id) == 0 {
		project.Id = NewId()
	}
	if len(project.RepositoryBranch) == 0 {
		project.RepositoryBranch = structures.ProjectDefaultBranchName
	}

	query := `insert into projects (id, name, description, 
							  repo_url, repo_login, repo_password, repo_branch, 
							  inventory,  inventory_list, 
							  collections_list,
							  variables, variables_list, variables_main, variables_vault,
							  vault_password) 
			   values (:id, :name, :description, 
					   :repo_url, :repo_login, :repo_password, :repo_branch, 
					   :inventory, :inventory_list, 
					   :collections_list,
					   :variables, :variables_list, :variables_main, :variables_vault,
					   :vault_password)`

	projectToSave := *project
	if _, err := s.projectEncrypt(&projectToSave); err != nil {
		return err
	}
	if _, err := s.db.NamedExec(query, projectToSave); err != nil {
		return err
	}
	return nil
}

func (s *Storage) ProjectUpdate(project *structures.Project) error {
	if project == nil {
		return errors.New("project update nil")
	}
	if len(project.Id) == 0 {
		return errors.New("project update empty id")
	}
	if len(project.Name) == 0 {
		return errors.New("project update empty name")
	}
	if len(project.RepositoryUrl) == 0 {
		return errors.New("project update empty repository url")
	}

	existingProject, err := s.ProjectGet(project.Id)
	if err != nil {
		return err
	}
	if existingProject == nil {
		return errors.New("project update project does not exist")
	}
	if existingProject.Name != project.Name && s.ProjectExistsByName(project.Name) {
		return errors.New("project update name exists")
	}

	query := `
		update projects set 
			name = :name, description = :description, 
			repo_url = :repo_url, repo_login = :repo_login, repo_password = :repo_password, repo_branch = :repo_branch,
			inventory = :inventory, inventory_list = :inventory_list,
			collections_list = :collections_list,
			variables = :variables, variables_list = :variables_list, variables_main = :variables_main, variables_vault = :variables_vault,
			vault_password = :vault_password, 
			deleted = false
		where id = :id`

	projectToSave := *project
	if _, err := s.projectEncrypt(&projectToSave); err != nil {
		return err
	}
	if _, err := s.db.NamedExec(query, projectToSave); err != nil {
		return err
	}
	return nil
}

func (s *Storage) ProjectDelete(id string) error {
	query := `update projects set deleted = true where id = $1`
	_, err := s.db.Exec(query, id)
	return err
}

func (s *Storage) ProjectHasLockedPlaybooks(id string) bool {
	query := `
		select count(1)
		from playbooks
		where project_id = $1 
		  and not coalesce(deleted, false) 
		  and locked`
	return s.queryExists(query, id)
}

///////////////////////////////////////////////////////////////////////////////
//Project User Access
///////////////////////////////////////////////////////////////////////////////

func (s *Storage) ProjectUserAccessExists(projectId, userId string) bool {
	query := `select count(1) 
              from projects_users_access 
              where project_id = $1 
                and user_id = $2`
	return s.queryExists(query, projectId, userId)
}

func (s *Storage) ProjectUserAccessCreate(projectId, userId string) error {
	if len(projectId) == 0 {
		return errors.New("project user access empty project id")
	}
	if len(userId) == 0 {
		return errors.New("project user access empty user id")
	}

	query := `insert into projects_users_access (project_id, user_id) values ($1, $2)`
	_, err := s.db.Exec(query, projectId, userId)
	return err
}

func (s *Storage) ProjectUserAccessDelete(projectId, userId string) error {
	query := `delete from projects_users_access where project_id = $1 and user_id = $2`
	_, err := s.db.Exec(query, projectId, userId)
	return err
}

///////////////////////////////////////////////////////////////////////////////
//Project Updates
///////////////////////////////////////////////////////////////////////////////

func (s *Storage) ProjectUpdateGet(id string) (*structures.ProjectUpdate, error) {
	query := `select id, project_id, date, success, revision, log 
              from project_updates 
              where id = $1 
                and not coalesce(deleted, false)`

	var update structures.ProjectUpdate
	if err := s.db.Get(&update, query, id); err != nil {
		return nil, err
	}
	return &update, nil
}

func (s *Storage) ProjectUpdateGetByProject(projectId string) ([]*structures.ProjectUpdate, error) {
	query := `select id, project_id, date, success, revision,  log 
              from project_updates 
              where project_id = $1 
                and not coalesce(deleted, false)
		      order by date desc`

	var updates []*structures.ProjectUpdate
	if err := s.db.Select(&updates, query, projectId); err != nil {
		return nil, err
	}
	return updates, nil
}

func (s *Storage) ProjectUpdateGetProjectLatest(projectId string) (*structures.ProjectUpdate, error) {
	query := `select id, project_id, date, success, revision, log 
              from project_updates 
              where project_id = $1 
                and not coalesce(deleted, false)
		      order by date desc
			  limit 1`

	var update structures.ProjectUpdate
	if err := s.db.Get(&update, query, projectId); err != nil {
		return nil, err
	}
	return &update, nil
}

func (s *Storage) ProjectUpdateInsert(update *structures.ProjectUpdate) error {
	if update == nil {
		return errors.New("project update insert nil")
	}
	if len(update.ProjectId) == 0 {
		return errors.New("project update insert empty project id")
	}
	if len(update.Id) == 0 {
		update.Id = NewId()
	}

	query := `insert into project_updates (id, project_id, date, success, revision, log) 
              values (:id, :project_id, :date, :success, :revision, :log)`
	_, err := s.db.NamedExec(query, update)
	return err
}

func (s *Storage) ProjectUpdateDelete(id string) error {
	query := `update project_updates set deleted = true where id = $1`
	_, err := s.db.Exec(query, id)
	return err
}

///////////////////////////////////////////////////////////////////////////////
//Playbooks
///////////////////////////////////////////////////////////////////////////////

func (s *Storage) PlaybookGet(id string) (*structures.Playbook, error) {
	query := `select id, project_id, filename, name, description, locked 
              from playbooks 
              where id = $1 
                and not coalesce(deleted, false)`

	var playbook structures.Playbook
	if err := s.db.Get(&playbook, query, id); err != nil {
		return nil, err
	}
	return &playbook, nil
}

func (s *Storage) PlaybookGetByProject(projectId string) ([]*structures.Playbook, error) {
	query := `select id, project_id, filename, name, description, locked 
              from playbooks 
              where project_id = $1 
                and not coalesce(deleted, false)
              order by name, filename`

	var playbooks []*structures.Playbook
	if err := s.db.Select(&playbooks, query, projectId); err != nil {
		return nil, err
	}
	return playbooks, nil
}

func (s *Storage) PlaybookInsert(playbook *structures.Playbook) error {
	if playbook == nil {
		return errors.New("playbook insert nil")
	}
	if len(playbook.ProjectId) == 0 {
		return errors.New("playbook insert empty project id")
	}
	if len(playbook.Filename) == 0 {
		return errors.New("playbook insert empty file name")
	}
	if len(playbook.Id) == 0 {
		playbook.Id = NewId()
	}

	query := `insert into playbooks (id, project_id, filename, name, description, locked) 
              values (:id, :project_id, :filename, :name, :description, :locked)`
	_, err := s.db.NamedExec(query, playbook)
	return err
}

func (s *Storage) PlaybookUpdate(playbook *structures.Playbook) error {
	if playbook == nil {
		return errors.New("playbook update nil")
	}
	if len(playbook.Id) == 0 {
		return errors.New("playbook update empty id")
	}
	if len(playbook.ProjectId) == 0 {
		return errors.New("playbook update empty project id")
	}
	if len(playbook.Filename) == 0 {
		return errors.New("playbook update empty file name")
	}

	existingPlaybook, err := s.PlaybookGet(playbook.Id)
	if err != nil {
		return err
	}
	if existingPlaybook == nil {
		return errors.New("playbook update existing not found")
	}
	if existingPlaybook.ProjectId != playbook.ProjectId {
		return errors.New("playbook update cannot change project id")
	}

	query := `update playbooks 
              set filename = :filename, name = :name, description = :description, locked = :locked, deleted = false 
              where id = :id`
	_, err = s.db.NamedExec(query, playbook)
	return err
}

func (s *Storage) PlaybookDelete(id string) error {
	query := `update playbooks set deleted=true where id=$1`
	_, err := s.db.Exec(query, id)
	return err
}

func (s *Storage) PlaybookLock(id string, value bool) error {
	query := `update playbooks set locked = $1 where id = $2`
	_, err := s.db.Exec(query, value, id)
	return err
}

///////////////////////////////////////////////////////////////////////////////
//Playbook Runs
///////////////////////////////////////////////////////////////////////////////

func (s *Storage) PlaybookRunGet(id string) (*structures.PlaybookRun, error) {
	query := `select id, playbook_id, user_id, mode, start_time, finish_time, result, inventory_file, variables_file
              from playbook_runs 
              where id = $1 
                and not coalesce(deleted, false)`

	var run structures.PlaybookRun
	if err := s.db.Get(&run, query, id); err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *Storage) PlaybookRunGetLatest(playbookId string) (*structures.PlaybookRun, error) {
	query := `select id, playbook_id, user_id, mode, start_time, finish_time, result, inventory_file, variables_file
              from playbook_runs 
              where playbook_id = $1 
                and not coalesce(deleted, false)
              order by start_time desc
              limit 1`

	var run structures.PlaybookRun
	if err := s.db.Get(&run, query, playbookId); err != nil {
		return nil, err
	}
	return &run, nil
}

func (s *Storage) PlaybookRunGetByPlaybook(playbookId string) ([]*structures.PlaybookRun, error) {
	query := `select id, playbook_id, user_id, mode, start_time, finish_time, result, inventory_file, variables_file
              from playbook_runs 
              where playbook_id = $1 
                and not coalesce(deleted, false)
              order by start_time desc`

	var runs []*structures.PlaybookRun
	if err := s.db.Select(&runs, query, playbookId); err != nil {
		return nil, err
	}
	return runs, nil
}

func (s *Storage) PlaybookRunInsert(run *structures.PlaybookRun) error {
	if run == nil {
		return errors.New("playbook run insert nil")
	}
	if len(run.PlaybookId) == 0 {
		return errors.New("playbook run insert empty playbook id")
	}
	if len(run.UserId) == 0 {
		return errors.New("playbook run insert empty user id")
	}
	if len(run.Id) == 0 {
		run.Id = NewId()
	}

	query := `insert into playbook_runs (id, playbook_id, user_id, mode, start_time, finish_time, result, inventory_file, variables_file)
              values (:id, :playbook_id, :user_id, :mode, :start_time, :finish_time, :result, :inventory_file, :variables_file)`
	_, err := s.db.NamedExec(query, run)
	return err
}

func (s *Storage) PlaybookRunUpdate(run *structures.PlaybookRun) error {
	if run == nil {
		return errors.New("playbook run update nil")
	}
	if len(run.Id) == 0 {
		return errors.New("playbook run update empty id")
	}
	if len(run.PlaybookId) == 0 {
		return errors.New("playbook run update empty playbook id")
	}
	if len(run.UserId) == 0 {
		return errors.New("playbook run update empty user id")
	}

	existingRun, err := s.PlaybookRunGet(run.Id)
	if err != nil {
		return err
	}
	if existingRun == nil {
		return errors.New("playbook run update existing run not found")
	}
	if existingRun.PlaybookId != run.PlaybookId {
		return errors.New("playbook run update cannot change playbook id")
	}
	if existingRun.UserId != run.UserId {
		return errors.New("playbook run update cannot change user id")
	}

	query := `update playbook_runs 
              set mode = :mode, start_time = :start_time, finish_time = :finish_time, result = :result, 
                  inventory_file = :inventory_file, variables_file = :variables_file, deleted = false
              where id = :id`
	_, err = s.db.NamedExec(query, run)
	return err
}

func (s *Storage) PlaybookRunDelete(id string) error {
	query := `update playbook_runs set deleted = true where id = $1`
	_, err := s.db.Exec(query, id)
	return err
}

///////////////////////////////////////////////////////////////////////////////
//Run Results
///////////////////////////////////////////////////////////////////////////////

func (s *Storage) RunResultGet(id string) (*structures.RunResult, error) {
	query := `select id, run_id, output, error
              from run_results 
              where id = $1 
                and not coalesce(deleted, false)`

	var result structures.RunResult
	if err := s.db.Get(&result, query, id); err != nil {
		return nil, err
	}
	return &result, nil
}

func (s *Storage) RunResultInsert(result *structures.RunResult) error {
	if result == nil {
		return errors.New("run result insert nil")
	}
	if len(result.RunId) == 0 {
		return errors.New("run result insert empty run id")
	}
	if len(result.Id) == 0 {
		result.Id = NewId()
	}

	query := `insert into run_results (id, run_id, output, error) values (:id, :run_id, :output, :error)`
	_, err := s.db.NamedExec(query, result)
	return err
}

func (s *Storage) RunResultUpdate(result *structures.RunResult) error {
	if result == nil {
		return errors.New("run result update nil")
	}
	if len(result.Id) == 0 {
		return errors.New("run result update empty id")
	}
	if len(result.RunId) == 0 {
		return errors.New("run result update empty run id")
	}

	existingResult, err := s.RunResultGet(result.Id)
	if err != nil {
		return err
	}
	if existingResult == nil {
		return errors.New("run result update existing result not found")
	}
	if existingResult.RunId != result.RunId {
		return errors.New("run result update cannot change run id")
	}

	query := `update run_results 
              set output = :output, error = :error, deleted = false 
              where id = :id`
	_, err = s.db.NamedExec(query, result)
	return err
}

func (s *Storage) RunResultDelete(id string) error {
	query := `update run_results set deleted = true where id = $1`
	_, err := s.db.Exec(query, id)
	return err
}

///////////////////////////////////////////////////////////////////////////////
//Keys
///////////////////////////////////////////////////////////////////////////////

func (s *Storage) keyDecrypt(k *structures.Key) (*structures.Key, error) {
	passwordDecrypted, err := DecryptString(s.config.Secret, k.Password)
	if err != nil {
		return nil, err
	}
	k.Password = passwordDecrypted
	return k, nil
}

func (s *Storage) keyDecryptAll(keys []*structures.Key) []*structures.Key {
	var decryptedKeys []*structures.Key

	for _, key := range keys {
		decrypted, err := s.keyDecrypt(key)
		if err != nil {
			log.Warnf("unable to decrypt key: %s", err)
			continue
		}
		decryptedKeys = append(decryptedKeys, decrypted)
	}

	return decryptedKeys
}

func (s *Storage) keyEncrypt(k *structures.Key) (*structures.Key, error) {
	passwordEncrypted, err := EncryptPassword(k.Password)
	if err != nil {
		return nil, err
	}
	k.Password = passwordEncrypted
	return k, nil
}

func (s *Storage) KeyGetAll() ([]*structures.Key, error) {
	query := `select id, name, password
	          from keys 
	          where not coalesce(deleted, false) 
	          order by name`

	var keys []*structures.Key
	if err := s.db.Select(&keys, query); err != nil {
		return nil, err
	}
	return s.keyDecryptAll(keys), nil
}

func (s *Storage) KeyGet(id string) (*structures.Key, error) {
	query := `select id, name, password
	          from keys 
	          where id = $1 
	            and not coalesce(deleted, false) 
	          order by name`

	var key structures.Key
	if err := s.db.Get(&key, query, id); err != nil {
		return nil, err
	}
	return s.keyDecrypt(&key)
}

func (s *Storage) KeyInsert(key *structures.Key) error {
	if key == nil {
		return errors.New("key insert nil")
	}
	if len(key.Name) == 0 {
		return errors.New("key insert empty name")
	}
	if len(key.Id) == 0 {
		key.Id = NewId()
	}

	query := `insert into keys (id, name, password) values (:id, :name, :password)`
	keyToSave := *key
	if _, err := s.keyEncrypt(&keyToSave); err != nil {
		return err
	}
	if _, err := s.db.NamedExec(query, keyToSave); err != nil {
		return err
	}
	return nil
}

func (s *Storage) KeyDelete(id string) error {
	query := `update keys set deleted = true where id = $1`
	_, err := s.db.Exec(query, id)
	return err
}
