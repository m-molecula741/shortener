// Package config предоставляет конфигурацию для приложения
package config

import (
	"flag"
	"os"
	"strconv"
)

// Константы для конфигурации
const (
	defaultStorageFile = "urls.json"
)

// Config представляет конфигурацию приложения
type Config struct {
	ServerAddress   string // адрес HTTP-сервера
	BaseURL         string // базовый адрес для сокращенных URL
	StorageFilePath string // путь к файлу для хранения URL
	DatabaseDSN     string // строка подключения к базе данных
	EnablePprof     bool   // включить профилирование pprof
	EnableHTTPS     bool   // включить HTTPS
	CertFile        string // путь к файлу сертификата
	KeyFile         string // путь к файлу ключа
}

// NewConfig создает новую конфигурацию
func NewConfig() *Config {
	cfg := &Config{}

	flag.StringVar(&cfg.ServerAddress, "a", "localhost:8080", "HTTP server address")
	flag.StringVar(&cfg.BaseURL, "b", "http://localhost:8080/", "base URL for shortened URLs")
	flag.StringVar(&cfg.StorageFilePath, "f", defaultStorageFile, "file storage path")
	flag.StringVar(&cfg.DatabaseDSN, "d", "", "database connection string")
	flag.BoolVar(&cfg.EnablePprof, "pprof", false, "enable pprof profiling")
	flag.BoolVar(&cfg.EnableHTTPS, "s", false, "enable HTTPS")
	flag.StringVar(&cfg.CertFile, "cert", "server.crt", "path to certificate file")
	flag.StringVar(&cfg.KeyFile, "key", "server.key", "path to key file")

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

	if envPprof := os.Getenv("ENABLE_PPROF"); envPprof != "" {
		if enabled, err := strconv.ParseBool(envPprof); err == nil {
			cfg.EnablePprof = enabled
		}
	}

	if envHTTPS := os.Getenv("ENABLE_HTTPS"); envHTTPS != "" {
		if enabled, err := strconv.ParseBool(envHTTPS); err == nil {
			cfg.EnableHTTPS = enabled
		}
	}

	if envCertFile := os.Getenv("CERT_FILE"); envCertFile != "" {
		cfg.CertFile = envCertFile
	}

	if envKeyFile := os.Getenv("KEY_FILE"); envKeyFile != "" {
		cfg.KeyFile = envKeyFile
	}

	return cfg
}
