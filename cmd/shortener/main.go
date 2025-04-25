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
	store := storage.NewInMemoryStorage()
	service := usecase.NewURLService(store)
	controller := controller.NewHTTPController(service)

	server := &http.Server{
		Addr:    ":8080",
		Handler: controller,
	}

	done := make(chan os.Signal, 1)
	signal.Notify(done, os.Interrupt, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		log.Printf("Сервер запущен на http://localhost%s\n", server.Addr)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Fatalf("Ошибка запуска сервера: %v", err)
		}
	}()

	<-done
	log.Println("Сервер останавливается...")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		log.Printf("Ошибка при остановке сервера: %v", err)
	}

	log.Println("Сервер остановлен")
}
