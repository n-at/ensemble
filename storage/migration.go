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
		migrations: []Migration{},
	}, nil
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
