package usecase

import (
	"crypto/rand"
	"encoding/base64"
)

const baseURL = "http://localhost:8080/"

type URLService struct {
	storage URLStorage
}

func NewURLService(storage URLStorage) *URLService {
	return &URLService{storage: storage}
}

func (s *URLService) Shorten(url string) (string, error) {
	shortID, err := generateShortID()
	if err != nil {
		return "", err
	}

	if err := s.storage.Save(shortID, url); err != nil {
		return "", err
	}

	return baseURL + shortID, nil
}

func (s *URLService) Expand(shortID string) (string, error) {
	return s.storage.Get(shortID)
}

func generateShortID() (string, error) {
    b := make([]byte, 6)
    if _, err := rand.Read(b); err != nil {
        return "", err
    }
    return base64.URLEncoding.EncodeToString(b)[:8], nil
}
