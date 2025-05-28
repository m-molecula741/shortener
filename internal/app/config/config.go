package config

import (
	"flag"
	"os"
)

const (
	defaultStorageFile = "urls.json"
)

type Config struct {
	ServerAddress   string // адрес HTTP-сервера
	BaseURL         string // базовый адрес для сокращенных URL
	StorageFilePath string // путь к файлу для хранения URL
	DatabaseDSN     string // строка подключения к базе данных
}

func NewConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080/", "base URL for shortened URLs")
	flag.StringVar(&cfg.StorageFilePath, "f", defaultStorageFile, "file storage path")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database connection string")

	flag.Parse()

	if envServerAddr := os.Getenv("SERVER_ADDRESS"); envServerAddr != "" {
		cfg.ServerAddress = envServerAddr
	}

	if envBaseURL := os.Getenv("BASE_URL"); envBaseURL != "" {
		cfg.BaseURL = envBaseURL
	}

	if envStoragePath := os.Getenv("FILE_STORAGE_PATH"); envStoragePath != "" {
		cfg.StorageFilePath = envStoragePath
	}

	if envDatabaseDSN := os.Getenv("DATABASE_DSN"); envDatabaseDSN != "" {
		cfg.DatabaseDSN = envDatabaseDSN
	}

	return cfg
}
