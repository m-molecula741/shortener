package storage

import (
	"errors"
	"sync"
)

type InMemoryStorage struct {
	mu   sync.Mutex
	urls map[string]string
}

func NewInMemoryStorage() *InMemoryStorage {
	return &InMemoryStorage{
		urls: make(map[string]string),
	}
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

var (
	ErrNotFound = errors.New("url not found")
)
