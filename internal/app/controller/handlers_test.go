package controller

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/m-molecula741/shortener/internal/app/middleware"
	"github.com/m-molecula741/shortener/internal/app/usecase"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockURLService мок для URLService
type MockURLService struct {
	ShortenFunc      func(url string) (string, error)
	ExpandFunc       func(shortID string) (string, error)
	PingDBFunc       func() error
	ShortenBatchFunc func(ctx context.Context, requests []usecase.BatchShortenRequest) ([]usecase.BatchShortenResponse, error)
	GetUserURLsFunc  func(ctx context.Context, userID string) ([]usecase.UserURL, error)
}

func (m *MockURLService) Shorten(url string) (string, error) {
	if m.ShortenFunc != nil {
		return m.ShortenFunc(url)
	}
	return "", nil
}

func (m *MockURLService) Expand(shortID string) (string, error) {
	if m.ExpandFunc != nil {
		return m.ExpandFunc(shortID)
	}
	return "", nil
}

func (m *MockURLService) PingDB() error {
	if m.PingDBFunc != nil {
		return m.PingDBFunc()
	}
	return nil
}

func (m *MockURLService) ShortenBatch(ctx context.Context, requests []usecase.BatchShortenRequest) ([]usecase.BatchShortenResponse, error) {
	if m.ShortenBatchFunc != nil {
		return m.ShortenBatchFunc(ctx, requests)
	}

	responses := make([]usecase.BatchShortenResponse, len(requests))
	for i, req := range requests {
		responses[i] = usecase.BatchShortenResponse{
			CorrelationID: req.CorrelationID,
			ShortURL:      "http://localhost:8080/batch" + string(rune(i+'1')),
		}
	}
	return responses, nil
}

func (m *MockURLService) GetUserURLs(ctx context.Context, userID string) ([]usecase.UserURL, error) {
	if m.GetUserURLsFunc != nil {
		return m.GetUserURLsFunc(ctx, userID)
	}
	return nil, nil
}

func TestHTTPController_handleShorten(t *testing.T) {
	tests := []struct {
		name           string
		mockService    *MockURLService
		requestBody    string
		expectedStatus int
		expectedBody   string
	}{
		{
			name: "successful shortening",
			mockService: &MockURLService{
				ShortenFunc: func(url string) (string, error) {
					return "abc123", nil
				},
			},
			requestBody:    "https://example.com",
			expectedStatus: http.StatusCreated,
			expectedBody:   "abc123",
		},
		{
			name: "empty body",
			mockService: &MockURLService{
				ShortenFunc: func(url string) (string, error) {
					return "unreachable", nil
				},
			},
			requestBody:    "",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Empty short url\n",
		},
		{
			name: "service error",
			mockService: &MockURLService{
				ShortenFunc: func(url string) (string, error) {
					return "", errors.New("storage error")
				},
			},
			requestBody:    "https://example.com",
			expectedStatus: http.StatusBadRequest,
			expectedBody:   "Shorten failed\n",
		},
		{
			name: "URL conflict",
			mockService: &MockURLService{
				ShortenFunc: func(url string) (string, error) {
					return "http://localhost:8080/existing123", &usecase.ErrURLConflict{
						ExistingShortURL: "http://localhost:8080/existing123",
					}
				},
			},
			requestBody:    "https://example.com",
			expectedStatus: http.StatusConflict,
			expectedBody:   "http://localhost:8080/existing123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := middleware.NewAuthMiddleware("test-key")
			require.NoError(t, err)
			controller := NewHTTPController(tt.mockService, auth)

			req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(tt.requestBody))
			w := httptest.NewRecorder()

			controller.handleShorten(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedBody, w.Body.String())
		})
	}
}

