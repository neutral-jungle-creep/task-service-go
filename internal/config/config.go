package config

import (
	"bytes"
	"encoding/json"
	"flag"
	"io"
	"os"
	"time"

	"github.com/pkg/errors"
	"task-service/pkg/http/server"
	"task-service/pkg/logging"
)

type Config struct {
	ServiceName string         `json:"serviceName"`
	ReleaseID   string         `json:"releaseId"`
	Logger      logging.Config `json:"logger"`
	HTTPServer  server.Config  `json:"httpServer"`
	Cache       CacheConfig    `json:"cache"`
}

type CacheConfig struct {
	MemoryCacheLimitMB         int           `json:"memoryCacheLimitMB"`
	MemoryMonitorCacheInterval time.Duration `json:"memoryMonitorCacheInterval"`
}

func NewConfigFromEnv() (*Config, error) {
	var configPath string

	flag.StringVar(&configPath, "path", "config.json", "Path to config file")
	flag.StringVar(&configPath, "p", "config.json", "Path to config file")
	flag.Parse()

	var config Config

	err := processEnv(&config, configPath)
	if err != nil {
		return nil, errors.Wrap(err, "unable process env")
	}

	return &config, nil
}

func processEnv(config *Config, configPath string) error {
	file, err := os.Open(configPath)
	if err != nil {
		return err
	}
	defer file.Close()

	var buf bytes.Buffer
	_, err = io.Copy(&buf, file)
	if err != nil {
		return err
	}

	err = json.Unmarshal(buf.Bytes(), config)
	if err != nil {
		return err
	}
	return nil
}
