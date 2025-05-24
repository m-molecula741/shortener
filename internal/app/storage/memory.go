package storage

import (
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
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

	// Очищаем файл перед сохранением
	if err := s.backup.Clear(); err != nil {
		return fmt.Errorf("cannot clear backup file: %w", err)
	}

	// Сохраняем все URL
	for shortID, url := range s.urls {
		uuid := uuid.New().String()
		if err := s.backup.SaveURL(uuid, shortID, url); err != nil {
			return fmt.Errorf("cannot backup URL: %w", err)
		}
	}

	return nil
}

var (
	ErrNotFound = errors.New("url not found")
)
