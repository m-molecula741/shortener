// Package middleware предоставляет middleware компоненты для HTTP сервера
package middleware

import (
	"net/http"
	"time"

	"github.com/m-molecula741/shortener/internal/app/logger"
)

// responseWriter реализует интерфейс http.ResponseWriter для сбора метрик
type responseWriter struct {
	http.ResponseWriter
	status int
	size   int
}

// WriteHeader устанавливает статус код ответа
func (rw *responseWriter) WriteHeader(status int) {
	rw.status = status
	rw.ResponseWriter.WriteHeader(status)
}

// Write записывает данные и подсчитывает их размер
func (rw *responseWriter) Write(b []byte) (int, error) {
	size, err := rw.ResponseWriter.Write(b)
	rw.size += size
	return size, err
}

// RequestLogger middleware для логирования HTTP запросов
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
