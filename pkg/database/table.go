package database

import (
	"errors"

	"github.com/cockroachdb/pebble"
)

type Key struct {
	version []byte
	tenant  []byte
	table   []byte
	id      []byte
}

type Marshaler[S any] interface {
	Marshal(S) (string, []byte, error)
	Unmarshal([]byte) (S, error)
}

type Table[S any] struct {
	db        *Database
	marshaler Marshaler[S]
	Name      string
	Tenant    string
}

// func NewTable[S any](db *Database, tenant string, name string, marshaler Marshaler[S]) *Table[S] {
// 	table := &Table[S]{
// 		marshaler: marshaler,
// 		Name:      name,
// 		Tenant:    tenant,
// 		db:        db,
// 	}
// 	return table
// }

func NewTable[S any](db *Database, name string, tenant string, marshaler Marshaler[S]) *Table[S] {
	table := &Table[S]{
		marshaler: marshaler,
		Name:      name,
		Tenant:    tenant,
		db:        db,
	}
	return table
}

func (s *Table[S]) Add(data S) error {
	id, buf, err := s.marshaler.Marshal(data)
	if err != nil {
		return err
	}
	k := s.getKey(id)
	return s.db.db.Set(k.encode(), buf, pebble.Sync)
}

func (s *Table[S]) Update(data S) error {
	id, buf, err := s.marshaler.Marshal(data)
	if err != nil {
		return err
	}
	k := s.getKey(id)
	return s.db.db.Set(k.encode(), buf, pebble.Sync)
}

func (s *Table[S]) Delete(id string) error {
	k := s.getKey(id)
	return s.db.db.Delete(k.encode(), pebble.Sync)
}

func (s *Table[S]) Get(id string) (*S, error) {
	k := s.getKey(id)
	d, iter, err := s.db.db.Get(k.encode())
	if err != nil {
		if errors.Is(err, pebble.ErrNotFound) {
			return nil, nil
		}
		return nil, err
	}
	errs := make([]error, 0)
	e, err := s.marshaler.Unmarshal(d)
	if err != nil {
		errs = append(errs, err)
	}
	if err := iter.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return &e, nil
}

func (s *Table[S]) All() ([]S, error) {
	var data []S
	data = make([]S, 0)

	k := s.getKey("")

	keyUpperBound := func(b []byte) []byte {
		end := make([]byte, len(b))
		copy(end, b)
		for i := len(end) - 1; i >= 0; i-- {
			end[i] = end[i] + 1
			if end[i] != 0 {
				return end[:i+1]
			}
		}
		return nil // no upper-bound
	}

	prefixIterOptions := func(prefix []byte) *pebble.IterOptions {
		return &pebble.IterOptions{
			LowerBound: prefix,
			UpperBound: keyUpperBound(prefix),
		}
	}
	errs := make([]error, 0)
	iter := s.db.db.NewIter(prefixIterOptions(k.encodepreffix()))
	for iter.First(); iter.Valid(); iter.Next() {
		d, err := s.marshaler.Unmarshal(iter.Value())
		if err != nil {
			errs = append(errs, err)
			break
		}
		data = append(data, d)
	}
	if err := iter.Close(); err != nil {
		errs = append(errs, err)
	}
	if len(errs) > 0 {
		return nil, errors.Join(errs...)
	}
	return data, nil
}

func (s *Table[S]) getKey(id string) *Key {
	return &Key{
		version: []byte("0"),
		tenant:  []byte(s.Tenant),
		table:   []byte(s.Name),
		id:      []byte(id),
	}
}

func (k *Key) encode() []byte {
	d := make([]byte, k.len())
	n := 0
	copy(d[n:], k.version)
	n = n + len(k.version)
	copy(d[n:], k.tenant)
	n = n + len(k.tenant)
	copy(d[n:], k.table)
	n = n + len(k.table)
	copy(d[n:], k.id)
	return d
}

func (k *Key) encodepreffix() []byte {
	d := make([]byte, k.lenpreffix())
	n := 0
	copy(d[n:], k.version)
	n = n + len(k.version)
	copy(d[n:], k.tenant)
	n = n + len(k.tenant)
	copy(d[n:], k.table)
	return d
}

func (k *Key) len() int {
	return len(k.version) +
		len(k.tenant) +
		len(k.table) +
		len(k.id)
}

func (k *Key) lenpreffix() int {
	return len(k.version) +
		len(k.tenant) +
		len(k.table)
}
