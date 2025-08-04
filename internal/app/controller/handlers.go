// Package controller предоставляет HTTP обработчики для сервиса сокращения URL.
// @title Сервис сокращения URL
// @version 1.0
// @description Сервис для сокращения длинных URL и управления ими
// @host localhost:8080
// @BasePath /
package controller

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/m-molecula741/shortener/docs" // импорт сгенерированной документации
	appmiddleware "github.com/m-molecula741/shortener/internal/app/middleware"
	"github.com/m-molecula741/shortener/internal/app/usecase"
	httpSwagger "github.com/swaggo/http-swagger"
)

// HTTPController обрабатывает HTTP запросы к сервису сокращения URL.
type HTTPController struct {
	service URLService
	router  *chi.Mux
	auth    *appmiddleware.AuthMiddleware
}

// NewHTTPController создает новый экземпляр HTTPController.
func NewHTTPController(service URLService, auth *appmiddleware.AuthMiddleware) *HTTPController {
	c := &HTTPController{
		service: service,
		router:  chi.NewRouter(),
		auth:    auth,
	}
	c.setupRoutes()
	return c
}

// ShortenRequest представляет запрос на сокращение URL.
type ShortenRequest struct {
	URL string `json:"url" example:"https://practicum.yandex.ru"` // URL для сокращения
}

// ShortenResponse представляет ответ с сокращенным URL.
type ShortenResponse struct {
	Result string `json:"result" example:"http://localhost:8080/abcd1234"` // Сокращенный URL
}

// setupRoutes настраивает маршруты для обработки HTTP запросов.
func (c *HTTPController) setupRoutes() {
	c.router.Use(chimiddleware.Logger)
	c.router.Use(chimiddleware.Recoverer)
	c.router.Use(appmiddleware.GzipMiddleware)
	c.router.Use(c.auth.Middleware)

	// Swagger UI и документация
	c.router.Get("/swagger/*", httpSwagger.Handler(
		httpSwagger.URL("http://localhost:8080/swagger/doc.json"),
	))

	// Основные роуты
	c.router.Post("/", c.handleShorten)
	c.router.Get("/{shortID}", c.handleRedirect)
	c.router.Post("/api/shorten", c.handleShortenJSON)
	c.router.Post("/api/shorten/batch", c.handleShortenBatch)
	c.router.Get("/ping", c.handlePing)
	c.router.Get("/api/user/urls", c.handleGetUserURLs)
	c.router.Delete("/api/user/urls", c.handleDeleteUserURLs)
}

// ServeHTTP реализует интерфейс http.Handler.
func (c *HTTPController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.router.ServeHTTP(w, r)
}

