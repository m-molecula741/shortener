package main

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/m-molecula741/shortener/internal/app/controller"
	"github.com/m-molecula741/shortener/internal/app/storage"
	"github.com/m-molecula741/shortener/internal/app/usecase"
)

func main() {
	storage := storage.NewInMemoryStorage()
	service := usecase.NewURLService(storage)
	handler := controller.NewHTTPController(service)

	router := http.NewServeMux()
	router.HandleFunc("/", handler.HandleRequest)

	server := &http.Server{
		Addr:    ":8080",
		Handler: router,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Println("Server starting on :8080")
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Server error: %v", err)
		}
	}()

	<-done
	log.Println("Server is shutting down...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Graceful shutdown failed: %v", err)
	}

	log.Println("Server stopped")
}
