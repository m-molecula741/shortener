package middleware

import (
	"compress/gzip"
	"net/http"
	"strings"
)

// compressibleTypes - типы контента, которые нужно сжимать
var compressibleTypes = map[string]bool{
	"application/json": true,
	"text/html":        true,
}

func shouldCompress(headers http.Header) bool {
	contentType := headers.Get("Content-Type")
	for typ := range compressibleTypes {
		if strings.Contains(contentType, typ) {
			return true
		}
	}
	return false
}

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

		// 2. Проверяем, поддерживает ли клиент gzip
		acceptEncoding := r.Header.Get("Accept-Encoding")
		clientSupportsGzip := strings.Contains(acceptEncoding, "gzip")

		// Если клиент не поддерживает gzip, просто передаем дальше
		if !clientSupportsGzip {
			next.ServeHTTP(w, r)
			return
		}

		// 3. Создаем обертку для перехвата ответа
		writer := &gzipResponseWriter{
			ResponseWriter: w,
			contentType:    "",
			shouldCompress: false,
		}

		next.ServeHTTP(writer, r)

		// 4. Если нужно сжимать и есть данные для сжатия
		if writer.shouldCompress && len(writer.data) > 0 {
			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Del("Content-Length")

			gz := gzip.NewWriter(w)
			defer gz.Close()

			if _, err := gz.Write(writer.data); err != nil {
				http.Error(w, "Compression failed", http.StatusInternalServerError)
				return
			}
		} else {
			// Возвращаем оригинальный ответ если сжатие не требуется
			w.Write(writer.data)
		}
	})
}

// gzipResponseWriter перехватывает ответ для анализа
type gzipResponseWriter struct {
	http.ResponseWriter
	data          []byte
	contentType   string
	shouldCompress bool
	wroteHeaders   bool
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeaders {
		w.WriteHeader(http.StatusOK)
	}
	w.data = append(w.data, b...)
	return len(b), nil
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if !w.wroteHeaders {
		w.wroteHeaders = true
		w.contentType = w.Header().Get("Content-Type")

		// Проверяем, нужно ли сжимать ответ
		w.shouldCompress = shouldCompress(w.Header())

		// Отключаем сжатие для специальных статусов
		if statusCode == http.StatusNoContent || 
		   statusCode == http.StatusNotModified ||
		   (statusCode >= 300 && statusCode < 400) {
			w.shouldCompress = false
		}

		w.ResponseWriter.WriteHeader(statusCode)
	}
}