package root

import (
	"task-service/pkg/logging"
)

func (r *Root) initObservability(logger *logging.Logger) error {
	asyncLogger := logging.NewAsyncLogger(
		r.ctx,
		logger,
	)

	r.RegisterBackgroundJob(func() error {
		return asyncLogger.Process()
	})
	r.RegisterStopHandler(func() {
		asyncLogger.Stop()
	})
	return nil
}
