package storage

import (
	"database/sql"
	"errors"
	log "github.com/sirupsen/logrus"
)

type Migration struct {
	version int
	name    string
	query   string
}

type Migrator struct {
	db         *sql.DB
	migrations []Migration
}

///////////////////////////////////////////////////////////////////////////////

func NewMigrator(db *sql.DB) (*Migrator, error) {
	if db == nil {
		return nil, errors.New("nil database")
	}
	return &Migrator{
		db:         db,
		migrations: migrations,
	}, nil
}

///////////////////////////////////////////////////////////////////////////////

var migrations = []Migration{
	{
		version: 1,
		name:    "users table",
		query: `
			create table users (
				id          varchar(64)     primary key,
				deleted     boolean,
				login       varchar(250)    unique not null,
				password    varchar(250)    not null,
				role        int             not null
			)
		`,
	}, {
		version: 2,
		name:    "users.login index",
		query:   `create index if not exists users_login on users (login)`,
	}, {
		version: 3,
		name:    "projects table",
		query: `
			create table projects (
				id              varchar(64)     primary key,
				deleted         boolean,
				name            varchar(250)    not null,
				description     text,
				repo_url        varchar(1000)   not null,
				repo_branch     varchar(250)    not null,
				inventory       varchar(250),
				inventory_list  text,
				variables       varchar(250),
				variables_list  text,
				vault_password  text
			)
		`,
	}, {
		version: 4,
		name:    "project updates table",
		query: `
			create table project_updates (
				id          varchar(64)     primary key,
				project_id  varchar(64)     not null,
				deleted     boolean,
				date        timestamp       not null,
				revision    text,
				log         text
			)
		`,
	}, {
		version: 5,
		name:    "project_updates.project_id index",
		query:   `create index if not exists project_updates_project_id on project_updates (project_id)`,
	}, {
		version: 6,
		name:    "playbooks table",
		query: `
			create table playbooks (
				id          varchar(64)     primary key,
				project_id  varchar(64)     not null,
				deleted     boolean,
				filename    varchar(250)    not null,
				name        text,
				description text,
				locked      boolean
			)
		`,
	}, {
		version: 7,
		name:    "playbooks.project_id index",
		query:   `create index if not exists playbooks_project_id on playbooks (project_id)`,
	}, {
		version: 8,
		name:    "playbook runs table",
		query: `
			create table playbook_runs (
				id          varchar(64)     primary key,
				playbook_id varchar(64)     not null,
				user_id     varchar(64)     not null,
				deleted     boolean,
				mode        integer         not null,
				start_time  timestamp,
				finish_time timestamp,
				result      int
			)
		`,
	}, {
		version: 9,
		name:    "playbook_runs.playbook_id index",
		query:   `create index if not exists playbook_runs_playbook_id on playbook_runs (playbook_id)`,
	}, {
		version: 10,
		name:    "playbook_runs.user_id index",
		query:   `create index if not exists playbook_runs_user_id on playbook_runs (user_id)`,
	}, {
		version: 11,
		name:    "run results table",
		query: `
			create table run_results (
				id      varchar(64) primary key,
				run_id  varchar(64) not null,
				deleted boolean,
				output  text
			)
		`,
	}, {
		version: 12,
		name:    "run_results.run_id index",
		query:   `create index if not exists run_results_run_id on run_results (run_id)`,
	}, {
		version: 13,
		name:    "sessions table",
		query: `
			create table sessions (
				id      varchar(64) 	primary key,
				user_id varchar(64) 	not null
			)
		`,
	}, {
		version: 14,
		name:    "session.created field",
		query:   `alter table sessions add column created timestamp`,
	}, {
		version: 15,
		name:    "projects users access table",
		query: `
			create table projects_users_access (
				project_id varchar(64) not null,
				user_id varchar(64) not null,
				constraint projects_users_access_pk primary key (project_id, user_id)
			)
		`,
	}, {
		version: 16,
		name:    "projects_users_access.project_id index",
		query:   `create index if not exists projects_users_access_project_id on projects_users_access (project_id)`,
	}, {
		version: 17,
		name:    "projects_users_access.user_id index",
		query:   `create index if not exists projects_users_access_user_id on projects_users_access (user_id)`,
	}, {
		version: 18,
		name:    "projects.name index",
		query:   `create index if not exists projects_name on projects (name)`,
	}, {
		version: 19,
		name:    "project_updates.success field",
		query:   `alter table project_updates add column success boolean`,
	}, {
		version: 20,
		name:    "projects.collections_list field",
		query:   `alter table projects add column collections_list text`,
	}, {
		version: 21,
		name:    "projects.variables_main field",
		query:   `alter table projects add column variables_main boolean`,
	}, {
		version: 22,
		name:    "projects.variables_vault field",
		query:   `alter table projects add column variables_vault boolean`,
	}, {
		version: 23,
		name:    "projects.repo_login field",
		query:   `alter table projects add column repo_login varchar(500)`,
	}, {
		version: 24,
		name:    "projects.repo_password field",
		query:   `alter table projects add column repo_password varchar(500)`,
	}, {
		version: 25,
		name:    "run_results.error field",
		query:   `alter table run_results add column error text`,
	}, {
		version: 26,
		name:    "store projects.repo_password as text",
		query:   `alter table projects alter column repo_password type text`,
	},
}

///////////////////////////////////////////////////////////////////////////////

func (m *Migrator) migrate() error {
	if err := m.createMigrationTable(); err != nil {
		return err
	}
	for _, migration := range m.migrations {
		exists, err := m.exists(migration.version)
		if err != nil {
			return err
		}
		if exists {
			continue
		}
		if err := m.apply(migration); err != nil {
			return err
		}
	}
	return nil
}

func (m *Migrator) createMigrationTable() error {
	log.Infof("checking migration table...")

	query := `
		create table if not exists __migrations (
			version integer primary key,
            name varchar(500)
		)
	`
	_, err := m.db.Exec(query)
	return err
}

func (m *Migrator) exists(version int) (bool, error) {
	query := `select count(*) as cnt from __migrations where version = $1`
	var count int

	if err := m.db.QueryRow(query, version).Scan(&count); err != nil {
		return false, err
	}

	return count > 0, nil
}

func (m *Migrator) apply(migration Migration) error {
	log.Infof("applying migration %d: %s", migration.version, migration.name)

	if _, err := m.db.Exec(migration.query); err != nil {
		return err
	}

	query := `insert into __migrations (version, name) values ($1, $2)`
	if _, err := m.db.Exec(query, migration.version, migration.name); err != nil {
		return err
	}

	return nil
}
