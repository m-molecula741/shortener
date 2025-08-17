package main

import (
	"context"
	"log"
	"net/http"
	_ "net/http/pprof"
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

	// Запускаем pprof только если включен debug режим
	if cfg.EnablePprof {
		go func() {
			log.Println("pprof server started at http://localhost:6060/debug/pprof/")
			if err := http.ListenAndServe("localhost:6060", nil); err != nil {
				log.Println("pprof server error:", err)
			}
		}()
	}

	// Инициализируем middleware аутентификации
	auth, err := middleware.NewAuthMiddleware("secret-key-for-auth")
	if err != nil {
		logger.Info().
			Err(err).
			Msg("Failed to initialize auth middleware")
		os.Exit(1)
	}

	var store usecase.URLStorage
	var dbPinger usecase.DatabasePinger

	if cfg.DatabaseDSN != "" {
		// Используем PostgreSQL как основное хранилище
		pgStorage, err := storage.NewPostgresStorage(cfg.DatabaseDSN, nil)
		if err != nil {
			logger.Info().
				Err(err).
				Msg("Failed to initialize PostgreSQL storage")
			os.Exit(1)
		}

		store = pgStorage
		dbPinger = pgStorage // PostgreSQL поддерживает ping

		defer func() {
			if err := pgStorage.Close(); err != nil {
				logger.Info().
					Err(err).
					Msg("Failed to close PostgreSQL connection")
			}
		}()

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

	urlService := usecase.NewURLService(store, cfg.BaseURL, dbPinger)
	var service controller.URLService = urlService
	httpController := controller.NewHTTPController(service, auth)

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
	logger.Info().Msg("Server stopped")

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Info().
			Err(err).
			Msg("Failed to gracefully shutdown the server")
	}

	// Закрываем сервис удаления URL
	urlService.Close()

	if fileStorage, ok := store.(*storage.InMemoryStorage); ok {
		if err := fileStorage.Backup(); err != nil {
			logger.Info().
				Err(err).
				Msg("Failed to backup storage")
		}
	}

	logger.Info().Msg("Server stopped")
}
