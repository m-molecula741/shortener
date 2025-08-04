package controller_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"

	"github.com/m-molecula741/shortener/internal/app/controller"
	"github.com/m-molecula741/shortener/internal/app/middleware"
	"github.com/m-molecula741/shortener/internal/app/usecase"
)

// Пример сокращения URL через JSON API
func Example_shortenURL() {
	// Создаем мок сервиса
	mockService := &MockURLService{
		ShortenWithUserFunc: func(ctx context.Context, url, userID string) (string, error) {
			return "http://localhost:8080/abc123", nil
		},
	}

	// Создаем контроллер
	auth, _ := middleware.NewAuthMiddleware("test-key")
	ctrl := controller.NewHTTPController(mockService, auth)

	// Создаем тестовый сервер
	ts := httptest.NewServer(ctrl)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/api/shorten",
		"application/json",
		bytes.NewBufferString(`{"url":"http://example.com"}`))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\nResponse: %s\n", resp.StatusCode, body)
	// Output:
	// Status: 201
	// Response: {"result":"http://localhost:8080/abc123"}
}

// Пример получения URL пользователя
func Example_getUserURLs() {
	// Создаем мок сервиса
	mockService := &MockURLService{
		GetUserURLsFunc: func(ctx context.Context, userID string) ([]usecase.UserURL, error) {
			return []usecase.UserURL{
				{
					ShortURL:    "http://localhost:8080/abc123",
					OriginalURL: "http://example.com",
				},
			}, nil
		},
	}

	// Создаем контроллер
	auth, _ := middleware.NewAuthMiddleware("test-key")
	ctrl := controller.NewHTTPController(mockService, auth)

	// Создаем тестовый сервер
	ts := httptest.NewServer(ctrl)
	defer ts.Close()

	// Создаем HTTP клиент с куками
	client := &http.Client{}
	req, _ := http.NewRequest("GET", ts.URL+"/api/user/urls", nil)
	req.AddCookie(&http.Cookie{
		Name:  "user_id",
		Value: "test-user-id",
	})

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\nResponse: %s\n", resp.StatusCode, body)
	// Output:
	// Status: 200
	// Response: [{"short_url":"http://localhost:8080/abc123","original_url":"http://example.com"}]
}

// Пример сокращения URL в текстовом формате
func Example_shortenURLPlainText() {
	// Создаем мок сервиса
	mockService := &MockURLService{
		ShortenWithUserFunc: func(ctx context.Context, url, userID string) (string, error) {
			return "http://localhost:8080/abc123", nil
		},
	}

	// Создаем контроллер
	auth, _ := middleware.NewAuthMiddleware("test-key")
	ctrl := controller.NewHTTPController(mockService, auth)

	// Создаем тестовый сервер
	ts := httptest.NewServer(ctrl)
	defer ts.Close()

	resp, err := http.Post(ts.URL+"/",
		"text/plain",
		bytes.NewBufferString("http://example.com"))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\nResponse: %s\n", resp.StatusCode, body)
	// Output:
	// Status: 201
	// Response: http://localhost:8080/abc123
}

// Пример пакетного сокращения URL
func Example_batchShorten() {
	// Создаем мок сервиса
	mockService := &MockURLService{
		ShortenBatchWithUserFunc: func(ctx context.Context, requests []usecase.BatchShortenRequest, userID string) ([]usecase.BatchShortenResponse, error) {
			responses := make([]usecase.BatchShortenResponse, len(requests))
			for i, req := range requests {
				responses[i] = usecase.BatchShortenResponse{
					CorrelationID: req.CorrelationID,
					ShortURL:      fmt.Sprintf("http://localhost:8080/batch%d", i+1),
				}
			}
			return responses, nil
		},
	}

	// Создаем контроллер
	auth, _ := middleware.NewAuthMiddleware("test-key")
	ctrl := controller.NewHTTPController(mockService, auth)

	// Создаем тестовый сервер
	ts := httptest.NewServer(ctrl)
	defer ts.Close()

	// Подготавливаем batch запрос
	batch := []usecase.BatchShortenRequest{
		{CorrelationID: "1", OriginalURL: "http://example1.com"},
		{CorrelationID: "2", OriginalURL: "http://example2.com"},
	}
	reqBody, _ := json.Marshal(batch)

	resp, err := http.Post(ts.URL+"/api/shorten/batch",
		"application/json",
		bytes.NewBuffer(reqBody))
	if err != nil {
		return
	}
	defer resp.Body.Close()

	// Читаем ответ
	body, _ := io.ReadAll(resp.Body)
	fmt.Printf("Status: %d\nResponse: %s\n", resp.StatusCode, body)
	// Output:
	// Status: 201
	// Response: [{"correlation_id":"1","short_url":"http://localhost:8080/batch1"},{"correlation_id":"2","short_url":"http://localhost:8080/batch2"}]
}

