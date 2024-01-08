package database

import (
	"bytes"
	"encoding/gob"
)

type Identifiable interface {
	Id() string
}

type BinaryMarshaller[T Identifiable] struct {
}

func (d BinaryMarshaller[T]) Marshal(v T) (string, []byte, error) {
	b := bytes.Buffer{}
	e := gob.NewEncoder(&b)
	err := e.Encode(v)
	if err != nil {
		return "", nil, err
	}
	return v.Id(), b.Bytes(), nil
}

func (d BinaryMarshaller[T]) Unmarshal(v []byte) (T, error) {
	b := bytes.NewBuffer(v)
	e := gob.NewDecoder(b)
	var t T
	if err := e.Decode(&t); err != nil {
		return t, err
	}
	return t, nil
}
