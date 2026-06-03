package background_test

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/pkg/background"
)

func TestRegistrar_Run_NoJobsBlocksUntilCtxCancel(t *testing.T) {
	t.Parallel()

	r := background.New()

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- r.Run(ctx) }()

	// Nothing should fire yet.
	select {
	case <-done:
		t.Fatal("Run returned before ctx was cancelled")
	case <-time.After(50 * time.Millisecond):
	}

	cancel()
	select {
	case err := <-done:
		require.NoError(t, err)
	case <-time.After(time.Second):
		t.Fatal("Run did not return after ctx cancel")
	}
}

func TestRegistrar_Run_ReturnsFirstJobError(t *testing.T) {
	t.Parallel()

	r := background.New()
	r.RegisterJob(func() error {
		time.Sleep(20 * time.Millisecond) // ensures the failing job wins the race
		return nil
	})
	want := errors.New("kaboom")
	r.RegisterJob(func() error { return want })

	err := r.Run(context.Background())
	require.ErrorIs(t, err, want)
}

func TestRegistrar_Run_CtxCancelWinsOverPendingJobs(t *testing.T) {
	t.Parallel()

	r := background.New()
	r.RegisterJob(func() error {
		time.Sleep(time.Second) // never finishes before ctx cancel
		return errors.New("never")
	})

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	err := r.Run(ctx)
	require.NoError(t, err, "ctx cancel should beat the slow job")
}

func TestRegistrar_Stop_RunsAllHandlersInParallel(t *testing.T) {
	t.Parallel()

	r := background.New()
	var ran int32

	const n = 5
	for i := 0; i < n; i++ {
		r.RegisterStopHandler(func() error {
			time.Sleep(50 * time.Millisecond)
			atomic.AddInt32(&ran, 1)
			return nil
		})
	}

	start := time.Now()
	require.NoError(t, r.Stop())
	elapsed := time.Since(start)

	assert.Equal(t, int32(n), atomic.LoadInt32(&ran), "all handlers must run")
	// If they ran sequentially elapsed would be >= n*50ms = 250ms.
	// Parallel execution should finish in well under 200ms.
	assert.Less(t, elapsed, 200*time.Millisecond, "handlers must run in parallel")
}

func TestRegistrar_Stop_NoHandlersIsNoOp(t *testing.T) {
	t.Parallel()

	r := background.New()
	assert.NotPanics(t, func() { _ = r.Stop() })
}

func TestRegistrar_Stop_JoinsErrors(t *testing.T) {
	t.Parallel()

	r := background.New()
	errA := errors.New("close A failed")
	errB := errors.New("close B failed")
	r.RegisterStopHandler(func() error { return errA })
	r.RegisterStopHandler(func() error { return nil })
	r.RegisterStopHandler(func() error { return errB })

	err := r.Stop()
	require.ErrorIs(t, err, errA)
	require.ErrorIs(t, err, errB)
}

func TestRegistrar_ConcurrentRegistrationIsSafe(t *testing.T) {
	t.Parallel()

	r := background.New()
	var wg sync.WaitGroup
	const goroutines = 20
	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			r.RegisterJob(func() error { return nil })
			r.RegisterStopHandler(func() error { return nil })
		}()
	}
	wg.Wait()

	// Tear-down should not blow up.
	_ = r.Stop()
}
