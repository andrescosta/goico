package env_test

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/andrescosta/goico/pkg/collection"
	"github.com/andrescosta/goico/pkg/env"
	"github.com/andrescosta/goico/pkg/test"
)

type envvs []envv

// 1 - Test overloading logic
// Combinations:
//    1.1 Arg, .env
//    1.2 Arg, locals

// default: development
// others: production, test
// .env.development.local
// .env.test.local
// .env.production.local
// .env.local
// .env.development
// .env.test
// .env.production
// .env.${service_name}
// .env

type envv struct {
	name  string
	value any
}

const serviceName = "mytests"

var (
	envm = map[string]envvs{
		".env.development.local": {
			{"svc.addr", "localhost:8985"},
			{"svc.db", "localhost:9995"},
			{"svc.q", "localhost:7970"},
		},
		".env.test.local": {
			{"svc.addr", "localhost:8885"},
			{"svc.db", "localhost:9895"},
			{"svc.q", "localhost:7870"},
		},
		".env.production.local": {
			{"svc.addr", "localhost:8785"},
			{"svc.db", "localhost:9795"},
			{"svc.q", "localhost:7770"},
		},
		".env.local": {
			{"svc.addr", "localhost:8685"},
			{"svc.db", "localhost:9695"},
			{"svc.q", "localhost:7670"},
		},
		".env.development": {
			{"svc.addr", "localhost:8588"},
			{"svc.db", "localhost:9598"},
			{"svc.queue", "localhost:7078"},
			{"timeout", 19 * time.Second},
			{"retries", int32(987)},
			{"bignumber", int64(9909000)},
			{"metrics", false},
			{"servers", "localhost:1,localhost:2,localhost:3"},
		},
		".env.test": {
			{"svc.addr", "localhost:8587"},
			{"svc.db", "localhost:9597"},
			{"svc.queue", "localhost:7077"},
			{"timeout", 9 * time.Second},
			{"retries", int32(1)},
			{"bignumber", int64(990000)},
			{"metrics", false},
			{"servers", "localhost:4,localhost:5,localhost:6"},
		},
		".env.production": {
			{"svc.addr", "localhost:8586"},
			{"svc.db", "localhost:9596"},
			{"svc.queue", "localhost:7076"},
			{"timeout", 6 * time.Second},
			{"retries", int32(10000)},
			{"bignumber", int64(100000)},
			{"metrics", true},
			{"servers", "localhost:7,localhost:8,localhost:9"},
		},
		".env." + serviceName: {
			{"svc.addr", "localhost:8581"},
			{"svc.db", "localhost:9591"},
			{"svc.queue", "localhost:7071"},
			{"timeout", 60 * time.Second},
			{"retries", int32(10001)},
			{"bignumber", int64(10090000)},
			{"metrics", false},
			{"servers", "localhost:10,localhost:11,localhost:12"},
		},
		".env": {
			{"svc.addr", "localhost:8585"},
			{"svc.db", "localhost:9595"},
			{"svc.queue", "localhost:7070"},
			{"timeout", 5 * time.Second},
			{"retries", int32(100)},
			{"bignumber", int64(100)},
			{"metrics", true},
			{"servers", "localhost:13,localhost:14,localhost:15"},
		},
	}

	invalids = envvs{
		{"timeout_i", "a890"},
		{"retries_i", "ret2"},
		{"bignumber_i", "big3"},
		{"metrics_i", "illegal"},
	}

	argsEqualToDotEnv = envvs{
		{"svc.addr", "localhost:8585"},
		{"svc.db", "localhost:9595"},
		{"svc.queue", "localhost:7575"},
		{"timeout", 155 * time.Second},
		{"retries", int32(1000)},
		{"bignumber", int64(1000)},
		{"metrics", false},
		{"servers", "localhost:16,localhost:17,localhost:18"},
	}

	argsLessThanDotEnv = envvs{
		{"svc.addr", "localhost:8585"},
		{"svc.db", "localhost:9595"},
	}
)

