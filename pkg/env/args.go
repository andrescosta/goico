package env

import (
	"flag"
	"os"
	"strings"
)

// parse --env:key=value or -env:key=value
func setEnvsUsingCommandLineArgs() error {
	return setEnvs(os.Args[1:])
}

func setEnvs(args []string) error {
	if len(args) > 0 {
		m := parse(args)
		for k, v := range m {
			if err := os.Setenv(k, v); err != nil {
				return err
			}
			_ = flag.String("env:"+k, v, "server address")
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
		v, ok = strings.CutPrefix(arg, "-e:")
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
