package usecase

import (
	"errors"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

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

func TestURLService_Shorten(t *testing.T) {
	tests := []struct {
		name    string
		storage *MockURLStorage
		url     string
		want    string
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
			s := &URLService{
				storage: tt.storage,
			}
			got, err := s.Shorten(tt.url)
			if (err != nil) != tt.wantErr {
				t.Errorf("URLService.Shorten() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if !tt.wantErr {
				assert.True(t, strings.HasPrefix(got, baseURL))
				assert.Len(t, strings.TrimPrefix(got, baseURL), 8)
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
			s := &URLService{
				storage: tt.storage,
			}
			got, err := s.Expand(tt.shortID)
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
