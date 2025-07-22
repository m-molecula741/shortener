package usecase

import (
	"context"
	"errors"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testBaseURL = "http://localhost:8080/"

type MockURLStorage struct {
	SaveFunc                func(shortID, url string) error
	GetFunc                 func(shortID string) (string, error)
	SaveBatchFunc           func(ctx context.Context, urls []URLPair) error
	GetUserURLsFunc         func(ctx context.Context, userID string) ([]UserURL, error)
	BatchDeleteUserURLsFunc func(ctx context.Context, userID string, shortIDs []string) error
	SaveBatchCallCount      int
	LastSavedBatch          []URLPair
}

func (m *MockURLStorage) Save(shortID, url string) error {
	if m.SaveFunc != nil {
		return m.SaveFunc(shortID, url)
	}
	return nil
}

func (m *MockURLStorage) Get(shortID string) (string, error) {
	if m.GetFunc != nil {
		return m.GetFunc(shortID)
	}
	return "", errors.New("not implemented")
}

func (m *MockURLStorage) SaveBatch(ctx context.Context, urls []URLPair) error {
	m.SaveBatchCallCount++
	m.LastSavedBatch = urls
	if m.SaveBatchFunc != nil {
		return m.SaveBatchFunc(ctx, urls)
	}
	return nil
}

func (m *MockURLStorage) GetUserURLs(ctx context.Context, userID string) ([]UserURL, error) {
	if m.GetUserURLsFunc != nil {
		return m.GetUserURLsFunc(ctx, userID)
	}
	return nil, nil
}

func (m *MockURLStorage) BatchDeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error {
	if m.BatchDeleteUserURLsFunc != nil {
		return m.BatchDeleteUserURLsFunc(ctx, userID, shortIDs)
	}
	return nil
}

// MockDatabasePinger мок для DatabasePinger
type MockDatabasePinger struct {
	PingFunc  func() error
	CloseFunc func() error
}

func (m *MockDatabasePinger) Ping() error {
	if m.PingFunc != nil {
		return m.PingFunc()
	}
	return nil
}

func (m *MockDatabasePinger) Close() error {
	if m.CloseFunc != nil {
		return m.CloseFunc()
	}
	return nil
}

func TestURLService_Shorten(t *testing.T) {
	tests := []struct {
		name             string
		storage          *MockURLStorage
		url              string
		wantErr          bool
		wantConflict     bool
		expectedShortURL string
	}{
		{
			name: "успешное сокращение URL",
			storage: &MockURLStorage{
				SaveFunc: func(shortID, url string) error {
					return nil
				},
			},
			url:     "https://example.com",
			wantErr: false,
		},
		{
			name: "ошибка сохранения",
			storage: &MockURLStorage{
				SaveFunc: func(shortID, url string) error {
					return errors.New("storage error")
				},
			},
			url:     "https://example.com",
			wantErr: true,
		},
		{
			name: "конфликт URL - URL уже существует",
			storage: &MockURLStorage{
				SaveFunc: func(shortID, url string) error {
					return &ErrURLConflict{ExistingShortURL: "existing123"}
				},
			},
			url:              "https://example.com",
			wantErr:          true,
			wantConflict:     true,
			expectedShortURL: testBaseURL + "existing123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewURLService(tt.storage, testBaseURL, nil)
			got, err := service.Shorten(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("URLService.Shorten() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantConflict {
				conflictErr, isConflict := IsURLConflict(err)
				assert.True(t, isConflict)
				assert.Equal(t, tt.expectedShortURL, conflictErr.ExistingShortURL)
				assert.Equal(t, tt.expectedShortURL, got)
			} else if !tt.wantErr {
				assert.True(t, strings.HasPrefix(got, testBaseURL))
				_, shortID := path.Split(got)
				assert.Len(t, shortID, 8)
			}
		})
	}
}

func TestURLService_Expand(t *testing.T) {
	tests := []struct {
		name    string
		storage *MockURLStorage
		shortID string
		want    string
		wantErr bool
	}{
		{
			name: "успешное получение оригинального URL",
			storage: &MockURLStorage{
				GetFunc: func(shortID string) (string, error) {
					return "https://example.com", nil
				},
			},
			shortID: "abc123",
			want:    "https://example.com",
			wantErr: false,
		},
		{
			name: "URL не найден",
			storage: &MockURLStorage{
				GetFunc: func(shortID string) (string, error) {
					return "", errors.New("not found")
				},
			},
			shortID: "notfound",
			want:    "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewURLService(tt.storage, testBaseURL, nil)
			got, err := service.Expand(tt.shortID)
			if (err != nil) != tt.wantErr {
				t.Errorf("URLService.Expand() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("URLService.Expand() = %v, want %v", got, tt.want)
			}
		})
	}
}

func Test_generateShortID(t *testing.T) {
	tests := []struct {
		name        string
		wantLen     int
		wantErr     bool
		checkUnique bool
	}{
		{
			name:        "генерация короткого ID",
			wantLen:     8,
			wantErr:     false,
			checkUnique: false,
		},
		{
			name:        "уникальность генерации",
			wantLen:     8,
			wantErr:     false,
			checkUnique: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := generateShortID()
			if (err != nil) != tt.wantErr {
				t.Errorf("generateShortID() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			assert.Len(t, got, tt.wantLen)
			assert.NotEmpty(t, got)

			if tt.checkUnique {
				got2, err2 := generateShortID()
				assert.NoError(t, err2)
				assert.NotEqual(t, got, got2)
			}
		})
	}
}

func TestURLService_PingDB(t *testing.T) {
	tests := []struct {
		name     string
		dbPinger DatabasePinger
		wantErr  bool
	}{
		{
			name: "успешный пинг базы данных",
			dbPinger: &MockDatabasePinger{
				PingFunc: func() error {
					return nil
				},
			},
			wantErr: false,
		},
		{
			name: "ошибка пинга базы данных",
			dbPinger: &MockDatabasePinger{
				PingFunc: func() error {
					return errors.New("connection failed")
				},
			},
			wantErr: true,
		},
		{
			name:     "пингер не настроен (nil)",
			dbPinger: nil,
			wantErr:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			storage := &MockURLStorage{}
			service := NewURLService(storage, testBaseURL, tt.dbPinger)

			if err := service.PingDB(); (err != nil) != tt.wantErr {
				t.Errorf("URLService.PingDB() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestURLService_ShortenBatch(t *testing.T) {
	tests := []struct {
		name                string
		requests            []BatchShortenRequest
		mockSaveBatchFunc   func(ctx context.Context, urls []URLPair) error
		wantErr             bool
		expectedCallCount   int
		expectedBatchLength int
	}{
		{
			name: "успешный batch запрос",
			requests: []BatchShortenRequest{
				{CorrelationID: "1", OriginalURL: "https://example.com"},
				{CorrelationID: "2", OriginalURL: "https://google.com"},
				{CorrelationID: "3", OriginalURL: "https://github.com"},
			},
			mockSaveBatchFunc: func(ctx context.Context, urls []URLPair) error {
				return nil
			},
			wantErr:             false,
			expectedCallCount:   1,
			expectedBatchLength: 3,
		},
		{
			name:                "пустой batch запрос",
			requests:            []BatchShortenRequest{},
			mockSaveBatchFunc:   nil,
			wantErr:             false,
			expectedCallCount:   0,
			expectedBatchLength: 0,
		},
		{
			name: "ошибка сохранения",
			requests: []BatchShortenRequest{
				{CorrelationID: "1", OriginalURL: "https://example.com"},
			},
			mockSaveBatchFunc: func(ctx context.Context, urls []URLPair) error {
				return errors.New("storage error")
			},
			wantErr:             true,
			expectedCallCount:   1,
			expectedBatchLength: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockStorage := &MockURLStorage{
				SaveBatchFunc: tt.mockSaveBatchFunc,
			}
			service := NewURLService(mockStorage, "http://localhost:8080/", nil)

			responses, err := service.ShortenBatch(context.Background(), tt.requests)

			if (err != nil) != tt.wantErr {
				t.Errorf("URLService.ShortenBatch() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				assert.Len(t, responses, len(tt.requests))

				// Проверяем, что correlation_id сохранились
				for i, response := range responses {
					assert.Equal(t, tt.requests[i].CorrelationID, response.CorrelationID)
					assert.Contains(t, response.ShortURL, "http://localhost:8080/")
					assert.NotEmpty(t, response.ShortURL)
				}
			}

			// Проверяем, что SaveBatch был вызван правильное количество раз
			assert.Equal(t, tt.expectedCallCount, mockStorage.SaveBatchCallCount)
			if tt.expectedCallCount > 0 {
				assert.Len(t, mockStorage.LastSavedBatch, tt.expectedBatchLength)
			}
		})
	}
}

func TestURLService_ShortenWithUser(t *testing.T) {
	tests := []struct {
		name             string
		storage          *MockURLStorage
		url              string
		userID           string
		wantErr          bool
		wantConflict     bool
		expectedShortURL string
	}{
		{
			name: "успешное сокращение URL с пользователем",
			storage: &MockURLStorage{
				SaveFunc: func(shortID, url string) error {
					return nil
				},
				SaveBatchFunc: func(ctx context.Context, urls []URLPair) error {
					return nil
				},
			},
			url:     "https://example.com",
			userID:  "user123",
			wantErr: false,
		},
		{
			name: "конфликт URL - URL уже существует",
			storage: &MockURLStorage{
				SaveFunc: func(shortID, url string) error {
					return &ErrURLConflict{ExistingShortURL: "existing123"}
				},
			},
			url:              "https://example.com",
			userID:           "user123",
			wantErr:          true,
			wantConflict:     true,
			expectedShortURL: testBaseURL + "existing123",
		},
		{
			name: "успешное сокращение URL без пользователя",
			storage: &MockURLStorage{
				SaveFunc: func(shortID, url string) error {
					return nil
				},
			},
			url:     "https://example.com",
			userID:  "",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewURLService(tt.storage, testBaseURL, nil)
			got, err := service.ShortenWithUser(context.Background(), tt.url, tt.userID)
			if (err != nil) != tt.wantErr {
				t.Errorf("URLService.ShortenWithUser() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if tt.wantConflict {
				conflictErr, isConflict := IsURLConflict(err)
				assert.True(t, isConflict)
				assert.Equal(t, tt.expectedShortURL, conflictErr.ExistingShortURL)
				assert.Equal(t, tt.expectedShortURL, got)
			} else if !tt.wantErr {
				assert.True(t, strings.HasPrefix(got, testBaseURL))
				_, shortID := path.Split(got)
				assert.Len(t, shortID, 8)
			}
		})
	}
}

func BenchmarkURLService_Shorten(b *testing.B) {
	storage := &MockURLStorage{
		SaveFunc: func(shortID, url string) error {
			return nil
		},
	}
	service := NewURLService(storage, "http://localhost:8080", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Shorten("http://example.com")
	}
}

func BenchmarkURLService_Expand(b *testing.B) {
	storage := &MockURLStorage{
		SaveFunc: func(shortID, url string) error {
			return nil
		},
		GetFunc: func(shortID string) (string, error) {
			return "http://example.com", nil
		},
	}
	service := NewURLService(storage, "http://localhost:8080", nil)

	shortURL, _ := service.Shorten("http://example.com")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.Expand(shortURL)
	}
}

func BenchmarkURLService_ShortenBatch(b *testing.B) {
	storage := &MockURLStorage{
		SaveBatchFunc: func(ctx context.Context, urls []URLPair) error {
			return nil
		},
	}
	service := NewURLService(storage, "http://localhost:8080", nil)

	requests := []BatchShortenRequest{
		{CorrelationID: "1", OriginalURL: "http://example1.com"},
		{CorrelationID: "2", OriginalURL: "http://example2.com"},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.ShortenBatch(context.Background(), requests)
	}
}
