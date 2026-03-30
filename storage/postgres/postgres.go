package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/storage"
	"go.uber.org/zap"
)

var _ storage.DBRepository = (*postgresStorage)(nil)

type postgresStorage struct {
	db *sql.DB
	sugarLog *zap.SugaredLogger
}

func NewPostgresStorage(dsn string) (*postgresStorage, error) {
	if dsn == "" {
        return nil, fmt.Errorf("database DSN is empty")
    }
	
	db, err := sql.Open("pgx", dsn)
	if err != nil {
		return nil, fmt.Errorf("failed to open database: %w", err)
	}

	// // Настройки пула соединений
    // db.SetMaxOpenConns(25)
    // db.SetMaxIdleConns(5)
    // db.SetConnMaxLifetime(5 * time.Minute)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, err
	}

	storage := &postgresStorage{db: db}

	if err := storage.createDB(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	return storage, nil
}

func (ps *postgresStorage) createDB(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS urls (
			id SERIAL PRIMARY KEY,
			short_url VARCHAR(255) UNIQUE NOT NULL,
			original_url TEXT NOT NULL
		);

		CREATE INDEX IF NOT EXISTS idx_irls_short ON urls(short_url);
		CREATE INDEX IF NOT EXISTS idx_urls_original ON urls(original_url);
	`
	// created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP

	_, err := ps.db.ExecContext(ctx, query)
	return err
}


// ----------------   Implement Pinger   ----------------

func (ps *postgresStorage) Ping(ctx context.Context) error {
	if ps.db == nil {
		return storage.ErrorStorageNotInitialized
	}

	return ps.db.PingContext(ctx)
}


// ---------------- Implement Repository ----------------

func (ps *postgresStorage) Save(url *model.URL) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		INSERT INTO urls (short_url, original_url)
		VALUES ($1, $2)
	`

	_, err := ps.db.ExecContext(ctx, query, url.ShortURL, url.OriginalURL)

	if err != nil {
		return fmt.Errorf("failed to save url: %w", err)
	}

    return nil
}

func (ps *postgresStorage) GetByShortURL(shortURL string) (*model.URL, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT id, short_url, original_url
		FROM urls
		WHERE short_url = $1
	`

	url := &model.URL{}

	err := ps.db.QueryRowContext(ctx, query, shortURL).Scan(
		&url.ID,
		&url.ShortURL,
		&url.OriginalURL,
	)

	if err == sql.ErrNoRows {
		return nil, storage.ErrorNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get by short url: %w", err)
	}

    return url, nil
}

func (ps *postgresStorage) GetByOringURL(origURL string) (*model.URL, error) {
    ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second) 
	defer cancel()

	query := `
		SELECT id, short_url, original_url
		FROM urls 
		WHERE original_url = $1
	`
	url := &model.URL{}
	err := ps.db.QueryRowContext(ctx, query, origURL).Scan(
		&url.ID,
		&url.ShortURL,
		&url.OriginalURL,
	)

	if  err == sql.ErrNoRows {
		return nil, storage.ErrorNotFound
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get by original url: %w", err)
	}

    return url, nil
}


// ----------------   Implement Closer   ----------------

func (ps *postgresStorage) Close() error {
	if ps.db != nil {
		return ps.db.Close()
	}

	return nil
}