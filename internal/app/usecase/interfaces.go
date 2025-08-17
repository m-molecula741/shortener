// Package usecase предоставляет интерфейсы для бизнес-логики
package usecase

import "context"

// URLStorage определяет интерфейс для хранилища URL
type URLStorage interface {
	Save(shortID, url string) error
	Get(shortID string) (string, error)
	SaveBatch(ctx context.Context, urls []URLPair) error
	GetUserURLs(ctx context.Context, userID string) ([]UserURL, error)
	BatchDeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error
}

// DatabasePinger определяет интерфейс для проверки соединения с базой данных
type DatabasePinger interface {
	Ping() error
	Close() error
}
