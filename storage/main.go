package storage

import (
	"database/sql"
	"ensemble/storage/structures"
	"errors"
	_ "github.com/lib/pq"
	log "github.com/sirupsen/logrus"
	"strings"
	"time"
)

type Configuration struct {
	Url string
}

type Storage struct {
	db *sql.DB
}

type Scanner interface {
	Scan(dest ...any) error
}

///////////////////////////////////////////////////////////////////////////////

func New(configuration Configuration) (*Storage, error) {
	db, err := sql.Open("postgres", configuration.Url)
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

	return &Storage{db: db}, nil
}

func (s *Storage) Close() error {
	return s.db.Close()
}

func queryExistsHelper(row *sql.Row) bool {
	if err := row.Err(); err != nil {
		return false
	}

	var count int
	if err := row.Scan(&count); err != nil {
		return false
	}

	return count > 0
}

///////////////////////////////////////////////////////////////////////////////
//Session
///////////////////////////////////////////////////////////////////////////////

func (s *Storage) SessionCreate(userId string) (*structures.Session, error) {
	id := NewId()
	created := time.Now()

	query := `insert into sessions (id, user_id, created) values ($1, $2, $3)`
	_, err := s.db.Exec(query, id, userId, created)
	if err != nil {
		return nil, err
	}

	session := &structures.Session{
		Id:      id,
		UserId:  userId,
		Created: created,
	}
	return session, nil
}

func (s *Storage) SessionGet(id string) (*structures.Session, error) {
	query := `select id, user_id, created from sessions where id = $1`
	row := s.db.QueryRow(query, id)
	if err := row.Err(); err != nil {
		return nil, err
	}

	session := &structures.Session{}
	if err := row.Scan(&session.Id, &session.UserId, &session.Created); err != nil {
		return nil, err
	}
	return session, nil
}

///////////////////////////////////////////////////////////////////////////////
//User
///////////////////////////////////////////////////////////////////////////////

func scanUser(s Scanner) (*structures.User, error) {
	var user structures.User
	if err := s.Scan(&user.Id, &user.Login, &user.Password, &user.Role); err != nil {
		return nil, err
	}
	return &user, nil
}

func (s *Storage) UserAnyExists() bool {
	query := `select count(1) from users`
	row := s.db.QueryRow(query)
	return queryExistsHelper(row)
}

func (s *Storage) UserExists(id string) bool {
	query := `select count(1) from users where id = $1 and not deleted`
	row := s.db.QueryRow(query, id)
	return queryExistsHelper(row)
}

func (s *Storage) UserExistsByLogin(login string) bool {
	query := `select count(1) from users where login = $1 and not deleted`
	row := s.db.QueryRow(query, login)
	return queryExistsHelper(row)
}

func (s *Storage) UserGet(id string) (*structures.User, error) {
	query := `select id, login, password, role from users where id = $1 and not deleted`
	row := s.db.QueryRow(query, id)
	if err := row.Err(); err != nil {
		return nil, err
	}
	return scanUser(row)
}

func (s *Storage) UserGetAll() ([]*structures.User, error) {
	query := `select id, login, password, role from users where not deleted order by login`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}

	var users []*structures.User

	for rows.Next() {
		user, err := scanUser(rows)
		if err != nil {
			log.Warnf("unable to read user: %s", err)
			continue
		}
		users = append(users, user)
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

	query := `insert into users (id, login, password, role) values ($1, $2, $3, $4)`
	if _, err := s.db.Exec(query, user.Id, user.Login, user.Password, user.Role); err != nil {
		return err
	}

	return nil
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

	query := `update users set login = $1, password = $2, role = $3 where id = $4`
	if _, err := s.db.Exec(query, user.Login, user.Password, user.Role, user.Id); err != nil {
		return err
	}

	return nil
}

