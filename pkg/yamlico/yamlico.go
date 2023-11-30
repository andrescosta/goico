package yamlico

import (
	"os"

	"gopkg.in/yaml.v2"
)

func Decode[T any](file string, t *T) error {
	r, err := os.Open(file)
	if err != nil {
		return err
	}
	d := yaml.NewDecoder(r)
	if err = d.Decode(t); err != nil {
		return err
	}
	return nil
}

func Encode[T any](t *T) (*string, error) {
	o, err := yaml.Marshal(t)
	if err != nil {
		return nil, err
	} else {
		s := string(o)
		return &s, err
	}

}