type (
	scenario struct {
		name        string
		environment string
		files       []string
		args        envvs
		expected    envvs
	}
)

func TestLoad(t *testing.T) {
	scenarios := []scenario{
		{
			"os.Args plus .env", "",
			[]string{".env"},
			argsEqualToDotEnv,
			argsEqualToDotEnv,
		},
		{
			"os.Args plus .env (partial)", "",
			[]string{".env"},
			argsLessThanDotEnv,
			merge(argsLessThanDotEnv, envm[".env"]),
		},
		{
			".env for default", "",
			[]string{".env"},
			nil,
			envm[".env"],
		},
		{
			".env for dev", env.Development,
			[]string{".env"},
			nil,
			envm[".env"],
		},
		{
			".env for prod ", env.Production,
			[]string{".env"},
			nil,
			envm[".env"],
		},
		{
			".env for test", env.Test,
			[]string{".env"},
			nil, envm[".env"],
		},
		{
			".env.development", env.Development,
			[]string{".env.development"},
			nil,
			envm[".env.development"],
		},
		{
			".env.production", env.Production,
			[]string{".env.production"},
			nil,
			envm[".env.production"],
		},
		{
			".env.test", env.Test,
			[]string{".env.test"},
			nil,
			envm[".env.test"],
		},
		{
			".env." + serviceName, "",
			[]string{".env." + serviceName},
			nil,
			envm[".env."+serviceName],
		},
		{
			".env." + serviceName, env.Development,
			[]string{".env." + serviceName},
			nil,
			envm[".env."+serviceName],
		},
		{
			".env." + serviceName, env.Production,
			[]string{".env." + serviceName},
			nil,
			envm[".env."+serviceName],
		},
		{
			".env." + serviceName + "plus .env", env.Production,
			[]string{".env", ".env." + serviceName},
			nil,
			mergeEnvs(".env."+serviceName, ".env"),
		},
		{
			".env." + serviceName, env.Test,
			[]string{".env." + serviceName},
			nil,
			envm[".env."+serviceName],
		},
		{
			".env plus .env.local", env.Development,
			[]string{".env", ".env.local"},
			nil,
			mergeEnvs(".env.local", ".env"),
		},
		{
			".env plus .env.local for test", env.Test, // env.local is loaded in test
			[]string{".env", ".env.local"},
			nil,
			envm[".env"],
		},
		{
			".env plus .env.development.local", env.Development,
			[]string{".env", ".env.development.local"},
			nil,
			mergeEnvs(".env.development.local", ".env"),
		},
		{
			".env plus .env.production.local", env.Production,
			[]string{".env", ".env.production.local"},
			nil,
			mergeEnvs(".env.production.local", ".env"),
		},
		{
			".env plus .env.test.local", env.Test,
			[]string{".env", ".env.test.local"},
			nil,
			mergeEnvs(".env.test.local", ".env"),
		},
		{
			".env.development plus .env.development.local", env.Development,
			[]string{".env.development", ".env.development.local"},
			nil,
			mergeEnvs(".env.development.local", ".env.development"),
		},
		{
			".env.production plus .env.production.local", env.Production,
			[]string{".env.production", ".env.production.local"},
			nil,
			mergeEnvs(".env.production.local", ".env.production"),
		},
		{
			".env.test plus .env.test.local", env.Test,
			[]string{".env.test", ".env.test.local"},
			nil,
			mergeEnvs(".env.test.local", ".env.test"),
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			initializeScenario(t, s)
			loaded, err := env.Load(serviceName)
			if !loaded && err == nil {
				t.Error(".env files were not loaded")
			}
			if loaded && err != nil {
				t.Errorf("loaded True but env.Load error: %v", err)
				return
			}
			test.Nil(t, err)
			for _, en := range s.expected {
				assertValue(t, en)
			}
		})
	}
}

