package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type PostgresStorage struct {
	pool *pgxpool.Pool
}

func NewPostgresStorage(dsn string) (*PostgresStorage, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	storage := &PostgresStorage{
		pool: pool,
	}

	// Создаем таблицу при инициализации
	if err := storage.createTable(); err != nil {
		pool.Close()
		return nil, fmt.Errorf("failed to create table: %w", err)
	}

	return storage, nil
}

func (s *PostgresStorage) createTable() error {
	query := `
		CREATE TABLE IF NOT EXISTS urls (
			short_id VARCHAR(8) PRIMARY KEY,
			original_url TEXT NOT NULL,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
	`
	_, err := s.pool.Exec(context.Background(), query)
	return err
}

// Save сохраняет URL в PostgreSQL (реализация URLStorage)
func (s *PostgresStorage) Save(shortID, url string) error {
	query := `
		INSERT INTO urls (short_id, original_url) 
		VALUES ($1, $2) 
		ON CONFLICT (short_id) 
		DO UPDATE SET original_url = EXCLUDED.original_url
	`
	_, err := s.pool.Exec(context.Background(), query, shortID, url)
	return err
}

// Get получает оригинальный URL по короткому ID (реализация URLStorage)
func (s *PostgresStorage) Get(shortID string) (string, error) {
	var originalURL string
	query := `SELECT original_url FROM urls WHERE short_id = $1`

	err := s.pool.QueryRow(context.Background(), query, shortID).Scan(&originalURL)
	if err != nil {
		return "", fmt.Errorf("URL not found: %w", err)
	}

	return originalURL, nil
}

// Ping проверяет соединение с базой данных (реализация DatabasePinger)
func (s *PostgresStorage) Ping() error {
	return s.pool.Ping(context.Background())
}

// Close закрывает соединение с базой данных (реализация DatabasePinger)
func (s *PostgresStorage) Close() {
	s.pool.Close()
}
