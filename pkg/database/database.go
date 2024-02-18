package database

import (
	"context"
	"sync"

	"github.com/cockroachdb/pebble"
	"github.com/cockroachdb/pebble/vfs"
	"github.com/rs/zerolog"
)

type dbLog struct {
	ctx context.Context
}

func (d *dbLog) Infof(format string, args ...interface{}) {
	zerolog.Ctx(d.ctx).Info().Msgf(format, args...)
}

func (d *dbLog) Fatalf(format string, args ...interface{}) {
	zerolog.Ctx(d.ctx).Fatal().Msgf(format, args...)
}

type Database struct {
	mu sync.RWMutex
	db *pebble.DB
}

type Option struct {
	InMemory bool
}

func Open(ctx context.Context, path string, ops Option) (*Database, error) {
	log := &dbLog{ctx: ctx}
	opts := &pebble.Options{
		Logger: log,
	}
	if ops.InMemory {
		opts.FS = vfs.NewMem()
	}
	db, err := pebble.Open(path, opts)
	if err != nil {
		return nil, err
	}

	return &Database{
		db: db,
		mu: sync.RWMutex{},
	}, nil
}

func (s *Database) Close() error {
	return s.db.Close()
}
