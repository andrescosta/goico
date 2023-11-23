package database

import (
	"context"

	"github.com/rs/zerolog"
	bolt "go.etcd.io/bbolt"
)

type Database struct {
	db *bolt.DB
}

func Open(ctx context.Context, path string) (*Database, error) {
	db, err := bolt.Open(path, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &Database{
		db: db,
	}, nil
}

func (s *Database) Close(ctx context.Context) error {
	logger := zerolog.Ctx(ctx)
	if err := s.db.Close(); err != nil {
		logger.Debug().Msgf("Error closing the DB:%s", err)
		return err
	}
	return nil
}
