package database

import (
	"time"

	bolt "go.etcd.io/bbolt"
)

type Database struct {
	db *bolt.DB
}

func Open(path string) (*Database, error) {
	db, err := bolt.Open(path, 0600, &bolt.Options{
		Timeout: 3 * time.Second,
	})
	if err != nil {
		return nil, err
	}
	return &Database{
		db: db,
	}, nil
}
func (s *Database) Close() error {
	if err := s.db.Close(); err != nil {
		return err
	}
	return nil
}
