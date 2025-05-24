package logger

import (
	"os"
	"time"

	"github.com/rs/zerolog"
)

var log zerolog.Logger

func Init() {
	output := zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	log = zerolog.New(output).With().Timestamp().Logger()
}

func Info() *zerolog.Event {
	return log.Info()
}

func GetLogger() *zerolog.Logger {
	return &log
}
