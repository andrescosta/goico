package log

import (
	"io"
	"log"
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

var writer *lumberjack.Logger

func Close() error {
	if writer != nil {
		return writer.Close()
	}
	return nil
}

func Luberjack() *lumberjack.Logger {
	return writer
}

func New() *zerolog.Logger {
	var empty map[string]string
	return NewWithContext(empty)
}

func NewWithContext(ctxInfo map[string]string) *zerolog.Logger {
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
	name = strings.Replace(name, "${workdir}", env.WorkDir(), 1)
	return
}

func newLogger(ctxInfo map[string]string, cfg config) *zerolog.Logger {
	var writers []io.Writer
	if cfg.console.Enabled {
		writers = append(writers, configureLogToConsole(cfg.console))
	}
	if cfg.file.Enabled && strings.TrimSpace(cfg.file.Name) != "" {
		setLogToFile(cfg.file)
		writers = append(writers, writer)
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

func setLogToFile(cfg file) {
	setLumberjack(cfg)
}

func setLumberjack(cfg file) {
	writer = &lumberjack.Logger{
		Filename:   cfg.Name,
		MaxBackups: cfg.MaxBackups,
		MaxSize:    cfg.MaxSize,
		MaxAge:     cfg.MaxAge,
	}
}
