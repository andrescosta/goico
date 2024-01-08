package database

import (
	"fmt"

	bolt "go.etcd.io/bbolt"
)

type NoTableError struct {
	table string
}

func (n NoTableError) Error() string {
	return fmt.Sprintf("%s does not exist.", n.table)
}

type Marshaler[S any] interface {
	Marshal(S) (string, []byte, error)
	Unmarshal([]byte) (S, error)
}
type Table[S any] struct {
	db        *Database
	marshaler Marshaler[S]
	Name      string
}

func NewTable[S any](db *Database, name string, marshaler Marshaler[S]) *Table[S] {
	table := &Table[S]{
		marshaler: marshaler,
		Name:      name,
		db:        db,
	}
	return table
}
func CreateTableIfNotExist[S any](db *Database, name string, marshaler Marshaler[S]) (*Table[S], error) {
	table := NewTable(db, name, marshaler)
	if err := db.db.Update(func(tx *bolt.Tx) error {
		_, err := tx.CreateBucketIfNotExists([]byte(table.Name))
		return err
	}); err != nil {
		return nil, err
	}
	return table, nil
}

func (s *Table[S]) Add(data S) error {
	if err := s.db.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.Name))
		if b == nil {
			return NoTableError{s.Name}
		}
		var err error
		id, buf, err := s.marshaler.Marshal(data)
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

func (s *Table[S]) Update(data S) error {
	if err := s.db.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.Name))
		if b == nil {
			return NoTableError{s.Name}
		}
		var err error
		id, buf, err := s.marshaler.Marshal(data)
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

func (s *Table[S]) Delete(id string) error {
	if err := s.db.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.Name))
		if b == nil {
			return NoTableError{s.Name}
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

func (s *Table[S]) Get(id string) (*S, error) {
	var data *S
	if err := s.db.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.Name))
		if b == nil {
			return NoTableError{s.Name}
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

func (s *Table[S]) All() ([]S, error) {
	var data []S
	data = make([]S, 0)
	if err := s.db.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte(s.Name))
		if b == nil {
			return NoTableError{s.Name}
		}
		c := b.Cursor()
		for k, v := c.First(); k != nil; k, v = c.Next() {
			d, err := s.marshaler.Unmarshal(v)
			if err != nil {
				return err
			}
			data = append(data, d)
		}
		return nil
	}); err != nil {
		return nil, err
	}
	return data, nil
}
