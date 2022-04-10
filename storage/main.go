package storage

import (
	"database/sql"
	"ensemble/storage/structures"
	"errors"
	_ "github.com/lib/pq"
	"time"
)

type Configuration struct {
	Url string
}

type Storage struct {
	db *sql.DB
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

func (s *Storage) UserAnyExists() bool {
	query := `select count(1) from users`
	row := s.db.QueryRow(query)
	return queryExistsHelper(row)
}

func (s *Storage) UserExists(id string) bool {
	query := `select count(1) from users where id = $1`
	row := s.db.QueryRow(query, id)
	return queryExistsHelper(row)
}

func (s *Storage) UserExistsByLogin(login string) bool {
	query := `select count(1) from users where login = $1`
	row := s.db.QueryRow(query, login)
	return queryExistsHelper(row)
}

func (s *Storage) UserGet(id string) (*structures.User, error) {
	query := `select id, login, password, role from users where id = $1 and not deleted`
	row := s.db.QueryRow(query, id)
	if err := row.Err(); err != nil {
		return nil, err
	}

	var user structures.User
	if err := row.Scan(&user.Id, &user.Login, &user.Password, &user.Role); err != nil {
		return nil, err
	}

	return &user, nil
}

func (s *Storage) UserGetAll() ([]*structures.User, error) {
	query := `select id, login, password, role from users where not deleted order by login`

	rows, err := s.db.Query(query)
	if err != nil {
		return nil, err
	}

	var users []*structures.User

	for rows.Next() {
		var user structures.User
		if err := rows.Scan(&user.Id, &user.Login, &user.Password, &user.Role); err != nil {
			continue
		}
		users = append(users, &user)
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

///////////////////////////////////////////////////////////////////////////////
