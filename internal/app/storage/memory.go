package storage

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/m-molecula741/shortener/internal/app/usecase"
)

type InMemoryStorage struct {
	mu     sync.Mutex
	urls   map[string]string
	backup *FileBackup
}

func NewInMemoryStorage(filePath string) (*InMemoryStorage, error) {
	backup := NewFileBackup(filePath)

	// Создаем хранилище
	s := &InMemoryStorage{
		urls:   make(map[string]string),
		backup: backup,
	}

	// Загружаем существующие URL из файла
	if urls, err := backup.LoadURLs(); err != nil {
		return nil, err
	} else {
		s.urls = urls
	}

	return s, nil
}

func (s *InMemoryStorage) Save(shortID, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.urls[shortID] = url
	return nil
}

func (s *InMemoryStorage) Get(shortID string) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	url, exists := s.urls[shortID]
	if !exists {
		return "", ErrNotFound
	}
	return url, nil
}

// Backup сохраняет все URL в файл
func (s *InMemoryStorage) Backup() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Сохраняем все URL
	for shortID, url := range s.urls {
		// Генерируем UUID только для новых записей, если запись уже есть в файле - используем существующий UUID
		if err := s.backup.SaveURL(uuid.New().String(), shortID, url); err != nil {
			return fmt.Errorf("cannot backup URL: %w", err)
		}
	}

	return nil
}

var (
	ErrNotFound = errors.New("url not found")
)

// SaveBatch сохраняет множество URL за одну операцию
func (s *InMemoryStorage) SaveBatch(urls []usecase.URLPair) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Сохраняем в память
	for _, url := range urls {
		s.urls[url.ShortID] = url.OriginalURL
	}

	return nil
}
