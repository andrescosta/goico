package log

import (
	"io"
	"os"
	"strings"
	"time"

	"github.com/andrescosta/goico/pkg/env"
	"github.com/rs/zerolog"
	"gopkg.in/natefinch/lumberjack.v2"
)

type (
	config struct {
		console
		file
		Level  zerolog.Level
		Caller bool
	}
	console struct {
		Enabled          bool
		ExcludeTimestamp bool
	}
	file struct {
		Enabled    bool
		Directory  string
		Name       string
		MaxSize    int
		MaxBackups int
		MaxAge     int
	}
)

func New() (*zerolog.Logger, io.WriteCloser) {
	var empty map[string]string
	return NewWithContext(empty)
}

func NewWithContext(ctxInfo map[string]string) (*zerolog.Logger, io.WriteCloser) {
	cfg := config{
		console: console{
			Enabled:          env.Bool("log.console.enabled", true),
			ExcludeTimestamp: env.Bool("log.console.exclude.timestamp", false),
		},
		Level:  env.Int("log.level", zerolog.InfoLevel),
		Caller: env.Bool("log.caller", false),
		file: file{
			Enabled:    env.Bool("log.file.enabled", false),
			Name:       getFileName(),
			MaxSize:    env.Int("log.file.max.size", 100),
			MaxBackups: env.Int("log.file.max.backups", 10),
			MaxAge:     env.Int("log.file.max.age", 24),
		},
	}
	return newLogger(ctxInfo, cfg)
}

func getFileName() (name string) {
	name = env.String("log.file.name", "file.log")
	name = strings.Replace(name, "${workdir}", env.Workdir(), 1)
	return
}

func newLogger(ctxInfo map[string]string, cfg config) (*zerolog.Logger, io.WriteCloser) {
	var writers []io.Writer
	var writerCloser io.WriteCloser
	if cfg.console.Enabled {
		writers = append(writers, configureLogToConsole(cfg.console))
	}
	if cfg.file.Enabled && strings.TrimSpace(cfg.file.Name) != "" {
		writerCloser = lumberjackLogger(cfg.file)
		writers = append(writers, writerCloser)
	}
	level := cfg.Level
	if len(writers) == 0 {
		level = zerolog.Disabled
		writers = append(writers, io.Discard)
	}
	mw := io.MultiWriter(writers...)
	ctx := zerolog.New(mw).With().Timestamp()
	if cfg.Caller {
		ctx = ctx.Caller()
	}
	for k, v := range ctxInfo {
		ctx = ctx.Str(k, v)
	}
	logger := ctx.Logger().Level(level)
	return &logger, writerCloser
}

func configureLogToConsole(cfg console) (writerc io.Writer) {
	writerc = zerolog.NewConsoleWriter(
		func(w *zerolog.ConsoleWriter) {
			w.Out = os.Stdout
			w.TimeFormat = time.RFC3339
			if cfg.ExcludeTimestamp {
				w.PartsExclude = []string{zerolog.TimestampFieldName}
			}
		},
	)
	return
}

func lumberjackLogger(cfg file) io.WriteCloser {
	return &lumberjack.Logger{
		Filename:   cfg.Name,
		MaxBackups: cfg.MaxBackups,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
	}
}

func DisableLog() {
	zerolog.SetGlobalLevel(zerolog.Disabled)
}
