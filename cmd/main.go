package main

import (
	"context"
	"os/signal"
	"syscall"

	"task-service/internal/config"
	"task-service/internal/root"
	"task-service/pkg/logging"
)

func main() {
	ctx, _ := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)

	cfg, err := config.NewConfigFromEnv()
	if err != nil {
		panic(err)
	}

	logger, err := logging.NewLogger(cfg.LogLevel, cfg.ServiceName, cfg.ReleaseID)
	if err != nil {
		panic(err)
	}

	application, err := root.New(ctx, cfg, logger)
	if err != nil {
		logger.Fatal("application could not been initialized", err)
	}

	err = application.Run()
	if err != nil {
		logger.Fatal("application terminated abnormally", err)
	}
}
