package env

import (
	"os"
	"strings"
)

// parse --env:key=value or -env:key=value
func setEnvsUsingCommandLineArgs() error {
	return load(os.Args[1:])
}

func load(args []string) error {
	if len(args) > 0 {
		m := parse(args)
		for k, v := range m {
			if err := os.Setenv(k, v); err != nil {
				return err
			}
		}
	}
	return nil
}

func parse(args []string) map[string]string {
	m := make(map[string]string)
	for _, arg := range args {
		v, ok := strings.CutPrefix(arg, "--env:")
		if ok {
			vs := strings.SplitN(v, "=", 2)
			if len(vs) == 2 {
				m[vs[0]] = vs[1]
			}
			continue
		}
		v, ok = strings.CutSuffix(arg, "-e:")
		if ok {
			vs := strings.SplitN(v, "=", 2)
			if len(vs) < 2 {
				continue
			}
			m[vs[0]] = vs[1]
		}
	}
	return m
}
