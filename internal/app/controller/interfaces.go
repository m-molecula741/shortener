package controller

type URLService interface {
	Shorten(url string) (string, error)
	Expand(shortID string) (string, error)
	PingDB() error
}
