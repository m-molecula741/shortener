package usecase

import (
	"errors"
	"path"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

const testBaseURL = "http://localhost:8080/"

type MockURLStorage struct {
	SaveFunc func(shortID, url string) error
	GetFunc  func(shortID string) (string, error)
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

// MockDatabasePinger мок для DatabasePinger
type MockDatabasePinger struct {
	PingFunc  func() error
	CloseFunc func()
}

func (m *MockDatabasePinger) Ping() error {
	if m.PingFunc != nil {
		return m.PingFunc()
	}
	return nil
}

func (m *MockDatabasePinger) Close() {
	if m.CloseFunc != nil {
		m.CloseFunc()
	}
}

func TestURLService_Shorten(t *testing.T) {
	tests := []struct {
		name    string
		storage *MockURLStorage
		url     string
		wantErr bool
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
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			service := NewURLService(tt.storage, testBaseURL, nil)
			got, err := service.Shorten(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("URLService.Shorten() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
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
	t.Run("генерация короткого ID", func(t *testing.T) {
		got, err := generateShortID()
		assert.NoError(t, err)
		assert.Len(t, got, 8)
	})

	t.Run("уникальность генерации", func(t *testing.T) {
		id1, err1 := generateShortID()
		assert.NoError(t, err1)

		id2, err2 := generateShortID()
		assert.NoError(t, err2)

		assert.NotEqual(t, id1, id2)
	})
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
