package storage

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/m-molecula741/shortener/internal/app/usecase"
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
			user_id VARCHAR(36),
			is_deleted BOOLEAN DEFAULT FALSE,
			created_at TIMESTAMP DEFAULT CURRENT_TIMESTAMP
		);
		CREATE UNIQUE INDEX IF NOT EXISTS idx_urls_original_url ON urls(original_url);
	`
	_, err := s.pool.Exec(context.Background(), query)
	return err
}

// Save сохраняет URL в PostgreSQL
func (s *PostgresStorage) Save(shortID, url string) error {
	query := `
		INSERT INTO urls (short_id, original_url) 
		VALUES ($1, $2)
	`
	_, err := s.pool.Exec(context.Background(), query, shortID, url)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			// Если нарушение уникальности по original_url, находим существующий short_id
			if pgErr.ConstraintName == "idx_urls_original_url" {
				var existingShortID string
				selectQuery := `SELECT short_id FROM urls WHERE original_url = $1`
				err := s.pool.QueryRow(context.Background(), selectQuery, url).Scan(&existingShortID)
				if err != nil {
					return fmt.Errorf("failed to get existing short_id: %w", err)
				}
				return &usecase.ErrURLConflict{ExistingShortURL: existingShortID}
			}
		}
		return err
	}
	return nil
}

// Get получает оригинальный URL по короткому ID
func (s *PostgresStorage) Get(shortID string) (string, error) {
	var originalURL string
	var isDeleted bool
	query := `SELECT original_url, is_deleted FROM urls WHERE short_id = $1`

	err := s.pool.QueryRow(context.Background(), query, shortID).Scan(&originalURL, &isDeleted)
	if err != nil {
		return "", fmt.Errorf("URL not found: %w", err)
	}

	// Если URL помечен как удаленный, возвращаем специальную ошибку
	if isDeleted {
		return "", &usecase.ErrURLDeleted{}
	}

	return originalURL, nil
}

// Ping проверяет соединение с базой данных
func (s *PostgresStorage) Ping() error {
	return s.pool.Ping(context.Background())
}

// Close закрывает соединение с базой данных
func (s *PostgresStorage) Close() error {
	s.pool.Close()
	return nil
}

// SaveBatch сохраняет множество URL за одну операцию в рамках транзакции
func (s *PostgresStorage) SaveBatch(ctx context.Context, urls []usecase.URLPair) error {
	// Начинаем транзакцию
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx) // Откатываем транзакцию в случае ошибки

	query := `
		INSERT INTO urls (short_id, original_url, user_id) 
		VALUES ($1, $2, $3) 
		ON CONFLICT (short_id) DO UPDATE SET user_id = EXCLUDED.user_id WHERE urls.user_id IS NULL
	`

	// Выполняем все вставки в рамках одной транзакции
	for _, url := range urls {
		_, err := tx.Exec(ctx, query, url.ShortID, url.OriginalURL, url.UserID)
		if err != nil {
			return fmt.Errorf("failed to save URL %s: %w", url.ShortID, err)
		}
	}

	// Коммитим транзакцию
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// GetUserURLs получает все URL пользователя
func (s *PostgresStorage) GetUserURLs(ctx context.Context, userID string) ([]usecase.UserURL, error) {
	query := `SELECT short_id, original_url FROM urls WHERE user_id = $1 AND is_deleted = FALSE`

	rows, err := s.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, fmt.Errorf("failed to query user URLs: %w", err)
	}
	defer rows.Close()

	var urls []usecase.UserURL
	for rows.Next() {
		var shortID, originalURL string
		if err := rows.Scan(&shortID, &originalURL); err != nil {
			return nil, fmt.Errorf("failed to scan row: %w", err)
		}

		urls = append(urls, usecase.UserURL{
			ShortURL:    fmt.Sprintf("http://localhost:8080/%s", shortID),
			OriginalURL: originalURL,
		})
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return urls, nil
}

// BatchDeleteUserURLs помечает URL пользователя как удаленные
func (s *PostgresStorage) BatchDeleteUserURLs(ctx context.Context, userID string, shortIDs []string) error {
	if len(shortIDs) == 0 {
		return nil
	}

	// Начинаем транзакцию
	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Обновляем is_deleted для указанных URL только если они принадлежат пользователю
	query := `
		UPDATE urls 
		SET is_deleted = TRUE 
		WHERE user_id = $1 AND short_id = ANY($2)
	`

	_, err = tx.Exec(ctx, query, userID, shortIDs)
	if err != nil {
		return fmt.Errorf("failed to mark URLs as deleted: %w", err)
	}

	// Коммитим транзакцию
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
