package usecase

type URLStorage interface {
	Save(shortID, url string) error
	Get(shortID string) (string, error)
}
