package test

import (
	"fmt"
	"os"
	"strings"
)

func SetargsV(name string, value string) {
	os.Args = append(os.Args, fmt.Sprintf("--env:%s=%s", name, value))
}

func Setargs(args ...string) {
	for _, arg := range args {
		os.Args = append(os.Args, fmt.Sprintf("--env:%s", arg))
	}
}

type BackupEnvData struct {
	args   []string
	envs   []string
	stdout *os.File
	stderr *os.File
	stdin  *os.File
}

func DoBackupEnv() BackupEnvData {
	old := os.Args
	newArgs := make([]string, len(old))
	copy(newArgs, os.Args)
	os.Args = newArgs
	return BackupEnvData{
		args:   old,
		envs:   os.Environ(),
		stdout: os.Stdout,
		stderr: os.Stderr,
		stdin:  os.Stdin,
	}
}

func RestoreEnv(b BackupEnvData) {
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
