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
	"github.com/m-molecula741/shortener/internal/app/storage"
	"github.com/m-molecula741/shortener/internal/app/usecase"
)

func main() {
	logger.Init()

	cfg := config.NewConfig()

	// Инициализируем хранилище с поддержкой файла
	store, err := storage.NewInMemoryStorage(cfg.StorageFilePath)
	if err != nil {
		logger.Info().
			Err(err).
			Msg("Failed to initialize storage")
		os.Exit(1)
	}

	service := usecase.NewURLService(store, cfg.BaseURL)
	controller := controller.NewHTTPController(service)

	server := &http.Server{
		Addr:    cfg.ServerAddress,
		Handler: logger.RequestLogger(controller),
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

	// Сохраняем данные перед выключением
	if err := store.Backup(); err != nil {
		logger.Info().
			Err(err).
			Msg("Failed to backup storage")
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Info().
			Err(err).
			Msg("Server shutdown error")
	}

	logger.Info().Msg("Server stopped")
}
