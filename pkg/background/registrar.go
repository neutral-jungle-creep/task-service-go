// Package background registers long-lived jobs and shutdown handlers so an
// application can start them in parallel and tear them down in parallel.
package background

import (
	"context"
	"sync"
)

// Job is a long-running background task. Returning a non-nil error from any
// job causes Registrar.Run to unblock with that error.
type Job func() error

// StopHandler runs during graceful shutdown — typically calls Close/Shutdown
// on a resource registered earlier. Errors from StopHandler are intentionally
// swallowed: shutdown has nothing useful to do with them at the call site.
type StopHandler func()

// Registrar collects Jobs and StopHandlers and runs them as a group.
// Safe for concurrent registration; Run/Stop are intended to be called once.
type Registrar struct {
	mu    sync.Mutex
	jobs  []Job
	stops []StopHandler
}

func New() *Registrar {
	return &Registrar{}
}

func (r *Registrar) RegisterJob(j Job) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.jobs = append(r.jobs, j)
}

func (r *Registrar) RegisterStopHandler(h StopHandler) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.stops = append(r.stops, h)
}

// Run starts every registered Job in its own goroutine and blocks until
// either:
//   - ctx is cancelled — returns nil;
//   - the first Job reports a non-nil error — returns that error.
//
// Run does NOT call Stop; callers wire that up via `defer r.Stop()` so they
// can choose the exact teardown moment.
func (r *Registrar) Run(ctx context.Context) error {
	r.mu.Lock()
	jobs := append([]Job(nil), r.jobs...) // snapshot, decouples from later registrations
	r.mu.Unlock()

	// buffered so any job can finish and report without blocking on send even
	// after Run has already returned via ctx.Done().
	errs := make(chan error, len(jobs))
	for _, j := range jobs {
		go func(job Job) {
			errs <- job()
		}(j)
	}

	select {
	case <-ctx.Done():
		return nil
	case err := <-errs:
		return err
	}
}

// Stop fires every registered StopHandler in parallel and blocks until all
// have returned. Safe to call after Run.
func (r *Registrar) Stop() {
	r.mu.Lock()
	stops := append([]StopHandler(nil), r.stops...)
	r.mu.Unlock()

	var wg sync.WaitGroup
	wg.Add(len(stops))
	for _, h := range stops {
		go func(handler StopHandler) {
			defer wg.Done()
			handler()
		}(h)
	}
	wg.Wait()
}
