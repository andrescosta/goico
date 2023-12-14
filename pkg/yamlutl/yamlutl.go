package yamlutl

import (
	"os"

	"gopkg.in/yaml.v2"
)

func DecodeFile[T any](name string, t *T) error {
	f, err := os.Open(name)
	if err != nil {
		return err
	}
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
	} else {
		s := string(o)
		return &s, err
	}

}