// Пример получения оригинального URL
func Example_getOriginalURL() {
	// Создаем мок сервиса
	mockService := &MockURLService{
		ExpandFunc: func(shortID string) (string, error) {
			return "http://example.com", nil
		},
	}

	// Создаем контроллер
	auth, _ := middleware.NewAuthMiddleware("test-key")
	ctrl := controller.NewHTTPController(mockService, auth)

	// Создаем тестовый сервер
	ts := httptest.NewServer(ctrl)
	defer ts.Close()

	// Отправляем запрос
	client := &http.Client{
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Get(ts.URL + "/abc123")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\nLocation: %s\n", resp.StatusCode, resp.Header.Get("Location"))
	// Output:
	// Status: 307
	// Location: http://example.com
}

// Пример удаления URL пользователя
func Example_deleteUserURLs() {
	// Создаем мок сервиса
	mockService := &MockURLService{
		DeleteUserURLsFunc: func(userID string, shortIDs []string) error {
			return nil
		},
	}

	// Создаем контроллер
	auth, _ := middleware.NewAuthMiddleware("test-key")
	ctrl := controller.NewHTTPController(mockService, auth)

	// Создаем тестовый сервер
	ts := httptest.NewServer(ctrl)
	defer ts.Close()

	// Подготавливаем запрос на удаление
	shortIDs := []string{"abc123", "def456"}
	reqBody, _ := json.Marshal(shortIDs)

	// Создаем HTTP клиент с куками
	client := &http.Client{}
	req, _ := http.NewRequest("DELETE", ts.URL+"/api/user/urls", bytes.NewBuffer(reqBody))
	req.Header.Set("Content-Type", "application/json")
	req.AddCookie(&http.Cookie{
		Name:  "user_id",
		Value: "test-user-id",
	})

	resp, err := client.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	// Output:
	// Status: 202
}

// Пример проверки работоспособности сервиса
func Example_pingService() {
	// Создаем мок сервиса
	mockService := &MockURLService{
		PingDBFunc: func() error {
			return nil
		},
	}

	// Создаем контроллер
	auth, _ := middleware.NewAuthMiddleware("test-key")
	ctrl := controller.NewHTTPController(mockService, auth)

	// Создаем тестовый сервер
	ts := httptest.NewServer(ctrl)
	defer ts.Close()

	resp, err := http.Get(ts.URL + "/ping")
	if err != nil {
		return
	}
	defer resp.Body.Close()

	fmt.Printf("Status: %d\n", resp.StatusCode)
	// Output:
	// Status: 200
}

// MockURLService реализует интерфейс URLService для тестов
type MockURLService struct {
	ShortenWithUserFunc      func(ctx context.Context, url, userID string) (string, error)
	GetUserURLsFunc          func(ctx context.Context, userID string) ([]usecase.UserURL, error)
	ExpandFunc               func(shortID string) (string, error)
	PingDBFunc               func() error
	DeleteUserURLsFunc       func(userID string, shortIDs []string) error
	ShortenBatchWithUserFunc func(ctx context.Context, requests []usecase.BatchShortenRequest, userID string) ([]usecase.BatchShortenResponse, error)
}

func (m *MockURLService) Shorten(url string) (string, error) {
	return "http://localhost:8080/abc123", nil
}

func (m *MockURLService) ShortenWithUser(ctx context.Context, url, userID string) (string, error) {
	if m.ShortenWithUserFunc != nil {
		return m.ShortenWithUserFunc(ctx, url, userID)
	}
	return "http://localhost:8080/abc123", nil
}

func (m *MockURLService) Expand(shortID string) (string, error) {
	if m.ExpandFunc != nil {
		return m.ExpandFunc(shortID)
	}
	return "http://example.com", nil
}

func (m *MockURLService) PingDB() error {
	if m.PingDBFunc != nil {
		return m.PingDBFunc()
	}
	return nil
}

func (m *MockURLService) ShortenBatch(ctx context.Context, requests []usecase.BatchShortenRequest) ([]usecase.BatchShortenResponse, error) {
	return nil, nil
}

func (m *MockURLService) ShortenBatchWithUser(ctx context.Context, requests []usecase.BatchShortenRequest, userID string) ([]usecase.BatchShortenResponse, error) {
	if m.ShortenBatchWithUserFunc != nil {
		return m.ShortenBatchWithUserFunc(ctx, requests, userID)
	}
	return nil, nil
}

func (m *MockURLService) GetUserURLs(ctx context.Context, userID string) ([]usecase.UserURL, error) {
	if m.GetUserURLsFunc != nil {
		return m.GetUserURLsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockURLService) DeleteUserURLs(userID string, shortIDs []string) error {
	if m.DeleteUserURLsFunc != nil {
		return m.DeleteUserURLsFunc(userID, shortIDs)
	}
	return nil
}
