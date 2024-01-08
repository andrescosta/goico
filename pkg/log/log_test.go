package log_test

import (
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/ioutil"
	. "github.com/andrescosta/goico/pkg/log"
	"github.com/rs/zerolog"
)

type (
	expectedScenario     func(*testing.T, string)
	expectedScenarioJson func(*testing.T, logLine)
)

type logLine struct {
	Message string
	Client  string
	Host    string
}

const (
	typeConsole = iota
	typeFile
)

type (
	backup struct {
		envs   []string
		stdout *os.File
	}
	scenario struct {
		types          int
		name           string
		context        map[string]string
		fileName       string
		envs           []string
		lvl            zerolog.Level
		text           string
		expected       *string
		expectedFn     expectedScenario
		expectedFnJson expectedScenarioJson
	}
)

func TestLogToFile(t *testing.T) {
	tempDir := t.TempDir()
	scenarios := []scenario{
		newFileScenario(
			"file_debug",
			[]string{
				"log.console.enabled=false",
				"log.console.exclude.timestamp=true",
				"log.file.enabled=true",
				"log.level=0",
				"log.file.name=${workdir}/file_debug.log",
			},
			zerolog.DebugLevel,
			"file_debug_text",
			"file_debug_text",
		),
		newFileScenarioWithFnAndContext(
			"file_debug_with_context",
			[]string{
				"log.console.enabled=false",
				"log.console.exclude.timestamp=true",
				"log.file.enabled=true",
				"log.level=0",
				"log.file.name=${workdir}/file_debug_with_context.log",
			},
			[]string{
				"client=windows",
				"host=127.0.0.1",
			},
			zerolog.DebugLevel,
			"file_debug_with_context_text",
			"file_debug_with_context_text",
			func(t *testing.T, l logLine) {
				if l.Host != "127.0.0.1" {
					t.Errorf("expected host=127.0.0.1 got %s", l.Host)
				}
				if l.Client != "windows" {
					t.Errorf("expected client=windows got %s", l.Client)
				}
			},
		),
		newFileScenario(
			"file_info",
			[]string{
				"log.console.enabled=false",
				"log.console.exclude.timestamp=true",
				"log.file.enabled=true",
				"log.level=1",
				"log.file.name=${workdir}/file_info.log",
			},
			zerolog.InfoLevel,
			"file_info_text",
			"file_info_text",
		),
		newFileScenario(
			"file_info_ts",
			[]string{
				"log.console.enabled=false",
				"log.console.exclude.timestamp=false",
				"log.file.enabled=true",
				"log.level=1",
				"log.file.name=${workdir}/file_info_ts.log",
			},
			zerolog.InfoLevel,
			"file_info_ts_text",
			"file_info_ts_text",
		),
		newFileScenarioWithFn(
			"file_debug_luberjack",
			[]string{
				"log.console.enabled=false",
				"log.console.exclude.timestamp=true",
				"log.file.enabled=true",
				"log.file.max.size=180",
				"log.file.max.backups=200",
				"log.file.max.age=300",
				"log.level=0",
				"log.file.name=${workdir}/file_debug_luberjack.log",
			},
			zerolog.DebugLevel,
			"file_debug_luberjack_txt",
			"file_debug_luberjack_txt",
			func(t *testing.T, l logLine) {
				lj := Luberjack()
				if lj.MaxSize != 180 {
					t.Errorf("expected Luberjack.MaxSize = 180 got %d", lj.MaxSize)
				}
				if lj.MaxAge != 300 {
					t.Errorf("expected Luberjack.MaxAge = 300 got %d", lj.MaxAge)
				}
				if lj.MaxBackups != 200 {
					t.Errorf("expected Luberjack.MaxBackups = 200 got %d", lj.MaxBackups)
				}
			},
		),
	}
	execute(t, scenarios, tempDir)
}
func TestLogToConsole(t *testing.T) {
	tempDir := t.TempDir()
	scenarios := []scenario{
		newConsoleScenario(
			"console_disabled",
			[]string{
				"log.console.enabled=false",
				"log.file.enabled=false",
			},
			zerolog.DebugLevel,
			"test_disabled",
			"",
		),
		newConsoleScenario(
			"console_debug",
			[]string{
				"log.console.enabled=true",
				"log.console.exclude.timestamp=true",
				"log.file.enabled=false",
				"log.level=0",
			},
			zerolog.DebugLevel,
			"test_debug",
			"test_debug",
		),
		newConsoleScenario(
			"console_info_no_log",
			[]string{
				"log.console.enabled=true",
				"log.console.exclude.timestamp=true",
				"log.file.enabled=false",
				"log.level=1",
			},
			zerolog.DebugLevel,
			"test_debug_nolog",
			"",
		),
		newConsoleScenario(
			"console_info",
			[]string{
				"log.console.enabled=true",
				"log.console.exclude.timestamp=true",
				"log.file.enabled=false",
				"log.level=1",
			},
			zerolog.InfoLevel,
			"test_info",
			"test_info",
		),
		newConsoleScenarioWithFn(
			"console_info_ts",
			[]string{
				"log.console.enabled=true",
				"log.console.exclude.timestamp=false",
				"log.file.enabled=false",
				"log.level=1",
			},
			nil,
			zerolog.InfoLevel,
			"test_info_ts",
			"test_info_ts",
			func(t *testing.T, s string) {
				year := strconv.Itoa(time.Now().Year())
				if !strings.Contains(s, year) {
					t.Errorf("expected contains %s", year)
				}
			},
		),
		newConsoleScenarioWithFn(
			"console_debug_caller",
			[]string{
				"log.console.enabled=true",
				"log.console.exclude.timestamp=true",
				"log.file.enabled=false",
				"log.level=0",
				"log.caller=true",
			},
			nil,
			zerolog.DebugLevel,
			"test_debug_caller",
			"test_debug_caller",
			func(t *testing.T, s string) {
				if !strings.Contains(s, "log_test.go") {
					t.Errorf("expecyed caller: log_test.go got %s", s)
				}
			},
		),
		newConsoleScenarioWithFn(
			"console_debug_nofile",
			[]string{
				"log.console.enabled=true",
				"log.console.exclude.timestamp=true",
				"log.file.enabled=false",
				"log.level=0",
				"log.file.name=${workdir}/console_debug_nofile.log",
			},
			nil,
			zerolog.DebugLevel,
			"test_debug_nofile",
			"test_debug_nofile",
			func(t *testing.T, s string) {
				file := filepath.Join(env.WorkDir(), "console_debug_nofile.log")
				e, err := ioutil.FileExists(file)
				if err != nil {
					t.Errorf("ioutil.FileExists: %s", err)
				}
				if e {
					t.Error("console_debug_nofile.log exists")
				}
			},
		),
		newConsoleScenarioWithFn(
			"console_debug_file",
			[]string{
				"log.console.enabled=true",
				"log.console.exclude.timestamp=true",
				"log.file.enabled=true",
				"log.level=0",
				"log.file.name=${workdir}/console_debug_file.log",
			},
			nil,
			zerolog.DebugLevel,
			"test_debug_nofile",
			"test_debug_nofile",
			func(t *testing.T, s string) {
				file := filepath.Join(env.WorkDir(), "console_debug_file.log")
				e, err := ioutil.FileExists(file)
				if err != nil {
					t.Errorf("ioutil.FileExists: %s", err)
				}
				if !e {
					t.Error("console_debug_file.log not exists")
				}
			},
		),
		newConsoleScenarioWithContext(
			"console_error_withcontext",
			[]string{
				"log.console.enabled=true",
				"log.console.exclude.timestamp=true",
				"log.file.enabled=false",
				"log.level=3",
			},
			[]string{
				"client=windows",
				"host=127.0.0.1",
			},
			zerolog.ErrorLevel,
			"console_error_withcontext_txt",
			"console_error_withcontext_txt",
		),
	}
	execute(t, scenarios, tempDir)
}

