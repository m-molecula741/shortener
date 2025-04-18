package controller

import (
	"io"
	"net/http"
)

type HTTPController struct {
	service URLService
}

func NewHTTPController(service URLService) *HTTPController {
	return &HTTPController{service: service}
}

func (c *HTTPController) HandleRequest(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodPost:
		c.handleShorten(w, r)
	case http.MethodGet:
		c.handleRedirect(w, r)
	default:
		http.Error(w, "Method not allowed", http.StatusBadRequest)
	}
}

func (c *HTTPController) handleShorten(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}

	shortURL, err := c.service.Shorten(string(body))
	if err != nil {
		http.Error(w, "Internal error", http.StatusBadRequest)
		return
	}

	w.Header().Set("Content-Type", "text/plain")
	w.WriteHeader(http.StatusCreated)
	w.Write([]byte(shortURL))
}

func (c *HTTPController) handleRedirect(w http.ResponseWriter, r *http.Request) {
	shortID := r.URL.Path[1:]
	originalURL, err := c.service.Expand(shortID)
	if err != nil {
		http.Error(w, "Not found", http.StatusBadRequest)
		return
	}

	w.Header().Set("Location", originalURL)
	w.WriteHeader(http.StatusTemporaryRedirect)
}
