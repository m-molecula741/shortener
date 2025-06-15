package usecase

// URLPair пара URL для batch операций
type URLPair struct {
	ShortID     string
	OriginalURL string
	UserID      string
}

// BatchShortenRequest запрос на сокращение URL в batch режиме
type BatchShortenRequest struct {
	CorrelationID string `json:"correlation_id"`
	OriginalURL   string `json:"original_url"`
}

// BatchShortenResponse ответ на batch запрос
type BatchShortenResponse struct {
	CorrelationID string `json:"correlation_id"`
	ShortURL      string `json:"short_url"`
}

// UserURL представляет URL пользователя
type UserURL struct {
	ShortURL    string `json:"short_url"`
	OriginalURL string `json:"original_url"`
}
