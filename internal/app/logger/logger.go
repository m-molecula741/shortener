// Package logger предоставляет функционал для логирования в сервисе
package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

// log глобальный логгер приложения
var log zerolog.Logger

// Init инициализирует глобальный логгер с настроенным форматом времени
func Init() {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log = zerolog.New(output).With().Timestamp().Logger()
}

// Info возвращает Event для логирования информационных сообщений
func Info() *zerolog.Event {
	return log.Info()
}

// GetLogger возвращает указатель на глобальный логгер
func GetLogger() *zerolog.Logger {
	return &log
}
