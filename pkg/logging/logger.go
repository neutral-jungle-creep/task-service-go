package logging

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"
)

type Config struct {
	LogLevel string `json:"logLevel"`
}

const defaultLogLevel = "info"

type level uint8

func (l level) String() string {
	var lvl string

	switch l {
	case 0:
		lvl = "debug"
	case 1:
		lvl = "info"
	case 2:
		lvl = "warn"
	case 3:
		lvl = "error"
	case 4:
		lvl = "fatal"
	}
	return lvl
}

func parseLogLevel(lvl string) (level, error) {
	var parsedLvl level
	var err error

	switch lvl {
	case "debug":
		parsedLvl = 0
	case "info":
		parsedLvl = 1
	case "warn":
		parsedLvl = 2
	case "error":
		parsedLvl = 3
	case "fatal":
		parsedLvl = 4
	default:
		err = errors.New("invalid log level")
	}
	return parsedLvl, err
}

const (
	DebugLevel level = iota
	InfoLevel
	WarnLevel
	ErrorLevel
	FatalLevel
)

type Logger struct {
	config config
	core   *log.Logger
}

type config struct {
	Level  level
	Fields map[string]any
}

type Entry struct {
	Level   string
	Time    time.Time
	Message string
	Stack   string
	Fields  map[string]any
}

func (l *Logger) newEntry(lvl level, msg string) Entry {
	return Entry{
		Level:   lvl.String(),
		Time:    time.Now(),
		Message: msg,
		Fields:  l.config.Fields,
	}
}

func (l *Logger) newEntryWithError(lvl level, msg string, err error, stack []byte) Entry {
	return Entry{
		Level:   lvl.String(),
		Time:    time.Now(),
		Message: fmt.Sprintf("%s: %s", msg, err.Error()),
		Stack:   string(stack),
		Fields:  l.config.Fields,
	}
}

func NewLogger(logLevel, serviceName, releaseID string) (*Logger, error) {
	if logLevel == "" {
		logLevel = defaultLogLevel
	}

	parsedLogLevel, err := parseLogLevel(logLevel)
	if err != nil {
		return nil, err
	}

	cfg := config{
		Level: parsedLogLevel,
		Fields: map[string]any{
			"serviceName": serviceName,
			"releaseId":   releaseID,
		},
	}

	logger := &Logger{
		core:   log.New(os.Stderr, "", log.LstdFlags),
		config: cfg,
	}

	return logger, nil
}

func (l *Logger) checkLevel(lvl level) bool {
	if l.config.Level > lvl {
		return false
	}
	return true
}

func (l *Logger) Fatal(msg string, err error) {
	if ok := l.checkLevel(FatalLevel); !ok {
		return
	}
	e := l.newEntryWithError(FatalLevel, msg, err, debug.Stack())
	l.core.Fatalf("%+v", e)
}

func (l *Logger) Error(msg string, err error) {
	if ok := l.checkLevel(ErrorLevel); !ok {
		return
	}

	e := l.newEntryWithError(ErrorLevel, msg, err, debug.Stack())
	l.core.Printf("%+v", e)
}

func (l *Logger) Info(msg string) {
	if ok := l.checkLevel(InfoLevel); !ok {
		return
	}

	e := l.newEntry(InfoLevel, msg)
	l.core.Printf("%+v", e)
}

func (l *Logger) Warn(msg string) {
	if ok := l.checkLevel(WarnLevel); !ok {
		return
	}

	e := l.newEntry(WarnLevel, msg)
	l.core.Printf("%+v", e)
}

func (l *Logger) Debug(msg string) {
	if ok := l.checkLevel(DebugLevel); !ok {
		return
	}

	e := l.newEntry(DebugLevel, msg)
	l.core.Printf("%+v", e)
}
