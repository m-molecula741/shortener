package storage

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresDB структура для работы с PostgreSQL
type PostgresDB struct {
	pool *pgxpool.Pool
}

// NewPostgresDB создает новое подключение к PostgreSQL
func NewPostgresDB(dsn string) (*PostgresDB, error) {
	pool, err := pgxpool.New(context.Background(), dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to create connection pool: %w", err)
	}

	return &PostgresDB{
		pool: pool,
	}, nil
}

// Ping проверяет соединение с базой данных
func (db *PostgresDB) Ping() error {
	return db.pool.Ping(context.Background())
}

// Close закрывает соединение с базой данных
func (db *PostgresDB) Close() {
	db.pool.Close()
}
