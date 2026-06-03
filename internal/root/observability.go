package root

import (
	"task-service/pkg/logging"
)

func (r *Root) initObservability(logger *logging.Logger) {
	r.logger = logging.NewAsyncLogger(
		r.ctx,
		logger,
	)

	r.RegisterBackgroundJob(func() error {
		return r.logger.Process()
	})
	r.RegisterStopHandler(func() error {
		r.logger.Stop()
		return nil
	})
}
