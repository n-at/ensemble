package storage

import (
	"database/sql"
	_ "github.com/lib/pq"
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
