//go:build integration

package integration_test

import (
	"context"
	"database/sql"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/go-playground/validator/v10"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/stretchr/testify/require"

	"task-service/internal/adapters/repositories"
	"task-service/internal/server"
	"task-service/internal/services"
	"task-service/pkg/http/protocol"
	"task-service/pkg/logging"
)

const defaultTestDSN = "host=127.0.0.1 port=5433 user=user password=user_password dbname=task-service-test sslmode=disable"

type testEnv struct {
	db     *sql.DB
	server *httptest.Server
	cancel context.CancelFunc
}

func (e *testEnv) close(t *testing.T) {
	t.Helper()
	e.server.Close()
	if e.cancel != nil {
		e.cancel()
	}
	require.NoError(t, e.db.Close())
}

func setupTestEnv(t *testing.T) *testEnv {
	t.Helper()

	dsn := os.Getenv("TEST_DB_POSTGRES_DSN")
	if dsn == "" {
		dsn = defaultTestDSN
	}

	db, err := sql.Open("pgx", dsn)
	require.NoError(t, err)

	pingCtx, pingCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer pingCancel()
	if err := db.PingContext(pingCtx); err != nil {
		_ = db.Close()
		t.Skipf("postgres is not available at %s (skipping integration tests): %v", dsn, err)
	}

	_, err = db.ExecContext(pingCtx, "TRUNCATE TABLE tasks RESTART IDENTITY")
	require.NoError(t, err)

	repo := repositories.NewTaskRepository(db)

	core, err := logging.NewLogger("error", "task-service-it", "test")
	require.NoError(t, err)

	ctx, cancel := context.WithCancel(context.Background())
	async := logging.NewAsyncLogger(ctx, core)
	go func() { _ = async.Process() }()

	cache, err := services.NewTaskCache(ctx, 8, time.Hour, repo)
	require.NoError(t, err)

	svc := services.NewTaskService(async, repo, cache)
	rh := protocol.NewResponseHandler(
		async,
		protocol.WithValidation(validator.New(validator.WithRequiredStructEnabled())),
	)
	api := server.NewAPI(rh, svc)

	httpSrv := httptest.NewServer(api.InitRoutes("/api/v1/task-service"))

	return &testEnv{
		db:     db,
		server: httpSrv,
		cancel: func() { async.Stop(); cancel() },
	}
}
