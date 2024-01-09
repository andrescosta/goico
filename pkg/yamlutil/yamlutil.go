package yamlutil

import (
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

func DecodeFile[T any](name string, t *T) error {
	name = filepath.Clean(name)
	f, err := os.Open(name)
	if err != nil {
		return err
	}
	defer f.Close()
	d := yaml.NewDecoder(f)
	if err = d.Decode(t); err != nil {
		return err
	}
	return nil
}

func Marshal[T any](t *T) (*string, error) {
	o, err := yaml.Marshal(t)
	if err != nil {
		return nil, err
	}
	s := string(o)
	return &s, err
}
