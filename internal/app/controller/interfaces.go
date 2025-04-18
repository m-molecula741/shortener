package controller

type UrlService interface {
	Shorten(url string) (string, error)
	Expand(shortID string) (string, error)
}
