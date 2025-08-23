package logging

import (
	"errors"
	"fmt"
	"log"
	"os"
	"runtime/debug"
	"time"
)

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
}

func NewLogger(logLevel, serviceName, releaseID string) (*Logger, error) {
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

	logger := Logger{
		core:   log.New(os.Stderr, "", log.LstdFlags),
		config: cfg,
	}

	return &logger, nil
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

	e := Entry{
		Level:   FatalLevel.String(),
		Time:    time.Now(),
		Message: fmt.Sprintf("%s: %s", msg, err.Error()),
		Stack:   string(debug.Stack()),
	}
	l.core.Fatal(e, l.config.Fields)
}
