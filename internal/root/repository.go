package root

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"

	"task-service/internal/adapters/repositories"
)

const pingTimeout = 5 * time.Second

func (r *Root) initRepositories() error {
	db, err := sql.Open("pgx", r.config.Database.DSN)
	if err != nil {
		return fmt.Errorf("open postgres: %w", err)
	}

	db.SetMaxOpenConns(r.config.Database.MaxOpenConns)
	db.SetMaxIdleConns(r.config.Database.MaxIdleConns)
	db.SetConnMaxLifetime(r.config.Database.ConnMaxLifetime)

	pingCtx, cancel := context.WithTimeout(r.ctx, pingTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return fmt.Errorf("ping postgres: %w", err)
	}

	r.RegisterStopHandler(func() { _ = db.Close() })

	r.repositories.taskRepository = repositories.NewTaskRepository(db)
	return nil
}
