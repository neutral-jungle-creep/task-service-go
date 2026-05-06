package postgres_test

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/pkg/postgres"
)

func TestNew_EmptyDSN(t *testing.T) {
	t.Parallel()

	_, err := postgres.New(context.Background(), postgres.Config{})
	require.Error(t, err)
	assert.ErrorIs(t, err, postgres.ErrEmptyDSN)
}

func TestNew_PingFails(t *testing.T) {
	t.Parallel()

	_, err := postgres.New(context.Background(), postgres.Config{
		DSN:         "host=127.0.0.1 port=1 user=none dbname=none sslmode=disable connect_timeout=1",
		PingTimeout: 500 * time.Millisecond,
	})
	require.Error(t, err, "ping must fail against an unreachable host")
	assert.NotErrorIs(t, err, postgres.ErrEmptyDSN)
}
