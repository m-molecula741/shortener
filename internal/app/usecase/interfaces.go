package usecase

type URLStorage interface {
	Save(shortID, url string) error
	Get(shortID string) (string, error)
	SaveBatch(urls []URLPair) error
}

type DatabasePinger interface {
	Ping() error
	Close()
}
