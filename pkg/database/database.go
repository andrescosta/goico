package database

import (
	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
)

type Database struct {
	db *pebble.DB
}

type Options struct {
	InMemory bool
}

func Open(path string, ops *Options) (*Database, error) {
	opts := &pebble.Options{}
	if ops != nil && ops.InMemory {
		opts.FS = vfs.NewMem()
	}
	db, err := pebble.Open(path, opts)
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
