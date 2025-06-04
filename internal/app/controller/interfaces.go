package controller

import (
	"context"

	"github.com/m-molecula741/shortener/internal/app/usecase"
)

type URLService interface {
	Shorten(url string) (string, error)
	Expand(shortID string) (string, error)
	PingDB() error
	ShortenBatch(ctx context.Context, requests []usecase.BatchShortenRequest) ([]usecase.BatchShortenResponse, error)
}
