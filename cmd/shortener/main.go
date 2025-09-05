package main

import (
	"context"
	"fmt"
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

var (
	buildVersion string
	buildDate    string
	buildCommit  string
)

// printBuildInfo выводит информацию о сборке в stdout
func printBuildInfo() {
	version := buildVersion
	if version == "" {
		version = "N/A"
	}

	date := buildDate
	if date == "" {
		date = "N/A"
	}

	commit := buildCommit
	if commit == "" {
		commit = "N/A"
	}

	fmt.Printf("Build version: %s\n", version)
	fmt.Printf("Build date: %s\n", date)
	fmt.Printf("Build commit: %s\n", commit)
}

// main является точкой входа приложения.
// Вся основная логика вынесена в функцию run для корректного завершения с кодом выхода.
func main() {
	if err := run(); err != nil {
		// Используем log.Fatal вместо os.Exit для корректного завершения
		log.Fatal("Application failed:", err)
	}
}

// run содержит основную логику приложения.
// Возвращает ошибку, если приложение не может быть запущено или корректно завершено.
func run() error {
	// Выводим информацию о сборке
	printBuildInfo()

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
		return fmt.Errorf("failed to initialize auth middleware: %w", err)
	}

	var store usecase.URLStorage
	var dbPinger usecase.DatabasePinger

	if cfg.DatabaseDSN != "" {
		// Используем PostgreSQL как основное хранилище
		pgStorage, err := storage.NewPostgresStorage(cfg.DatabaseDSN, nil)
		if err != nil {
			return fmt.Errorf("failed to initialize PostgreSQL storage: %w", err)
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
			return fmt.Errorf("failed to initialize file storage: %w", err)
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

	// Канал для передачи ошибок сервера
	serverErrChan := make(chan error, 1)

	go func() {
		if cfg.EnableHTTPS {
			logger.Info().
				Str("address", cfg.ServerAddress).
				Str("cert", cfg.CertFile).
				Str("key", cfg.KeyFile).
				Msg("Starting HTTPS server")
			if err := server.ListenAndServeTLS(cfg.CertFile, cfg.KeyFile); err != nil && err != http.ErrServerClosed {
				serverErrChan <- fmt.Errorf("HTTPS server error: %w", err)
			}
		} else {
			logger.Info().
				Str("address", cfg.ServerAddress).
				Msg("Starting HTTP server")
			if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				serverErrChan <- fmt.Errorf("HTTP server error: %w", err)
			}
		}
	}()

	// Ждем либо сигнал завершения, либо ошибку сервера
	select {
	case <-done:
		logger.Info().Msg("Received shutdown signal")
	case err := <-serverErrChan:
		return err
	}

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
	return nil
}
