package config

import (
	"flag"
)

type Config struct {
	ServerAddress string // адрес HTTP-сервера
	BaseURL       string // базовый адрес для сокращенных URL
}

func NewConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080/", "base URL for shortened URLs")

	flag.Parse()

	return cfg
}
