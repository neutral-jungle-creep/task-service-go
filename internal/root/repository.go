package root

import (
	"task-service/internal/adapters/repositories"
	"task-service/pkg/postgres"
)

func (r *Root) initRepositories() error {
	db, err := postgres.New(r.ctx, postgres.Config{
		DSN:             r.config.Database.DSN,
		MaxOpenConns:    r.config.Database.MaxOpenConns,
		MaxIdleConns:    r.config.Database.MaxIdleConns,
		ConnMaxLifetime: r.config.Database.ConnMaxLifetime,
	})
	if err != nil {
		return err
	}

	r.RegisterStopHandler(func() error { return db.Close() })

	r.repositories.taskRepository = repositories.NewTaskRepository(db)
	return nil
}
