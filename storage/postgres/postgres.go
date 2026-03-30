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
		return nil, fmt.Errorf("failed to open dataqbase: %w", err)
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

	return &postgresStorage{db: db}, nil
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
    // TODO: реализовать позже
    return nil
}

func (ps *postgresStorage) GetByShortURL(shortURL string) (*model.URL, error) {
    // TODO: реализовать позже
    return nil, storage.ErrorNotFound
}

func (ps *postgresStorage) GetByOringURL(origURL string) (*model.URL, error) {
    // TODO: реализовать позже  
    return nil, storage.ErrorNotFound
}



// ----------------   Implement Closer   ----------------

func (ps *postgresStorage) Close() error {
	if ps.db != nil {
		return ps.db.Close()
	}

	return nil
}