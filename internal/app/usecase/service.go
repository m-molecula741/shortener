package usecase

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"strings"
)

// Структуры для batch операций
type URLPair struct {
	ShortID     string
	OriginalURL string
}

type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

type URLService struct {
	storage  URLStorage
	baseURL  string
	dbPinger DatabasePinger
}

func NewURLService(storage URLStorage, baseURL string, dbPinger DatabasePinger) *URLService {
	if !strings.HasSuffix(baseURL, "/") {
		baseURL = baseURL + "/"
	}

	return &URLService{
		storage:  storage,
		baseURL:  baseURL,
		dbPinger: dbPinger,
	}
}

func (s *URLService) Shorten(url string) (string, error) {
	shortID, err := generateShortID()
	if err != nil {
		return "", err
	}

	if err := s.storage.Save(shortID, url); err != nil {
		if conflictErr, isConflict := IsURLConflict(err); isConflict {
			return s.baseURL + conflictErr.ExistingShortURL, &ErrURLConflict{
				ExistingShortURL: s.baseURL + conflictErr.ExistingShortURL,
			}
		}
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

// PingDB проверяет соединение с базой данных
func (s *URLService) PingDB() error {
	if s.dbPinger == nil {
		return nil // если пингер не настроен, возвращаем nil
	}
	return s.dbPinger.Ping()
}

// ShortenBatch сокращает множество URL за одну операцию
func (s *URLService) ShortenBatch(ctx context.Context, requests []BatchShortenRequest) ([]BatchShortenResponse, error) {
	if len(requests) == 0 {
		return []BatchShortenResponse{}, nil
	}

	// Подготавливаем данные для batch сохранения
	urlPairs := make([]URLPair, len(requests))
	responses := make([]BatchShortenResponse, len(requests))

	for i, req := range requests {
		shortID, err := generateShortID()
		if err != nil {
			return nil, err
		}

		urlPairs[i] = URLPair{
			ShortID:     shortID,
			OriginalURL: req.OriginalURL,
		}

		responses[i] = BatchShortenResponse{
			CorrelationID: req.CorrelationID,
			ShortURL:      s.baseURL + shortID,
		}
	}

	// Сохраняем все URL одной операцией
	if err := s.storage.SaveBatch(ctx, urlPairs); err != nil {
		return nil, err
	}

	return responses, nil
}
