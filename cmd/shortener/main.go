package main

import (
	"context"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m-molecula741/shortener/internal/app/config"
	"github.com/m-molecula741/shortener/internal/app/controller"
	"github.com/m-molecula741/shortener/internal/app/logger"
	"github.com/m-molecula741/shortener/internal/app/middleware"
	"github.com/m-molecula741/shortener/internal/app/storage"
	"github.com/m-molecula741/shortener/internal/app/usecase"
)

func main() {
	logger.Init()

	cfg := config.NewConfig()

	var store usecase.URLStorage
	var dbPinger usecase.DatabasePinger

	if cfg.DatabaseDSN != "" {
		// Используем PostgreSQL как основное хранилище
		pgStorage, err := storage.NewPostgresStorage(cfg.DatabaseDSN)
		if err != nil {
			logger.Info().
				Err(err).
				Msg("Failed to initialize PostgreSQL storage")
			os.Exit(1)
		}

		store = pgStorage
		dbPinger = pgStorage // PostgreSQL поддерживает ping

		defer pgStorage.Close()

		logger.Info().Msg("Using PostgreSQL storage")
	} else {
		fileStorage, err := storage.NewInMemoryStorage(cfg.StorageFilePath)
		if err != nil {
			logger.Info().
				Err(err).
				Msg("Failed to initialize file storage")
			os.Exit(1)
		}

		store = fileStorage
		dbPinger = nil // файловое хранилище не поддерживает ping

		logger.Info().Msg("Using file storage")
	}

	service := usecase.NewURLService(store, cfg.BaseURL, dbPinger)
	httpController := controller.NewHTTPController(service)

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: middleware.RequestLogger(httpController),
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		logger.Info().
			Str("address", cfg.ServerAddress).
			Msg("Starting server")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			logger.Info().
				Err(err).
				Msg("Server error")
			os.Exit(1)
		}
	}()

	<-done
	logger.Info().Msg("Server is shutting down...")

	// Graceful shutdown
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Info().
			Err(err).
			Msg("Server shutdown error")
	}

	if fileStorage, ok := store.(*storage.InMemoryStorage); ok {
		if err := fileStorage.Backup(); err != nil {
			logger.Info().
				Err(err).
				Msg("Failed to backup storage")
		}
	}

	logger.Info().Msg("Server stopped")
}