func execute(t *testing.T, scenarios []scenario, tempDir string) {
	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			b := makeBackup()
			setEnvs(s.envs, tempDir)
			t.Cleanup(func() {
				restoreBackup(b)
			})
			var w *os.File
			var r io.Reader
			var err error
			if s.types == typeConsole {
				r, w, err = os.Pipe()
				if err != nil {
					t.Fatalf("os.Pipe: %s", err)
				}
				os.Stdout = w
			}
			var logger *zerolog.Logger
			if s.context != nil {
				logger = NewWithContext(s.context)
			} else {
				logger = New()
			}
			logger.WithLevel(s.lvl).Msg(s.text)
			if err := Close(); err != nil {
				t.Errorf("log.Close: %s", err)
			}
			if s.types == typeConsole {
				if err := w.Close(); err != nil {
					t.Errorf("File.Close: %s", err)
				}
				outb, err := io.ReadAll(r)
				if err != nil {
					t.Errorf("io.ReadAll: %s", err)
					return
				}
				out := string(outb)
				if s.expected != nil {
					if *s.expected == "" {
						if out != "" {
							t.Errorf(`expected <empty string> got %s`, out)
							return
						}
						return
					}
					if !strings.Contains(out, *s.expected) {
						t.Errorf(`expected %s got %s`, *s.expected, out)
					}
					if s.context != nil {
						for k := range s.context {
							if !strings.Contains(out, k+"=") {
								t.Errorf(`expected %s `, k)
							}
						}
					}
					if s.expectedFn != nil {
						s.expectedFn(t, out)
					}
				}
				return
			}
			if s.types == typeFile {
				fileName := filepath.Join(tempDir, s.fileName)
				fo, err := os.Open(fileName)
				if err != nil {
					t.Errorf("os.Open: %s", err)
					return
				}
				r = fo
				defer func() {
					if err := fo.Close(); err != nil {
						t.Errorf("os.File %s", err)
					}
				}()
				logLine := logLine{}
				json.NewDecoder(r).Decode(&logLine)
				if err != nil {
					t.Errorf("io.ReadAll: %s", err)
					return
				}
				if s.expected != nil {
					if *s.expected == "" {
						if logLine.Message != "" {
							t.Errorf(`expected <empty string> got %s`, logLine.Message)
							return
						}
						return
					}
					if logLine.Message != *s.expected {
						t.Errorf(`expected %s got %s`, *s.expected, logLine.Message)
					}
					if s.expectedFnJson != nil {
						s.expectedFnJson(t, logLine)
					}
				}
			}
		})
	}

}
func makeBackup() *backup {
	return &backup{
		envs:   os.Environ(),
		stdout: os.Stdout,
	}
}
func restoreBackup(b *backup) {
	os.Clearenv()
	setEnvs(b.envs, "")
	os.Stdout = b.stdout
}
func setEnvs(envs []string, tempDir string) {
	for _, ss := range envs {
		sss := strings.Split(ss, "=")
		os.Setenv(sss[0], sss[1])
	}
	if tempDir != "" {
		os.Setenv(env.WorkDirVar, tempDir)
		os.Setenv(env.BaseDir(), tempDir)
	}
}
func buildContextMap(context []string) map[string]string {
	m := make(map[string]string)
	for _, c := range context {
		v := strings.Split(c, "=")
		m[v[0]] = v[1]
	}
	return m
}

