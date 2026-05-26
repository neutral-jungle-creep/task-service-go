package config

import (
	"fmt"
	"time"

	"github.com/joho/godotenv"
	"github.com/kelseyhightower/envconfig"
)

const dotEnvFilename = ".env"

type Config struct {
	ServiceName string `envconfig:"SERVICE_NAME" default:"task-service-go"`
	ReleaseID   string `envconfig:"RELEASE_ID"`
	LogLevel    string `envconfig:"LOG_LEVEL" default:"info"`
	RouteGroup  string `envconfig:"ROUTE_GROUP" default:"/api/v1/task-service"`

	HTTPServer HTTPServerConfig `envconfig:"HTTP_SERVER"`
	Database   DatabaseConfig   `envconfig:"DB"`
	Cache      CacheConfig      `envconfig:"CACHE"`
}

type HTTPServerConfig struct {
	ListenPort        string        `envconfig:"LISTEN_PORT" default:"8888"`
	KeepAliveTime     time.Duration `envconfig:"KEEP_ALIVE_TIME" default:"60s"`
	KeepAliveTimeout  time.Duration `envconfig:"KEEP_ALIVE_TIMEOUT" default:"10s"`
	ReadHeaderTimeout time.Duration `envconfig:"READ_HEADER_TIMEOUT" default:"10s"`
	ReadTimeout       time.Duration `envconfig:"READ_TIMEOUT" default:"10s"`
	WriteTimeout      time.Duration `envconfig:"WRITE_TIMEOUT" default:"10s"`

	MaxRequestBodyBytes int64 `envconfig:"MAX_REQUEST_BODY_BYTES" default:"1048576"`

	IPRateLimit      float64       `envconfig:"IP_RATE_LIMIT" default:"50"`
	IPRateBurst      int           `envconfig:"IP_RATE_BURST" default:"100"`
	IPRateLimiterTTL time.Duration `envconfig:"IP_RATE_LIMITER_TTL" default:"10m"`
}

type DatabaseConfig struct {
	DSN             string        `envconfig:"POSTGRES_DSN" required:"true"`
	MaxOpenConns    int           `envconfig:"POSTGRES_MAX_OPEN_CONNS" default:"10"`
	MaxIdleConns    int           `envconfig:"POSTGRES_MAX_IDLE_CONNS" default:"5"`
	ConnMaxLifetime time.Duration `envconfig:"POSTGRES_MAX_LIFETIME" default:"30m"`
	QueryTimeout    time.Duration `envconfig:"POSTGRES_QUERY_TIMEOUT" default:"5s"`
}

type CacheConfig struct {
	MemoryCacheLimitMB         int           `envconfig:"MEMORY_LIMIT_MB" default:"1024"`
	MemoryMonitorCacheInterval time.Duration `envconfig:"MEMORY_MONITOR_INTERVAL" default:"5s"`
}

func NewConfigFromFile(fileName string) (*Config, error) {
	_ = godotenv.Load(fileName) // local-only convenience; ignore the error so the container start path still works

	cfg := &Config{}
	if err := envconfig.Process("", cfg); err != nil {
		return nil, fmt.Errorf("process env: %w", err)
	}
	return cfg, nil
}

func DotEnvFilename() string {
	return dotEnvFilename
}
