// Package middleware предоставляет middleware компоненты для HTTP сервера
package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// compressibleTypes содержит MIME-типы, для которых включается сжатие
var compressibleTypes = map[string]bool{
	"application/json": true,
	"text/html":        true,
}

// GzipMiddleware обеспечивает сжатие ответов с помощью gzip
func GzipMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// 1. Обработка входящего gzip
		if strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			gz, err := gzip.NewReader(r.Body)
			if err != nil {
				http.Error(w, "Invalid gzip body", http.StatusBadRequest)
				return
			}
			defer gz.Close()
			r.Body = gz
		}

		// 2. Проверяем поддержку gzip клиентом
		acceptsGzip := strings.Contains(r.Header.Get("Accept-Encoding"), "gzip")
		if !acceptsGzip {
			next.ServeHTTP(w, r)
			return
		}

		// 3. Используем перехватчик с копированием заголовков
		writer := &gzipResponseWriter{
			ResponseWriter: w,
			acceptsGzip:    acceptsGzip,
		}
		defer writer.Close()

		next.ServeHTTP(writer, r)
	})
}

// gzipResponseWriter реализует интерфейс http.ResponseWriter для сжатия ответов
type gzipResponseWriter struct {
	http.ResponseWriter
	gz          *gzip.Writer
	headers     http.Header
	wroteHeader bool
	acceptsGzip bool
}

// Write реализует интерфейс io.Writer для сжатия данных
func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	contentType := w.headers.Get("Content-Type")
	shouldCompress := w.acceptsGzip && shouldCompressContentType(contentType)

	if shouldCompress {
		return w.gz.Write(b)
	}
	return w.ResponseWriter.Write(b)
}

// WriteHeader устанавливает статус код и необходимые заголовки для сжатия
func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true

	// Копируем заголовки
	w.headers = w.Header().Clone()

	contentType := w.headers.Get("Content-Type")
	shouldCompress := w.acceptsGzip && shouldCompressContentType(contentType) &&
		statusCode != http.StatusNoContent &&
		statusCode != http.StatusNotModified &&
		!(statusCode >= 300 && statusCode < 400)

	if shouldCompress {
		w.headers.Set("Content-Encoding", "gzip")
		w.headers.Del("Content-Length")
		w.gz = gzip.NewWriter(w.ResponseWriter)
	}

	// Применяем заголовки
	for k, v := range w.headers {
		w.ResponseWriter.Header()[k] = v
	}

	w.ResponseWriter.WriteHeader(statusCode)
}

// Close закрывает gzip.Writer если он был создан
func (w *gzipResponseWriter) Close() {
	if w.gz != nil {
		w.gz.Close()
	}
}

// shouldCompressContentType проверяет, нужно ли сжимать контент данного типа
func shouldCompressContentType(contentType string) bool {
	for typ := range compressibleTypes {
		if strings.Contains(contentType, typ) {
			return true
		}
	}
	return false
}
