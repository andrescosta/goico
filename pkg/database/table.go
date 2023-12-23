package database

import (
	"context"
	"fmt"

	"github.com/rs/zerolog"
	bolt "go.etcd.io/bbolt"
)

type Marshaler[S any] interface {
	Marshal(S) ([]byte, error)
	MarshalObj(S) (string, []byte, error)
	Unmarshal([]byte) (S, error)
}
type Table[S any] struct {
	db        *Database
	marshaler Marshaler[S]
	name      string
}

func NewTable[S any](_ context.Context, db *Database, name string, marshaler Marshaler[S]) (*Table[S], error) {
	table := &Table[S]{
		marshaler: marshaler,
		name:      name,
		db:        db,
	}
	if err := db.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(table.name))
		return err
	}); err != nil {
		return nil, err
	}
	return table, nil
}

func (s *Table[S]) Add(_ context.Context, data S) (uint64, error) {
	var id uint64
	if err := s.db.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.name))
		if b == nil {
			return fmt.Errorf("table %s does not exist", s.name)
		}
		var err error
		id, buf, err := s.marshaler.MarshalObj(data)
		if err != nil {
			return err
		}
		if err = b.Put([]byte(id), buf); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return 0, err
	}
	return id, nil
}

func (s *Table[S]) Update(_ context.Context, data S) error {
	if err := s.db.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.name))
		if b == nil {
			return fmt.Errorf("table %s does not exist", s.name)
		}
		var err error
		id, buf, err := s.marshaler.MarshalObj(data)
		if err != nil {
			return err
		}
		if err = b.Put([]byte(id), buf); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *Table[S]) Delete(_ context.Context, id string) error {
	if err := s.db.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.name))
		if b == nil {
			return fmt.Errorf("table %s does not exist", s.name)
		}
		if err := b.Delete([]byte(id)); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *Table[S]) Get(_ context.Context, id string) (*S, error) {
	var data *S
	if err := s.db.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.name))
		if b == nil {
			return fmt.Errorf("table %s does not exist", s.name)
		}
		d := b.Get([]byte(id))
		if d != nil {
			e, err := s.marshaler.Unmarshal(d)
			if err != nil {
				return err
			}
			data = &e
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return data, nil
}

func (s *Table[S]) All(ctx context.Context) ([]S, error) {
	var data []S
	data = make([]S, 0)
	logger := zerolog.Ctx(ctx)
	if err := s.db.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.name))
		if b == nil {
			return fmt.Errorf("table %s does not exist", s.name)
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			d, err := s.marshaler.Unmarshal(v)
			if err != nil {
				logger.Warn().Msgf("marshaler.Unmarshal: Error deserializing data %s", err)
				continue
			}
			data = append(data, d)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return data, nil
}
