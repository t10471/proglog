package main

import (
	"context"
	"log"
	"os/signal"

	"github.com/sethvargo/go-envconfig"
	"go.uber.org/zap"
	"golang.org/x/sys/unix"

	"github.com/travisjeffery/proglog/internal/config"
	"github.com/travisjeffery/proglog/internal/di"
)

func main() {
	if err := run(); err != nil {
		log.Fatal(err)
	}
}

func run() error {
	ctx := context.Background()
	var cfg *config.Env
	if err := envconfig.Process(ctx, &cfg); err != nil {
		return err
	}
	if err := setupLogger(cfg); err != nil {
		return err
	}

	service, err := di.InitializeService(cfg)
	if err != nil {
		return err
	}
	notifyCtx, stop := signal.NotifyContext(ctx, unix.SIGINT, unix.SIGTERM)
	defer stop()

	service.Serve()
	<-notifyCtx.Done()
	return service.Shutdown()
}

func setupLogger(cfg *config.Env) error {
	var c zap.Config
	if cfg.Environment == config.Local {
		c = zap.NewDevelopmentConfig()
	} else {
		c = zap.NewProductionConfig()
	}
	c.Level = zap.NewAtomicLevelAt(cfg.LogLevel)
	l, err := c.Build()
	if err == nil {
		zap.ReplaceGlobals(l)
	}
	return err
}
