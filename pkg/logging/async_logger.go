package logging

import (
	"context"
	"runtime/debug"
)

type AsyncLogger struct {
	Logger
	ctx     context.Context
	cancel  context.CancelFunc
	logChan chan Entry
}

func NewAsyncLogger(ctx context.Context, l *Logger) *AsyncLogger {
	ctx, cancel := context.WithCancel(ctx)

	logger := &AsyncLogger{
		ctx:     ctx,
		cancel:  cancel,
		Logger:  *l,
		logChan: make(chan Entry),
	}
	return logger
}

func (l *AsyncLogger) write(e Entry) {
	l.logChan <- e
}

func (l *AsyncLogger) AsyncFatal(msg string, err error) {
	if ok := l.checkLevel(FatalLevel); !ok {
		return
	}

	e := l.newEntryWithError(FatalLevel, msg, err, debug.Stack())
	l.write(e)
}

func (l *AsyncLogger) AsyncError(msg string, err error) {
	if ok := l.checkLevel(ErrorLevel); !ok {
		return
	}

	e := l.newEntryWithError(ErrorLevel, msg, err, debug.Stack())
	l.write(e)
}

func (l *AsyncLogger) AsyncInfo(msg string) {
	if ok := l.checkLevel(InfoLevel); !ok {
		return
	}

	e := l.newEntry(InfoLevel, msg)
	l.write(e)
}

func (l *AsyncLogger) AsyncWarn(msg string) {
	if ok := l.checkLevel(WarnLevel); !ok {
		return
	}

	e := l.newEntry(WarnLevel, msg)
	l.write(e)
}

func (l *AsyncLogger) AsyncDebug(msg string) {
	if ok := l.checkLevel(DebugLevel); !ok {
		return
	}

	e := l.newEntry(DebugLevel, msg)
	l.write(e)
}

func (l *AsyncLogger) Process() error {
	for {
		select {
		case <-l.ctx.Done():
			return nil
		case e, ok := <-l.logChan:
			if ok {
				if e.Level == FatalLevel.String() {
					l.core.Fatalf("%+v", e)
				}
				l.core.Printf("%+v", e)
			}
		}
	}
}

func (l *AsyncLogger) Stop() {
	l.cancel()
}
