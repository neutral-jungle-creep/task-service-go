package root

import (
	"task-service/pkg/logging"
)

func (r *Root) initObservability(logger *logging.Logger) error {
	r.logger = logging.NewAsyncLogger(
		r.ctx,
		logger,
	)

	r.RegisterBackgroundJob(func() error {
		return r.logger.Process()
	})
	r.RegisterStopHandler(func() {
		r.logger.Stop()
	})
	return nil
}