// @Summary Сокращение URL (текстовый формат)
// @Description Принимает URL в текстовом формате и возвращает сокращенную версию
// @Tags URLs
// @Accept plain
// @Produce plain
// @Param url body string true "URL для сокращения"
// @Success 201 {string} string "Сокращенный URL"
// @Failure 400 {string} string "Неверный запрос"
// @Failure 409 {string} string "URL уже существует"
// @Router / [post]
func (c *HTTPController) handleShorten(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	if len(body) == 0 {
		http.Error(w, "Empty short url", http.StatusBadRequest)
		return
	}

	// Получаем userID из контекста
	userID, _ := appmiddleware.GetUserIDFromContext(r.Context())

	shortURL, err := c.service.ShortenWithUser(r.Context(), string(body), userID)
	if err != nil {
		if conflictErr, isConflict := usecase.IsURLConflict(err); isConflict {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte(conflictErr.ExistingShortURL))
			return
		}
		http.Error(w, "Shorten failed", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

// @Summary Получение оригинального URL
// @Description Перенаправляет на оригинальный URL по его короткому идентификатору
// @Tags URLs
// @Param shortID path string true "Короткий идентификатор URL"
// @Success 307 {string} string "Перенаправление"
// @Failure 404 {string} string "URL не найден"
// @Failure 410 {string} string "URL был удален"
// @Router /{shortID} [get]
func (c *HTTPController) handleRedirect(w http.ResponseWriter, r *http.Request) {
	shortID := chi.URLParam(r, "shortID")
	if shortID == "" {
		shortID = r.URL.Path[1:]
	}

	originalURL, err := c.service.Expand(shortID)
	if err != nil {
		if usecase.IsURLDeleted(err) {
			w.Header().Set("Content-Type", "text/plain; charset=utf-8")
			w.WriteHeader(http.StatusGone)
			w.Write([]byte("URL has been deleted"))
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("URL not found"))
		return
	}

	w.Header().Set("Location", originalURL)
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.WriteHeader(http.StatusTemporaryRedirect)
}

// @Summary Сокращение URL (JSON формат)
// @Description Принимает URL в формате JSON и возвращает сокращенную версию
// @Tags URLs
// @Accept json
// @Produce json
// @Param request body ShortenRequest true "URL для сокращения"
// @Success 201 {object} ShortenResponse "Сокращенный URL"
// @Failure 400 {string} string "Неверный запрос"
// @Failure 409 {object} ShortenResponse "URL уже существует"
// @Router /api/shorten [post]
func (c *HTTPController) handleShortenJSON(w http.ResponseWriter, r *http.Request) {
	var req ShortenRequest

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if req.URL == "" {
		http.Error(w, "URL is required", http.StatusBadRequest)
		return
	}

	// Получаем userID из контекста
	userID, _ := appmiddleware.GetUserIDFromContext(r.Context())

	shortURL, err := c.service.ShortenWithUser(r.Context(), req.URL, userID)
	if err != nil {
		if conflictErr, isConflict := usecase.IsURLConflict(err); isConflict {
			response := ShortenResponse{
				Result: conflictErr.ExistingShortURL,
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusConflict)
			json.NewEncoder(w).Encode(response)
			return
		}
		http.Error(w, "Shorten failed", http.StatusInternalServerError)
		return
	}

	response := ShortenResponse{
		Result: shortURL,
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(response)
}

// @Summary Проверка работоспособности
// @Description Проверяет подключение к базе данных
// @Tags System
// @Success 200 {string} string "БД доступна"
// @Failure 500 {string} string "Ошибка подключения к БД"
// @Router /ping [get]
func (c *HTTPController) handlePing(w http.ResponseWriter, r *http.Request) {
	if err := c.service.PingDB(); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

// @Summary Пакетное сокращение URL
// @Description Принимает массив URL и возвращает их сокращенные версии
// @Tags URLs
// @Accept json
// @Produce json
// @Param request body []usecase.BatchShortenRequest true "Массив URL для сокращения"
// @Success 201 {array} usecase.BatchShortenResponse "Массив сокращенных URL"
// @Failure 400 {string} string "Неверный запрос"
// @Router /api/shorten/batch [post]
func (c *HTTPController) handleShortenBatch(w http.ResponseWriter, r *http.Request) {
	var requests []usecase.BatchShortenRequest

	if err := json.NewDecoder(r.Body).Decode(&requests); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(requests) == 0 {
		http.Error(w, "Empty batch", http.StatusBadRequest)
		return
	}

	// Получаем userID из контекста
	userID, _ := appmiddleware.GetUserIDFromContext(r.Context())

	responses, err := c.service.ShortenBatchWithUser(r.Context(), requests, userID)
	if err != nil {
		http.Error(w, "Batch shorten failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(responses)
}

// @Summary Получение URL пользователя
// @Description Возвращает все сокращенные URL текущего пользователя
// @Tags Users
// @Produce json
// @Security Cookie
// @Success 200 {array} usecase.UserURL "Список URL пользователя"
// @Success 204 "URL не найдены"
// @Failure 401 {string} string "Не авторизован"
// @Failure 500 {string} string "Внутренняя ошибка сервера"
// @Router /api/user/urls [get]
func (c *HTTPController) handleGetUserURLs(w http.ResponseWriter, r *http.Request) {
	// Получаем ID пользователя из контекста (middleware уже добавил его)
	userID, ok := appmiddleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Получаем URL пользователя
	urls, err := c.service.GetUserURLs(r.Context(), userID)
	if err != nil {
		http.Error(w, "Failed to get user URLs", http.StatusInternalServerError)
		return
	}

	// Если у пользователя нет URL, возвращаем 204
	if len(urls) == 0 {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Возвращаем URL пользователя
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(urls)
}

// @Summary Удаление URL пользователя
// @Description Удаляет указанные URL пользователя
// @Tags Users
// @Accept json
// @Security Cookie
// @Param shortIDs body []string true "Массив коротких идентификаторов для удаления"
// @Success 202 "Запрос на удаление принят"
// @Failure 401 {string} string "Не авторизован"
// @Failure 400 {string} string "Неверный запрос"
// @Router /api/user/urls [delete]
func (c *HTTPController) handleDeleteUserURLs(w http.ResponseWriter, r *http.Request) {
	userID, ok := appmiddleware.GetUserIDFromContext(r.Context())
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	var shortIDs []string
	if err := json.NewDecoder(r.Body).Decode(&shortIDs); err != nil {
		http.Error(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	if len(shortIDs) == 0 {
		http.Error(w, "Empty short IDs list", http.StatusBadRequest)
		return
	}

	if err := c.service.DeleteUserURLs(userID, shortIDs); err != nil {
		http.Error(w, "Failed to queue deletion request", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusAccepted)
}
