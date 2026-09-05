package main

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
	"time"

	apphttp "github.com/drobyshevv/url-shortening-service/internal/app"
	"github.com/drobyshevv/url-shortening-service/internal/config"
)

const (
	envLocal = "local"
	envDev   = "dev"
)

func main() {
	cfg, err := config.MustLoad()
	if err != nil {
		slog.Error("failed to load config", "error", err)
		panic(err)
	}
	log := setupLogger(cfg.Env)

	log.Info("starting application", "env", cfg.Env)

	app, err := apphttp.NewApp(log, cfg)
	if err != nil {
		log.Error("failed to init app", "error", err)
		panic(err)

	}

	errCh := make(chan error, 1)

	go func() {
		errCh <- app.Run()
	}()

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	select {
	case err := <-errCh:
		if err != nil {
			log.Error("server stopped unexpectedly", "error", err)
		}

	case <-ctx.Done():
		log.Info("shutdown signal received")
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		log.Info("starting graceful shutdown")

		if err := app.Stop(shutdownCtx); err != nil {
			log.Error("graceful shutdown failed", "error", err)
			app.Close()
		}
		log.Info("application stopped")
	}
}

func setupLogger(env string) *slog.Logger {
	var log *slog.Logger

	switch env {
	case envLocal:
		log = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	case envDev:
		log = slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
	}

	return log
}
