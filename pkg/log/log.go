package log

import (
	"io"
	"os"
	"path"
	"strings"
	"time"

	"github.com/andrescosta/goico/pkg/env"

	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type (
	Config struct {
		Console
		File
		Level  zerolog.Level
		Caller bool
	}

	Console struct {
		Enabled          bool
		ExcludeTimestamp bool
	}

	File struct {
		Enabled          bool
		EncodeLogsAsJson bool
		Directory        string
		Name             string
		MaxSize          int
		MaxBackups       int
		MaxAge           int
	}
)

func NewUsingEnv() *zerolog.Logger {
	var empty map[string]string
	return NewUsingEnvAndValues(empty)
}
func NewUsingEnvAndValues(values map[string]string) *zerolog.Logger {
	config := Config{
		Console: Console{
			Enabled:          env.EnvAsBool("log.console.enabled", true),
			ExcludeTimestamp: env.EnvAsBool("log.console.exclude.timestamp", false),
		},
		Level:  env.EnvAsInt("log.level", zerolog.InfoLevel),
		Caller: env.EnvAsBool("log.caller", false),
		File: File{
			Enabled:          env.EnvAsBool("log.file.enabled", false),
			EncodeLogsAsJson: env.EnvAsBool("log.file.json", false),
			Directory:        env.Env("log.file.dir", ".\\log"),
			Name:             env.Env("log.file.name", "file.log"),
			MaxSize:          env.EnvAsInt("log.file.max.size", 100),
			MaxBackups:       env.EnvAsInt("log.file.max.backups", 10),
			MaxAge:           env.EnvAsInt("log.file.max.age", 24),
		},
	}
	return New(values, config)
}

func New(values map[string]string, config Config) *zerolog.Logger {
	var writers []io.Writer

	if config.Console.Enabled {
		writers = append(writers, configureLogToConsole(config.Console))
	}
	if config.File.Enabled && strings.TrimSpace(config.File.Name) != "" {
		writers = append(writers, configureLogToFile(config.File))
	}
	mw := io.MultiWriter(writers...)

	zerolog.SetGlobalLevel(config.Level)

	ctx := zerolog.New(mw).With().Timestamp()

	if config.Caller {
		ctx = ctx.Caller()
	}

	for k, v := range values {
		ctx = ctx.Str(k, v)
	}

	logger := ctx.Logger()

	return &logger
}

func configureLogToConsole(config Console) (writer io.Writer) {
	writer = zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) {
			w.Out = os.Stdout
			w.TimeFormat = time.RFC3339
			if config.ExcludeTimestamp {
				w.PartsExclude = []string{zerolog.TimestampFieldName}
			}
		},
	)
	return
}

func configureLogToFile(config File) (writer io.Writer) {
	return configureLumberjack(config)
}

func configureLumberjack(config File) (writer io.Writer) {
	writer = &lumberjack.Logger{
		Filename:   path.Join(config.Directory, config.Name),
		MaxBackups: config.MaxBackups,
		MaxSize:    config.MaxSize,
		MaxAge:     config.MaxAge,
	}
	return
}
