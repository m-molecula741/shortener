package usecase

import (
	"crypto/rand"
	"encoding/base64"
	"strings"
)

type URLService struct {
	storage URLStorage
	baseURL string
}

func NewURLService(storage URLStorage, baseURL string) *URLService {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL + "/"
	}

	return &URLService{
		storage: storage,
		baseURL: baseURL,
	}
}

func (s *URLService) Shorten(url string) (string, error) {
	shortID, err := generateShortID()
	if err != nil {
		return "", err
	}

	if err := s.storage.Save(shortID, url); err != nil {
		return "", err
	}

	return s.baseURL + shortID, nil
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
