package log

import (
	"io"
	"log"
	"os"
	"path"
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
		Enabled          bool
		EncodeLogsAsJSON bool
		Directory        string
		Name             string
		MaxSize          int
		MaxBackups       int
		MaxAge           int
	}
)

func New() *zerolog.Logger {
	var empty map[string]string
	return NewWithContext(empty)
}
func NewWithContext(ctxInfo map[string]string) *zerolog.Logger {
	cfg := config{
		console: console{
			Enabled:          env.AsBool("log.console.enabled", true),
			ExcludeTimestamp: env.AsBool("log.console.exclude.timestamp", false),
		},
		Level:  env.AsInt("log.level", zerolog.InfoLevel),
		Caller: env.AsBool("log.caller", false),
		file: file{
			Enabled:          env.AsBool("log.file.enabled", false),
			EncodeLogsAsJSON: env.AsBool("log.file.JSON", false),
			Directory:        env.Env("log.file.dir", ".\\log"),
			Name:             env.Env("log.file.name", "file.log"),
			MaxSize:          env.AsInt("log.file.max.size", 100),
			MaxBackups:       env.AsInt("log.file.max.backups", 10),
			MaxAge:           env.AsInt("log.file.max.age", 24),
		},
	}
	return newLogger(ctxInfo, cfg)
}

func newLogger(ctxInfo map[string]string, cfg config) *zerolog.Logger {
	var writers []io.Writer
	if cfg.console.Enabled {
		writers = append(writers, configureLogToConsole(cfg.console))
	}
	if cfg.file.Enabled && strings.TrimSpace(cfg.file.Name) != "" {
		writers = append(writers, configureLogToFile(cfg.file))
	}
	level := cfg.Level
	if len(writers) == 0 {
		log.Println("Console and file loggers disabled. Logging is not enabled.")
		level = zerolog.Disabled
		writers = append(writers, io.Discard)
	}
	mw := io.MultiWriter(writers...)
	zerolog.SetGlobalLevel(level)
	ctx := zerolog.New(mw).With().Timestamp()
	if cfg.Caller {
		ctx = ctx.Caller()
	}
	for k, v := range ctxInfo {
		ctx = ctx.Str(k, v)
	}
	logger := ctx.Logger()
	return &logger
}

func configureLogToConsole(cfg console) (writer io.Writer) {
	writer = zerolog.NewConsoleWriter(
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

func configureLogToFile(cfg file) (writer io.Writer) {
	return configureLumberjack(cfg)
}

func configureLumberjack(cfg file) (writer io.Writer) {
	writer = &lumberjack.Logger{
		Filename:   path.Join(cfg.Directory, cfg.Name),
		MaxBackups: cfg.MaxBackups,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
	}
	return
}
