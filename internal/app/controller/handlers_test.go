package controller

import (
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
)

type MockURLService struct {
	ShortenFunc func(url string) (string, error)
	ExpandFunc  func(shortID string) (string, error)
}

func (m *MockURLService) Shorten(url string) (string, error) {
	return m.ShortenFunc(url)
}

func (m *MockURLService) Expand(shortID string) (string, error) {
	return m.ExpandFunc(shortID)
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
			expectedBody:   "Bad request\n",
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
			expectedBody:   "Internal error\n",
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