func TestHTTPController_handleRedirect(t *testing.T) {
	tests := []struct {
		name           string
		mockService    *MockURLService
		shortID        string
		expectedStatus int
		expectedLoc    string
	}{
		{
			name: "successful redirect",
			mockService: &MockURLService{
				ExpandFunc: func(shortID string) (string, error) {
					return "https://original.url", nil
				},
			},
			shortID:        "abc123",
			expectedStatus: http.StatusTemporaryRedirect,
			expectedLoc:    "https://original.url",
		},
		{
			name: "not found",
			mockService: &MockURLService{
				ExpandFunc: func(shortID string) (string, error) {
					return "", errors.New("not found")
				},
			},
			shortID:        "invalid",
			expectedStatus: http.StatusBadRequest,
			expectedLoc:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := middleware.NewAuthMiddleware("test-key")
			require.NoError(t, err)
			controller := NewHTTPController(tt.mockService, auth)

			req := httptest.NewRequest(http.MethodGet, "/"+tt.shortID, nil)
			w := httptest.NewRecorder()

			controller.handleRedirect(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedLoc, w.Header().Get("Location"))
		})
	}
}

func TestHandleShortenJSON(t *testing.T) {
	tests := []struct {
		name           string
		request        ShortenRequest
		mockResponse   string
		mockError      error
		expectedStatus int
		expectedResult string
	}{
		{
			name: "успешное сокращение URL",
			request: ShortenRequest{
				URL: "https://practicum.yandex.ru",
			},
			mockResponse:   "http://localhost:8080/abc123",
			mockError:      nil,
			expectedStatus: http.StatusCreated,
			expectedResult: "http://localhost:8080/abc123",
		},
		{
			name: "пустой URL",
			request: ShortenRequest{
				URL: "",
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "конфликт URL",
			request: ShortenRequest{
				URL: "https://practicum.yandex.ru",
			},
			mockResponse: "http://localhost:8080/existing123",
			mockError: &usecase.ErrURLConflict{
				ExistingShortURL: "http://localhost:8080/existing123",
			},
			expectedStatus: http.StatusConflict,
			expectedResult: "http://localhost:8080/existing123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockURLService{
				ShortenFunc: func(url string) (string, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			auth, err := middleware.NewAuthMiddleware("test-key")
			require.NoError(t, err)
			controller := NewHTTPController(mockService, auth)

			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			controller.handleShortenJSON(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated || tt.expectedStatus == http.StatusConflict {
				var response ShortenResponse
				err := json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, response.Result)
			}
		})
	}
}

func TestHTTPController_handlePing(t *testing.T) {
	tests := []struct {
		name           string
		mockService    *MockURLService
		expectedStatus int
	}{
		{
			name: "successful ping",
			mockService: &MockURLService{
				PingDBFunc: func() error {
					return nil
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "ping error",
			mockService: &MockURLService{
				PingDBFunc: func() error {
					return errors.New("database connection failed")
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := middleware.NewAuthMiddleware("test-key")
			require.NoError(t, err)
			controller := NewHTTPController(tt.mockService, auth)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()

			controller.handlePing(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestHTTPController_handleShortenBatch(t *testing.T) {
	tests := []struct {
		name           string
		requests       []usecase.BatchShortenRequest
		mockService    *MockURLService
		expectedStatus int
		expectedCount  int
		requestBody    string
		invalidJSON    bool
	}{
		{
			name: "успешный batch запрос",
			requests: []usecase.BatchShortenRequest{
				{CorrelationID: "1", OriginalURL: "https://example.com"},
				{CorrelationID: "2", OriginalURL: "https://google.com"},
			},
			mockService: &MockURLService{
				ShortenBatchFunc: func(ctx context.Context, requests []usecase.BatchShortenRequest) ([]usecase.BatchShortenResponse, error) {
					responses := make([]usecase.BatchShortenResponse, len(requests))
					for i, req := range requests {
						responses[i] = usecase.BatchShortenResponse{
							CorrelationID: req.CorrelationID,
							ShortURL:      "http://localhost:8080/test" + string(rune(i+'1')),
						}
					}
					return responses, nil
				},
			},
			expectedStatus: http.StatusCreated,
			expectedCount:  2,
		},
		{
			name:           "пустой batch запрос",
			requests:       []usecase.BatchShortenRequest{},
			mockService:    &MockURLService{},
			expectedStatus: http.StatusBadRequest,
			expectedCount:  0,
		},
		{
			name:           "невалидный JSON",
			mockService:    &MockURLService{},
			expectedStatus: http.StatusBadRequest,
			requestBody:    "invalid json",
			invalidJSON:    true,
		},
		{
			name: "ошибка сервиса",
			requests: []usecase.BatchShortenRequest{
				{CorrelationID: "1", OriginalURL: "https://example.com"},
			},
			mockService: &MockURLService{
				ShortenBatchFunc: func(ctx context.Context, requests []usecase.BatchShortenRequest) ([]usecase.BatchShortenResponse, error) {
					return nil, errors.New("service error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			expectedCount:  0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := middleware.NewAuthMiddleware("test-key")
			require.NoError(t, err)
			controller := NewHTTPController(tt.mockService, auth)

			var body []byte
			var err2 error

			if tt.invalidJSON {
				body = []byte(tt.requestBody)
			} else {
				body, err2 = json.Marshal(tt.requests)
				require.NoError(t, err2)
			}

			req := httptest.NewRequest(http.MethodPost, "/api/shorten/batch", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")

			rr := httptest.NewRecorder()
			controller.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.expectedStatus == http.StatusCreated {
				assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

				var responses []usecase.BatchShortenResponse
				err2 = json.Unmarshal(rr.Body.Bytes(), &responses)
				require.NoError(t, err2)

				assert.Len(t, responses, tt.expectedCount)
				for i, response := range responses {
					assert.Equal(t, tt.requests[i].CorrelationID, response.CorrelationID)
					assert.NotEmpty(t, response.ShortURL)
				}
			}
		})
	}
}

func TestHTTPController_handleGetUserURLs(t *testing.T) {
	tests := []struct {
		name           string
		mockService    *MockURLService
		expectedStatus int
		expectedURLs   []usecase.UserURL
	}{
		{
			name: "успешное получение URL пользователя",
			mockService: &MockURLService{
				GetUserURLsFunc: func(ctx context.Context, userID string) ([]usecase.UserURL, error) {
					return []usecase.UserURL{
						{
							ShortURL:    "http://localhost:8080/abc123",
							OriginalURL: "https://example.com",
						},
						{
							ShortURL:    "http://localhost:8080/def456",
							OriginalURL: "https://google.com",
						},
					}, nil
				},
			},
			expectedStatus: http.StatusOK,
			expectedURLs: []usecase.UserURL{
				{
					ShortURL:    "http://localhost:8080/abc123",
					OriginalURL: "https://example.com",
				},
				{
					ShortURL:    "http://localhost:8080/def456",
					OriginalURL: "https://google.com",
				},
			},
		},
		{
			name: "нет URL у пользователя",
			mockService: &MockURLService{
				GetUserURLsFunc: func(ctx context.Context, userID string) ([]usecase.UserURL, error) {
					return nil, nil
				},
			},
			expectedStatus: http.StatusNoContent,
			expectedURLs:   nil,
		},
		{
			name: "ошибка получения URL",
			mockService: &MockURLService{
				GetUserURLsFunc: func(ctx context.Context, userID string) ([]usecase.UserURL, error) {
					return nil, errors.New("storage error")
				},
			},
			expectedStatus: http.StatusInternalServerError,
			expectedURLs:   nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth, err := middleware.NewAuthMiddleware("test-key")
			require.NoError(t, err)
			controller := NewHTTPController(tt.mockService, auth)

			req := httptest.NewRequest(http.MethodGet, "/api/user/urls", nil)

			// Добавляем валидную куку пользователя
			testUserID := "test-user-123"
			err = auth.SetUserID(httptest.NewRecorder(), testUserID)
			require.NoError(t, err)

			// Получаем зашифрованную куку
			tempW := httptest.NewRecorder()
			err = auth.SetUserID(tempW, testUserID)
			require.NoError(t, err)

			result := tempW.Result()
			cookies := result.Cookies()
			defer result.Body.Close()
			require.Len(t, cookies, 1)
			req.AddCookie(cookies[0])

			w := httptest.NewRecorder()

			controller.handleGetUserURLs(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusOK {
				var urls []usecase.UserURL
				err := json.NewDecoder(w.Body).Decode(&urls)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedURLs, urls)
			}
		})
	}
}