func TestLoadErrors(t *testing.T) {
	b := env.Backup()
	t.Cleanup(func() {
		env.Restore(b)
	})
	os.Setenv(env.EnviromentVar, "nope")
	loaded, err := env.Load(serviceName)
	if loaded {
		t.Error("expected not loaded got loaded")
	}
	if err == nil {
		t.Error("expected error got <nil>")
	}
	if err := os.Setenv(env.EnviromentVar, env.Development); err != nil {
		t.Errorf("not expected error got %v", err)
	}
	if loaded {
		t.Errorf("not expected to load any file")
	}
}

func TestDirs(t *testing.T) {
	b := env.Backup()
	t.Cleanup(func() {
		env.Restore(b)
	})
	tempDir := t.TempDir()
	initializeDirVars(tempDir)
	workDir := env.Workdir()
	if workDir != tempDir {
		t.Errorf("work dir is different %s - %s", tempDir, workDir)
	}
	baseDir := env.Basedir()
	if baseDir != tempDir {
		t.Errorf("base dir is different %s - %s", tempDir, workDir)
	}
	m := env.WorkdirPlus("mydir")
	l := filepath.Join(tempDir, "mydir")
	if m != l {
		t.Errorf("dirs are different %s - %s", m, l)
	}
}

func TestDefault(t *testing.T) {
	scenarios := []scenario{
		{
			"os.Args plus .env", "",
			[]string{".env"},
			argsEqualToDotEnv, nil,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			initializeScenario(t, s)

			_, err := env.Load(serviceName)
			test.Nil(t, err)
			v := env.String("myserver1", "localhost:7890")
			if v != "localhost:7890" {
				t.Errorf("expected localhost:7890 got %s", v)
			}
			bo := env.Bool("errors1", false)
			if bo {
				t.Errorf("expected false got %t", bo)
			}
			du := env.Duration("timeout1", 16*time.Second)
			if *du != 16*time.Second {
				t.Errorf("expected 16s got %s", du)
			}
			se := env.StringOrNil("server2")
			if se != nil {
				t.Errorf("expected <nil> got %s", *se)
			}
			in := env.Int("retries1", 99)
			if in != 99 {
				t.Errorf("expected 99 got %d", in)
			}
			svrs := collection.NewSet("1", "2", "3")
			savs := env.Array("serverss", "1,2,3")
			for _, sav := range savs {
				if !svrs.Has(sav) {
					t.Errorf("not expected %s", sav)
				}
			}
		})
	}
}

func TestInvalid(t *testing.T) {
	e := envm[".env"]
	delete(envm, ".env")
	envm[".env"] = invalids
	scenarios := []scenario{
		{
			".env invalid", "",
			[]string{".env"},
			argsEqualToDotEnv, nil,
		},
	}

	for _, s := range scenarios {
		t.Run(s.name, func(t *testing.T) {
			initializeScenario(t, s)
			t.Cleanup(func() {
				delete(envm, ".env")
				envm[".env"] = e
			})
			loaded, err := env.Load(serviceName)
			if !loaded && err == nil {
				t.Errorf("files not loaded")
			}
			test.Nil(t, err)
			bo := env.Bool("metrics_i", true)
			if !bo {
				t.Errorf("expected invalid value got %t", bo)
			}
			bo = env.Bool("metrics_i")
			if bo {
				t.Errorf("expected false got %t", bo)
			}
			du := env.Duration("timeout_i", 89*time.Second)
			if *du != 89*time.Second {
				t.Errorf("expected 89s got %s", du)
			}
			du = env.Duration("timeout_i")
			if du != nil {
				t.Errorf("expected <nil> got %s", du)
			}
			in := env.Int("retries_i", 999)
			if in != 999 {
				t.Errorf("expected 999 got %d", in)
			}
			in = env.Int[int]("retries_i")
			if in != 0 {
				t.Errorf("expected 0 got %d", in)
			}
			in = env.Int("bignumber_i", 9999)
			if in != 9999 {
				t.Errorf("expected 9999 got %d", in)
			}
		})
	}
}

