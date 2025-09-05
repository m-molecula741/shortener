// Package config предоставляет конфигурацию для приложения
package config

import (
	"encoding/json"
	"flag"
	"os"
	"strconv"
)

// Константы для конфигурации
const (
	defaultStorageFile = "urls.json"
)

// JSONConfig представляет структуру JSON файла конфигурации
type JSONConfig struct {
	ServerAddress   string `json:"server_address"`
	BaseURL         string `json:"base_url"`
	FileStoragePath string `json:"file_storage_path"`
	DatabaseDSN     string `json:"database_dsn"`
	EnableHTTPS     bool   `json:"enable_https"`
}

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
	ConfigFile      string // путь к файлу конфигурации JSON
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
	flag.StringVar(&cfg.ConfigFile, "c", "", "path to JSON config file")
	flag.StringVar(&cfg.ConfigFile, "config", "", "path to JSON config file")

	flag.Parse()

	// Проверяем переменную окружения CONFIG
	if envConfigFile := os.Getenv("CONFIG"); envConfigFile != "" {
		cfg.ConfigFile = envConfigFile
	}

	// Загружаем конфигурацию из JSON файла, если указан
	if cfg.ConfigFile != "" {
		if err := cfg.loadFromJSON(); err != nil {
			// Если не удалось загрузить JSON, продолжаем с текущими значениями
			// (не прерываем работу, как это обычно делается с конфигурацией)
		}
	}

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

// loadFromJSON загружает конфигурацию из JSON файла
// Применяет значения только для тех полей, которые не были установлены флагами или переменными окружения
func (cfg *Config) loadFromJSON() error {
	data, err := os.ReadFile(cfg.ConfigFile)
	if err != nil {
		return err
	}

	var jsonCfg JSONConfig
	if err := json.Unmarshal(data, &jsonCfg); err != nil {
		return err
	}

	// Применяем значения из JSON только если они не установлены
	// (приоритет: флаги > env > JSON > defaults)

	// Для строковых полей проверяем, что они равны значению по умолчанию
	if cfg.ServerAddress == "localhost:8080" && jsonCfg.ServerAddress != "" {
		cfg.ServerAddress = jsonCfg.ServerAddress
	}

	if cfg.BaseURL == "http://localhost:8080/" && jsonCfg.BaseURL != "" {
		cfg.BaseURL = jsonCfg.BaseURL
	}

	if cfg.StorageFilePath == defaultStorageFile && jsonCfg.FileStoragePath != "" {
		cfg.StorageFilePath = jsonCfg.FileStoragePath
	}

	if cfg.DatabaseDSN == "" && jsonCfg.DatabaseDSN != "" {
		cfg.DatabaseDSN = jsonCfg.DatabaseDSN
	}

	// Для булевых полей применяем значение из JSON только если оно true и текущее значение false
	if !cfg.EnableHTTPS && jsonCfg.EnableHTTPS {
		cfg.EnableHTTPS = jsonCfg.EnableHTTPS
	}

	return nil
}
