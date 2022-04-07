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
