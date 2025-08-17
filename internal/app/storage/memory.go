// Package storage предоставляет различные реализации хранилища URL
package storage

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/google/uuid"
	"github.com/m-molecula741/shortener/internal/app/usecase"
)

// InMemoryStorage представляет хранилище URL в памяти
type InMemoryStorage struct {
	mu     sync.Mutex
	urls   map[string]string
	users  map[string][]string // userID -> []shortID
	backup *FileBackup
}

// NewInMemoryStorage создает новый экземпляр InMemoryStorage
func NewInMemoryStorage(filePath string) (*InMemoryStorage, error) {
	backup := NewFileBackup(filePath)

	// Создаем хранилище
	s := &InMemoryStorage{
		urls:   make(map[string]string),
		users:  make(map[string][]string),
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

// Save сохраняет URL в памяти
func (s *InMemoryStorage) Save(shortID, url string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Проверяем, есть ли уже такой URL
	for existingShortID, existingURL := range s.urls {
		if existingURL == url {
			return &usecase.ErrURLConflict{ExistingShortURL: existingShortID}
		}
	}

	s.urls[shortID] = url
	return nil
}

// Get получает URL из памяти
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

// Ошибки для хранилища
var (
	ErrNotFound = errors.New("url not found")
)

// SaveBatch сохраняет множество URL за одну операцию
func (s *InMemoryStorage) SaveBatch(ctx context.Context, urls []usecase.URLPair) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Сохраняем в память
	for _, url := range urls {
		// Сохраняем URL если его еще нет
		if _, exists := s.urls[url.ShortID]; !exists {
			s.urls[url.ShortID] = url.OriginalURL
		}

		// Связываем с пользователем если указан userID
		if url.UserID != "" {
			found := false
			if shortIDs, exists := s.users[url.UserID]; exists {
				for _, existingShortID := range shortIDs {
					if existingShortID == url.ShortID {
						found = true
						break
					}
				}
			}

			// Добавляем связь если еще не существует
			if !found {
				s.users[url.UserID] = append(s.users[url.UserID], url.ShortID)
			}
		}
	}

	return nil
}

// GetUserURLs получает все URL пользователя
func (s *InMemoryStorage) GetUserURLs(ctx context.Context, userID string) ([]usecase.UserURL, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	shortIDs, exists := s.users[userID]
	if !exists {
		return nil, nil
	}

	urls := make([]usecase.UserURL, 0, len(shortIDs))
	for _, shortID := range shortIDs {
		originalURL, exists := s.urls[shortID]
		if !exists {
			continue
		}

		urls = append(urls, usecase.UserURL{
			ShortURL:    fmt.Sprintf("http://localhost:8080/%s", shortID),
			OriginalURL: originalURL,
		})
	}

	return urls, nil
}

// BatchDeleteUserURLs помечает URL пользователя как удаленные
func (s *InMemoryStorage) BatchDeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, shortID := range shortIDs {
		if url, exists := s.urls[shortID]; exists {
			if urlWithUser, hasUser := s.users[userID]; hasUser {
				for i, userURL := range urlWithUser {
					if userURL == shortID {
						// Удаляем из списка пользователя
						s.users[userID] = append(urlWithUser[:i], urlWithUser[i+1:]...)
						break
					}
				}
			}
			delete(s.urls, shortID)
			_ = url
		}
	}

	return nil
}
