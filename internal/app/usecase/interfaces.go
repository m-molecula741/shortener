package usecase

import "context"

type URLStorage interface {
	Save(shortID, url string) error
	Get(shortID string) (string, error)
	SaveBatch(ctx context.Context, urls []URLPair) error
}

type DatabasePinger interface {
	Ping() error
	Close()
}
