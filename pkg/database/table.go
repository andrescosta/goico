package database

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/rs/zerolog"
	bolt "go.etcd.io/bbolt"
)

type Marshaler[S any] interface {
	Marshal(uint64, S) ([]byte, error)
	MarshalObj(S) (uint64, []byte, error)
	Unmarshal(uint64, []byte) (S, error)
}

type Table[S any] struct {
	db        *Database
	marshaler Marshaler[S]
	name      string
}

func GetTable[S any](ctx context.Context, db *Database, name string, marshaler Marshaler[S]) (*Table[S], error) {
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
			return fmt.Errorf("table does not exist")
		}
		var err error
		id, err = b.NextSequence()
		if err != nil {
			return err
		}
		buf, err := s.marshaler.Marshal(id, data)
		if err != nil {
			return err
		}
		if err = b.Put(itob(id), buf); err != nil {
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
			return fmt.Errorf("table does not exist")
		}
		var err error
		id, buf, err := s.marshaler.MarshalObj(data)
		if err != nil {
			return err
		}
		if err = b.Put(itob(id), buf); err != nil {
			return err
		}
		return nil
	}); err != nil {
		return err
	}
	return nil
}

func (s *Table[S]) Get(_ context.Context, id uint64) (S, error) {
	var data S
	s.db.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.name))
		if b == nil {
			return fmt.Errorf("table does not exist")
		}
		var err error
		data, err = s.marshaler.Unmarshal(id, b.Get(itob(id)))
		if err != nil {
			return err
		}
		return nil
	})
	return data, nil
}

func (s *Table[S]) All(ctx context.Context) ([]S, error) {
	var data []S
	data = make([]S, 0)
	logger := zerolog.Ctx(ctx)
	if err := s.db.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.name))
		if b == nil {
			return fmt.Errorf("table does not exist")
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			d, err := s.marshaler.Unmarshal(btoi(k), v)
			if err != nil {
				logger.Warn().Msgf("Error deserializing data %s", err)
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

func itob(v uint64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, v)
	return b
}

func btoi(v []byte) uint64 {
	return binary.BigEndian.Uint64(v)
}
