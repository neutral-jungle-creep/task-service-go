package config_test

import (
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"task-service/internal/config"
)

const dotEnvFilenameExample = "../../.env.example"

func TestNewConfigFromFile_RejectsMissingEnvOrInvalidPath(t *testing.T) {
	const key = "DB_POSTGRES_DSN"
	t.Setenv(key, os.Getenv(key))
	require.NoError(t, os.Unsetenv(key))

	_, err := config.NewConfigFromFile("/non-existent.env")
	require.Error(t, err)
}

func TestParseAndValidate(t *testing.T) {
	cfg, err := config.NewConfigFromFile(dotEnvFilenameExample)
	require.NoError(t, err)

	assert.Equal(t, "task-service-go", cfg.ServiceName)
	assert.Equal(t, "local", cfg.ReleaseID)
	assert.Equal(t, "debug", cfg.LogLevel)
	assert.Equal(t, "/api/v1/task-service", cfg.RouteGroup)

	assert.Equal(t, "8888", cfg.HTTPServer.ListenPort)
	assert.Equal(t, 60*time.Second, cfg.HTTPServer.KeepAliveTime)
	assert.Equal(t, 10*time.Second, cfg.HTTPServer.KeepAliveTimeout)
	assert.Equal(t, 10*time.Second, cfg.HTTPServer.ReadHeaderTimeout)
	assert.Equal(t, 10*time.Second, cfg.HTTPServer.ReadTimeout)
	assert.Equal(t, 10*time.Second, cfg.HTTPServer.WriteTimeout)

	assert.Equal(t,
		"host=localhost port=5432 user=user password=user_password dbname=task-service sslmode=disable",
		cfg.Database.DSN,
	)
	assert.Equal(t, 10, cfg.Database.MaxOpenConns)
	assert.Equal(t, 5, cfg.Database.MaxIdleConns)
	assert.Equal(t, 30*time.Minute, cfg.Database.ConnMaxLifetime)
	assert.Equal(t, 5*time.Second, cfg.Database.QueryTimeout)

	assert.Equal(t, 100, cfg.Cache.MemoryCacheLimitMB)
	assert.Equal(t, 10*time.Second, cfg.Cache.MemoryMonitorCacheInterval)
}
