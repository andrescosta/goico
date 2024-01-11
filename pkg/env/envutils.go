package env

import (
	"os"
	"strings"
)

type BackupVals struct {
	args   []string
	envs   []string
	stdout *os.File
	stderr *os.File
	stdin  *os.File
}

func Backup() BackupVals {
	old := os.Args
	newArgs := make([]string, len(old))
	copy(newArgs, os.Args)
	os.Args = newArgs
	return BackupVals{
		args:   old,
		envs:   os.Environ(),
		stdout: os.Stdout,
		stderr: os.Stderr,
		stdin:  os.Stdin,
	}
}

func Restore(b BackupVals) {
	os.Args = b.args
	os.Clearenv()
	for _, ss := range b.envs {
		sss := strings.Split(ss, "=")
		_ = os.Setenv(sss[0], sss[1])
	}
	os.Stdout = b.stdout
	os.Stderr = b.stderr
	os.Stdin = b.stdin
}