func initializeScenario(t *testing.T, s scenario) {
	b := env.Backup()
	t.Cleanup(func() {
		env.Restore(b)
	})

	tempDir := t.TempDir()
	initializeDirVars(tempDir)

	if s.environment != "" {
		os.Setenv(env.EnviromentVar, s.environment)
	}
	if s.args != nil {
		for idx, e := range s.args {
			pref := "--env"
			if idx%2 == 0 {
				pref = "-e"
			}
			os.Args = append(os.Args, fmt.Sprintf("%s:%s", pref, e.string()))
		}
	}

	createEnvFiles(t, tempDir, s.files)
}

func initializeDirVars(tempDir string) {
	os.Setenv(env.BaseDirVar, tempDir)
	os.Setenv(env.WorkDirVar, tempDir)
}

func assertValue(t *testing.T, en envv) {
	switch en.value.(type) {
	case string:
		val := en.value.(string)
		if !strings.Contains(val, ",") {
			v := env.String(en.name)
			if v != val {
				t.Errorf(fmt.Sprintf("expecting for %s(string): %s got %s", en.name, en.value, val))
			}
			return
		}
		va2 := strings.Split(val, ",")
		va1 := env.Array(en.name, "")
		slices.Sort(va1)
		slices.Sort(va2)
		if !slices.Equal(va1, va2) {
			t.Errorf(fmt.Sprintf("expecting for %s(string): %s got %s", en.name, en.value, val))
		}
	case bool:
		v := env.Bool(en.name)
		if v != en.value {
			t.Errorf(fmt.Sprintf("expecting for %s(bool): %t got %t", en.name, en.value, v))
		}
	case int32:
		v := env.Int[int32](en.name)
		if v != en.value {
			t.Errorf(fmt.Sprintf("expecting for %s(int32): %d got %d", en.name, en.value, v))
		}
	case int64:
		v := env.Int[int64](en.name)
		if v != en.value {
			t.Errorf(fmt.Sprintf("expecting for %s(int64): %d got %d", en.name, en.value, v))
		}
	case time.Duration:
		v := env.Duration(en.name)
		if v == nil || *v != en.value.(time.Duration) {
			t.Errorf(fmt.Sprintf("expecting for %s(time.Duration): %s got %s", en.name, en.value, v))
		}
	}
}

func createEnvFiles(t *testing.T, dir string, files []string) {
	t.Helper()
	for _, f := range files {
		createEnvFile(t, dir, f, envm[f])
	}
}

func createEnvFile(t *testing.T, dir string, name string, e envvs) {
	t.Helper()
	file := filepath.Join(dir, name)
	err := os.WriteFile(file, e.bytes(t),
		os.ModeAppend)
	test.Nil(t, err, fmt.Sprintf("os.WriteFile: error writing file %s:%s", file, err))
}

func (e envvs) bytes(t *testing.T) []byte {
	t.Helper()
	var v bytes.Buffer
	for _, ee := range e {
		s := fmt.Sprintln(ee.string())
		_, err := v.WriteString(s)
		test.Nil(t, err)
	}
	return v.Bytes()
}

func (e envv) string() string {
	s := fmt.Sprintf("%s=", e.name)
	switch e.value.(type) {
	case string, time.Duration:
		s += fmt.Sprintf("%s", e.value)
	case bool:
		s += fmt.Sprintf("%t", e.value)
	case int32, int64:
		s += fmt.Sprintf("%d", e.value)
	}
	return s
}

func mergeEnvs(e1 string, e2 string) envvs {
	return merge(envm[e1], envm[e2])
}

func merge(e1 envvs, e2 envvs) envvs {
	es := make(envvs, 0)
	set := collection.NewSetFn(e1, func(v envv) string { return v.name })
	es = append(es, e1...)
	for _, e := range e2 {
		if !set.Has(e.name) {
			es = append(es, e)
		}
	}
	return es
}
