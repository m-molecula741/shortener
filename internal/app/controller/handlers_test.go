package controller

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// MockURLService мок для URLService
type MockURLService struct {
	ShortenFunc func(url string) (string, error)
	ExpandFunc  func(shortID string) (string, error)
	PingDBFunc  func() error
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			controller := NewHTTPController(tt.mockService)

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
			controller := NewHTTPController(tt.mockService)

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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockService := &MockURLService{
				ShortenFunc: func(url string) (string, error) {
					return tt.mockResponse, tt.mockError
				},
			}

			controller := NewHTTPController(mockService)

			reqBody, err := json.Marshal(tt.request)
			require.NoError(t, err)

			req := httptest.NewRequest(http.MethodPost, "/api/shorten", bytes.NewBuffer(reqBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()

			controller.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)

			if tt.expectedStatus == http.StatusCreated {
				var response ShortenResponse
				err = json.NewDecoder(w.Body).Decode(&response)
				require.NoError(t, err)
				assert.Equal(t, tt.expectedResult, response.Result)
			}
		})
	}
}

func TestHTTPController_handlePing(t *testing.T) {
	tests := []struct {
		name           string
		service        URLService
		expectedStatus int
	}{
		{
			name: "успешный пинг базы данных",
			service: &MockURLService{
				PingDBFunc: func() error {
					return nil
				},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "ошибка пинга базы данных",
			service: &MockURLService{
				PingDBFunc: func() error {
					return errors.New("database connection failed")
				},
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c := NewHTTPController(tt.service)

			req := httptest.NewRequest(http.MethodGet, "/ping", nil)
			w := httptest.NewRecorder()

			c.handlePing(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}
