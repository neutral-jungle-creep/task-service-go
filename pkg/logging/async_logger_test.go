package logging_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/pkg/logging"
)

func TestAsyncLogger_StopUnblocksProcess(t *testing.T) {
	t.Parallel()

	core, err := logging.NewLogger("debug", "svc", "rel")
	require.NoError(t, err)

	async := logging.NewAsyncLogger(context.Background(), core)

	done := make(chan error, 1)
	go func() { done <- async.Process() }()

	async.Stop()

	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(2 * time.Second):
		t.Fatal("Process did not exit after Stop")
	}
}

func TestAsyncLogger_FlushesEntries(t *testing.T) {
	t.Parallel()

	core, err := logging.NewLogger("debug", "svc", "rel")
	require.NoError(t, err)

	async := logging.NewAsyncLogger(context.Background(), core)
	go func() { _ = async.Process() }()

	assert.NotPanics(t, func() {
		async.AsyncDebug("debug")
		async.AsyncInfo("info")
		async.AsyncWarn("warn")
	})

	async.Stop()
}
