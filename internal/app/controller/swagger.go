package controller

import (
	"github.com/go-chi/chi/v5"
	_ "github.com/m-molecula741/shortener/docs" // импорт сгенерированной документации
	httpSwagger "github.com/swaggo/http-swagger"
)

// SetupSwagger настраивает маршруты для Swagger UI и документации
func SetupSwagger(router *chi.Mux) {
	router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))
}
