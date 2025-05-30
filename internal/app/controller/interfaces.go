package controller

import "github.com/m-molecula741/shortener/internal/app/usecase"

type URLService interface {
	Shorten(url string) (string, error)
	Expand(shortID string) (string, error)
	PingDB() error
	ShortenBatch(requests []usecase.BatchShortenRequest) ([]usecase.BatchShortenResponse, error)
}
