// Package postgres opens a *sql.DB backed by the pgx/v5 stdlib driver and
// applies pool sizing/timeouts. Importing this package registers the "pgx"
// driver via its blank import.
package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
)

const defaultPingTimeout = 5 * time.Second

// ErrEmptyDSN is returned when New is called without a DSN.
var ErrEmptyDSN = errors.New("postgres dsn is empty")

// Config carries the connection settings that callers commonly tune from env.
type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	PingTimeout     time.Duration
}

// New opens a *sql.DB, applies pool sizing and verifies connectivity with a
// PingContext bounded by Config.PingTimeout (defaultPingTimeout if zero).
// On ping failure the pool is closed before the error is returned, so the
// caller never receives a partially-initialised DB.
func New(ctx context.Context, cfg Config) (*sql.DB, error) {
	if cfg.DSN == "" {
		return nil, ErrEmptyDSN
	}

	db, err := sql.Open("pgx", cfg.DSN)
	if err != nil {
		return nil, fmt.Errorf("open postgres: %w", err)
	}

	if cfg.MaxOpenConns > 0 {
		db.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		db.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		db.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}

	pingTimeout := cfg.PingTimeout
	if pingTimeout <= 0 {
		pingTimeout = defaultPingTimeout
	}

	pingCtx, cancel := context.WithTimeout(ctx, pingTimeout)
	defer cancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return db, nil
}
