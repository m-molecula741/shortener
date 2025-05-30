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
}

func NewHTTPController(service URLService) *HTTPController {
	c := &HTTPController{
		service: service,
		router:  chi.NewRouter(),
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

	c.router.Post("/", c.handleShorten)
	c.router.Get("/{shortID}", c.handleRedirect)
	c.router.Post("/api/shorten", c.handleShortenJSON)
	c.router.Post("/api/shorten/batch", c.handleShortenBatch)
	c.router.Get("/ping", c.handlePing)
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

	shortURL, err := c.service.Shorten(string(body))
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
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		http.Error(w, "Error expand short url", http.StatusBadRequest)
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

	shortURL, err := c.service.Shorten(req.URL)
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

	responses, err := c.service.ShortenBatch(requests)
	if err != nil {
		http.Error(w, "Batch shorten failed", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(responses)
}
