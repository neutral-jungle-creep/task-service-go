package logging_test

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/pkg/logging"
)

func TestNewLogger_DefaultLevel(t *testing.T) {
	t.Parallel()

	l, err := logging.NewLogger("", "svc", "rel")
	require.NoError(t, err)
	require.NotNil(t, l)
}

func TestNewLogger_KnownLevels(t *testing.T) {
	t.Parallel()

	for _, lvl := range []string{"debug", "info", "warn", "error", "fatal"} {
		l, err := logging.NewLogger(lvl, "svc", "rel")
		require.NoError(t, err, "level %q should be accepted", lvl)
		require.NotNil(t, l)
	}
}

func TestNewLogger_UnknownLevel(t *testing.T) {
	t.Parallel()

	_, err := logging.NewLogger("trace", "svc", "rel")
	require.Error(t, err)
}

func TestLogger_NonFatalMethodsDoNotPanic(t *testing.T) {
	t.Parallel()

	l, err := logging.NewLogger("debug", "svc", "rel")
	require.NoError(t, err)

	assert.NotPanics(t, func() {
		l.Debug("d")
		l.Info("i")
		l.Warn("w")
		l.Error("e", errors.New("err"))
	})
}
