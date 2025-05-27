package usecase

type URLStorage interface {
	Save(shortID, url string) error
	Get(shortID string) (string, error)
}

type DatabasePinger interface {
	Ping() error
	Close()
}