func newScenario(types int,
	name string,
	envs []string,
	context []string,
	lvl zerolog.Level,
	text string,
	expected string,
	expectedFn expectedScenario,
	expexpectedScenarioJson expectedScenarioJson) scenario {
	var fileName string
	if types == typeFile {
		fileName = name + ".log"
	}
	var contextmap map[string]string = nil
	if context != nil {
		contextmap = buildContextMap(context)
	}
	return scenario{
		types:          types,
		name:           name,
		fileName:       fileName,
		envs:           envs,
		context:        contextmap,
		lvl:            lvl,
		text:           text,
		expected:       &expected,
		expectedFn:     expectedFn,
		expectedFnJson: expexpectedScenarioJson,
	}
}
func newConsoleScenario(name string,
	envs []string,
	lvl zerolog.Level,
	text string,
	expected string) scenario {
	return newScenario(0, name, envs, nil, lvl, text, expected, nil, nil)
}
func newFileScenario(name string,
	envs []string,
	lvl zerolog.Level,
	text string,
	expected string) scenario {
	return newScenario(1, name, envs, nil, lvl, text, expected, nil, nil)
}

func newConsoleScenarioWithFn(name string,
	envs []string,
	context []string,
	lvl zerolog.Level,
	text string,
	expected string,
	expectedFn expectedScenario) scenario {
	return newScenario(0, name, envs, context, lvl, text, expected, expectedFn, nil)
}
func newConsoleScenarioWithContext(name string,
	envs []string,
	context []string,
	lvl zerolog.Level,
	text string,
	expected string) scenario {
	return newScenario(0, name, envs, context, lvl, text, expected, nil, nil)
}
func newFileScenarioWithFn(name string,
	envs []string,
	lvl zerolog.Level,
	text string,
	expected string,
	expectedScenarioJsonFn expectedScenarioJson) scenario {
	return newScenario(1, name, envs, nil, lvl, text, expected, nil, expectedScenarioJsonFn)
}
func newFileScenarioWithFnAndContext(name string,
	envs []string,
	context []string,
	lvl zerolog.Level,
	text string,
	expected string,
	expectedFn expectedScenarioJson) scenario {
	return newScenario(1, name, envs, context, lvl, text, expected, nil, expectedFn)
}