func (s *Storage) UserDelete(id string) error {
	query := `update users set deleted=true where id = $1`
	if _, err := s.db.Exec(query, id); err != nil {
		return err
	}
	return nil
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

func scanProject(s Scanner) (*structures.Project, error) {
	var project structures.Project
	var inventoryList, variablesList string
	if err := s.Scan(
		&project.Id,
		&project.Name,
		&project.Description,
		&project.RepositoryUrl,
		&project.RepositoryBranch,
		&project.Inventory,
		&inventoryList,
		&project.Variables,
		&variablesList,
		&project.VaultPassword); err != nil {
		return nil, err
	}

	project.InventoryList = strings.Split(inventoryList, "|")
	project.VariablesList = strings.Split(variablesList, "|")

	return &project, nil
}

func (s *Storage) ProjectExists(id string) bool {
	query := `select count(1) from projects where id = $1 and not deleted`
	row := s.db.QueryRow(query, id)
	return queryExistsHelper(row)
}

func (s *Storage) ProjectExistsByName(name string) bool {
	query := `select count(1) from projects where name = $1 and not null`
	row := s.db.QueryRow(query, name)
	return queryExistsHelper(row)
}

func (s *Storage) ProjectGet(id string) (*structures.Project, error) {
	query := `select id, 
                     name, 
                     description, 
                     repo_url, 
                     repo_branch, 
                     inventory, 
                     inventory_list, 
                     variables, 
                     variables_list,
                     vault_password
              from projects
              where id = $1 and not deleted
	`

	row := s.db.QueryRow(query, id)
	if err := row.Err(); err != nil {
		return nil, err
	}

	return scanProject(row)
}

func (s *Storage) ProjectGetAll() ([]*structures.Project, error) {
	query := `select id, 
                     name, 
                     description, 
                     repo_url, 
                     repo_branch, 
                     inventory, 
                     inventory_list, 
                     variables, 
                     variables_list,
                     vault_password
              from projects
              where not deleted
              order by name
	`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}

	var projects []*structures.Project
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			log.Warnf("unable to read project: %s", err)
			continue
		}
		projects = append(projects, project)
	}

	return projects, nil
}

func (s *Storage) ProjectGetByUser(userId string) ([]*structures.Project, error) {
	query := `select id, 
                     name, 
                     description, 
                     repo_url, 
                     repo_branch, 
                     inventory, 
                     inventory_list, 
                     variables, 
                     variables_list,
                     vault_password
              from projects 
                left join projects_users_access on (projects_users_access.project_id = projects.id) 
              where not deleted and projects_users_access.user_id = $1
              order by name
	`

	rows, err := s.db.Query(query, userId)
	if err != nil {
		return nil, err
	}

	var projects []*structures.Project
	for rows.Next() {
		project, err := scanProject(rows)
		if err != nil {
			log.Warnf("unable to read project: %s", err)
		}
		projects = append(projects, project)
	}

	return projects, nil
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

	query := `
		insert into projects (id, name, description, repo_url, repo_branch, inventory, inventory_list, variables, variables_list, vault_password) 
		values ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10)
	`

	inventoryList := strings.Join(project.InventoryList, "|")
	variablesList := strings.Join(project.VariablesList, "|")

	if _, err := s.db.Exec(
		query,
		project.Id,
		project.Name,
		project.Description,
		project.RepositoryUrl,
		project.RepositoryBranch,
		project.Inventory,
		inventoryList,
		project.Variables,
		variablesList,
		project.VaultPassword); err != nil {
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
			name = $1, 
			description = $2, 
			repo_url = $3, 
			repo_branch = $4,
			inventory = $5,
			inventory_list = $6,
			variables = $7,
			variables_list = $8,
			vault_password = $9
		where id = $10
	`

	inventoryList := strings.Join(project.InventoryList, "|")
	variablesList := strings.Join(project.VariablesList, "|")

	if _, err := s.db.Exec(
		query,
		project.Name,
		project.Description,
		project.RepositoryUrl,
		project.RepositoryBranch,
		project.Inventory,
		inventoryList,
		project.Variables,
		variablesList,
		project.VaultPassword,
		project.Id); err != nil {
		return err
	}

	return nil
}

func (s *Storage) ProjectDelete(id string) error {
	query := `update projects set deleted = true where id = $1`
	if _, err := s.db.Exec(query, id); err != nil {
		return err
	}
	return nil
}

///////////////////////////////////////////////////////////////////////////////
//Project User Access
///////////////////////////////////////////////////////////////////////////////

func (s *Storage) ProjectUserAccessExists(projectId, userId string) bool {
	query := `select count(1) from projects_users_access where project_id = $1 and user_id = $2`
	row := s.db.QueryRow(query, projectId, userId)
	return queryExistsHelper(row)
}

func (s *Storage) ProjectUserAccessCreate(projectId, userId string) error {
	if len(projectId) == 0 {
		return errors.New("project user access empty project id")
	}
	if len(userId) == 0 {
		return errors.New("project user access empty user id")
	}

	query := `insert into projects_users_access (project_id, user_id) values ($1, $2)`
	if _, err := s.db.Exec(query, projectId, userId); err != nil {
		return err
	}
	return nil
}

func (s *Storage) ProjectUserAccessDelete(projectId, userId string) error {
	query := `delete from projects_users_access where project_id = $1 and user_id = $2`
	if _, err := s.db.Exec(query, projectId, userId); err != nil {
		return err
	}
	return nil
}
