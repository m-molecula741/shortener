package controller

import (
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-chi/chi/v5"
	chimiddleware "github.com/go-chi/chi/v5/middleware"
	appmiddleware "github.com/m-molecula741/shortener/internal/app/middleware"
	"github.com/m-molecula741/shortener/internal/app/usecase"
)

type HTTPController struct {
	service URLService
	router  *chi.Mux
	auth    *appmiddleware.AuthMiddleware
}

func NewHTTPController(service URLService, auth *appmiddleware.AuthMiddleware) *HTTPController {
	c := &HTTPController{
		service: service,
		router:  chi.NewRouter(),
		auth:    auth,
	}
	c.setupRoutes()
	return c
}

type ShortenRequest struct {
	URL string `json:"url"`
}

type ShortenResponse struct {
	Result string `json:"result"`
}

func (c *HTTPController) setupRoutes() {
	c.router.Use(chimiddleware.Logger)
	c.router.Use(chimiddleware.Recoverer)
	c.router.Use(appmiddleware.GzipMiddleware)
	c.router.Use(c.auth.Middleware)

	c.router.Post("/", c.handleShorten)
	c.router.Get("/{shortID}", c.handleRedirect)
	c.router.Post("/api/shorten", c.handleShortenJSON)
	c.router.Post("/api/shorten/batch", c.handleShortenBatch)
	c.router.Get("/ping", c.handlePing)
	c.router.Get("/api/user/urls", c.handleGetUserURLs)
	c.router.Delete("/api/user/urls", c.handleDeleteUserURLs)
}

func (c *HTTPController) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	c.router.ServeHTTP(w, r)
}

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

func (c *HTTPController) handlePing(w http.ResponseWriter, r *http.Request) {
	if err := c.service.PingDB(); err != nil {
		http.Error(w, "Database connection failed", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusOK)
}

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

type respPair struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}

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

	// Преобразуем в нужный формат
	response := make([]respPair, len(urls))
	for i, url := range urls {
		response[i] = respPair{
			ShortURL:    url.ShortURL,
			OriginalURL: url.OriginalURL,
		}
	}

	// Возвращаем URL пользователя
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(response)
}

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
