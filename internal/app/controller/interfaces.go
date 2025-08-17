// Package controller предоставляет интерфейсы для контроллеров
package controller

import (
	"context"

	"github.com/m-molecula741/shortener/internal/app/usecase"
)

// URLService определяет интерфейс для сервиса URL
type URLService interface {
	Shorten(url string) (string, error)
	ShortenWithUser(ctx context.Context, url, userID string) (string, error)
	Expand(shortID string) (string, error)
	PingDB() error
	ShortenBatch(ctx context.Context, requests []usecase.BatchShortenRequest) ([]usecase.BatchShortenResponse, error)
	ShortenBatchWithUser(ctx context.Context, requests []usecase.BatchShortenRequest, userID string) ([]usecase.BatchShortenResponse, error)
	GetUserURLs(ctx context.Context, userID string) ([]usecase.UserURL, error)
	DeleteUserURLs(userID string, shortIDs []string) error
}
