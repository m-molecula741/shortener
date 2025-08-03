package middleware

import (
	"net/http"
	"time"

	"github.com/m-molecula741/shortener/internal/app/logger"
)

type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

func RequestLogger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()

		// Создаем обертку для ResponseWriter, чтобы отслеживать статус и размер ответа
		wrapped := &responseWriter{
			ResponseWriter: w,
			status:         http.StatusOK,
		}

		// Выполняем запрос
		next.ServeHTTP(wrapped, r)

		// Логируем информацию о запросе и ответе
		logger.Info().
			Str("method", r.Method).
			Str("uri", r.RequestURI).
			Int("status", wrapped.status).
			Int("size", wrapped.size).
			Dur("duration", time.Since(start)).
			Msg("HTTP request processed")
	})
}
