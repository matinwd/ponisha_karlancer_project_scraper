package db

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/jackc/pgx/v5/pgxpool"
)

func NewPool(ctx context.Context, dsn string) (*pgxpool.Pool, error) {
	pool, err := pgxpool.New(ctx, dsn)
	if err != nil {
		return nil, err
	}
	return pool, nil
}

func EnsureSchema(ctx context.Context, pool *pgxpool.Pool, basePath string) error {
	schemaPath := filepath.Join(basePath, "db", "schema.sql")
	content, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	_, err = pool.Exec(ctx, string(content))
	if err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}
