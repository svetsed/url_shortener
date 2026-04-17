package postgres

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/svetsed/url_shortener/internal/model"
	"github.com/svetsed/url_shortener/internal/storage"
	"go.uber.org/zap"
)

var _ storage.DBRepository = (*postgresStorage)(nil)

type postgresStorage struct {
	db       *sql.DB
	sugarLog *zap.SugaredLogger
}

func NewPostgresStorage(dsn string, sugarLog *zap.SugaredLogger) (*postgresStorage, error) {
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

	storage := &postgresStorage{db: db, sugarLog: sugarLog}

	if err := storage.createDB(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("failed to create database: %w", err)
	}

	return storage, nil
}

func (ps *postgresStorage) createDB(ctx context.Context) error {
	query := `
		CREATE TABLE IF NOT EXISTS users (
			id SERIAL PRIMARY KEY,
			user_uuid UUID NOT NULL UNIQUE,
			created_at TIMESTAMP WITH TIME ZONE DEFAULT NOW()
		);

		CREATE TABLE IF NOT EXISTS urls (
			id SERIAL PRIMARY KEY,
			short_url VARCHAR(255) UNIQUE NOT NULL,
			original_url TEXT NOT NULL,
			user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE,
			is_deleted BOOLEAN DEFAULT FALSE
		);

		CREATE INDEX IF NOT EXISTS idx_urls_short ON urls(short_url);
		CREATE INDEX IF NOT EXISTS idx_urls_original ON urls(original_url);
		CREATE INDEX IF NOT EXISTS idx_urls_is_deleted ON urls(is_deleted);
		CREATE INDEX IF NOT EXISTS idx_urls_user_id ON urls(user_id);
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

	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		ps.sugarLog.Errorf("SaveURL in BeginTx: was error %w", err)
		return fmt.Errorf("failed to begin tx for save one url: %w", err)
	}
	defer tx.Rollback()

	queryUser := `
		INSERT INTO users (user_uuid) VALUES ($1) ON CONFLICT (user_uuid) DO NOTHING;
	`

	queryURLs := `
		INSERT INTO urls (short_url, original_url, user_id)
		VALUES ($1, $2, (SELECT id FROM users WHERE user_uuid = $3));
	`

	stmtUser, err := tx.PrepareContext(ctx, queryUser)
	if err != nil {
		ps.sugarLog.Errorf("SaveURL in PrepareContext-queryUser: was error %w", err)
		return fmt.Errorf("failed to prepare query for save url: %w", err)
	}
	defer stmtUser.Close()

	stmtURLs, err := tx.PrepareContext(ctx, queryURLs)
	if err != nil {
		ps.sugarLog.Errorf("SaveURL in PrepareContext-queryURLs: was error %w", err)
		return fmt.Errorf("failed to prepare query for save url: %w", err)
	}
	defer stmtURLs.Close()

	_, err = stmtUser.ExecContext(ctx, url.UserID)
	if err != nil {
		ps.sugarLog.Errorf("SaveURL in ExecContext-queryUser: was error %w", err)
		return fmt.Errorf("failed to save url: %w", err)
	}

	_, err = stmtURLs.ExecContext(ctx, url.ShortURL, url.OriginalURL, url.UserID)
	if err != nil {
		ps.sugarLog.Errorf("SaveURL in ExecContext-queryURLs: was error %w", err)
		return fmt.Errorf("failed to save url: %w", err)
	}

	return tx.Commit()
}

func (ps *postgresStorage) SaveManyURL(newURLs []*model.URL) error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	tx, err := ps.db.BeginTx(ctx, nil)
	if err != nil {
		ps.sugarLog.Errorf("SaveManyURL in BeginTx: was error %w", err)
		return fmt.Errorf("failed to begin tx for save many urls: %w", err)
	}
	defer tx.Rollback()

	queryUser := `
		INSERT INTO users (user_uuid) VALUES ($1) ON CONFLICT (user_uuid) DO NOTHING;
	`

	queryURLs := `
		INSERT INTO urls (short_url, original_url, user_id)
		VALUES ($1, $2, (SELECT id FROM users WHERE user_uuid = $3));
	`

	stmtUser, err := tx.PrepareContext(ctx, queryUser)
	if err != nil {
		ps.sugarLog.Errorf("SaveManyURL in PrepareContext-queryUser: was error %w", err)
		return fmt.Errorf("failed to prepare query for save many urls: %w", err)
	}
	defer stmtUser.Close()

	stmtURLs, err := tx.PrepareContext(ctx, queryURLs)
	if err != nil {
		ps.sugarLog.Errorf("SaveManyURL in PrepareContext-queryURLs: was error %w", err)
		return fmt.Errorf("failed to prepare query for save many urls: %w", err)
	}
	defer stmtURLs.Close()

	// can be diff users_id
	for _, url := range newURLs {
		_, err := stmtUser.ExecContext(ctx, url.UserID)
		if err != nil {
			ps.sugarLog.Errorf("SaveManyURL in ExecContext-queryUser: was error %w", err)
			return fmt.Errorf("failed to exec query for save many urls: %w", err)
		}

		_, err = stmtURLs.ExecContext(ctx, url.ShortURL, url.OriginalURL, url.UserID)
		if err != nil {
			ps.sugarLog.Errorf("SaveManyURL in ExecContext-queryURLs: was error %w", err)
			return fmt.Errorf("failed to exec query for save many urls: %w", err)
		}
	}

	return tx.Commit()
}

func (ps *postgresStorage) GetByShortURL(shortURL string) (*model.URL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT urls.id, short_url, original_url, user_uuid, is_deleted
		FROM urls
		JOIN users ON urls.user_id = users.id
		WHERE short_url = $1;
	`

	stmt, err := ps.db.PrepareContext(ctx, query)
	if err != nil {
		ps.sugarLog.Errorf("GetByShortURL in PrepareContext: was error %w; shortURL= %s", err, shortURL)
		return nil, fmt.Errorf("failed to prepare query for GetByShortURL: %w", err)
	}
	defer stmt.Close()

	url := &model.URL{}

	err = stmt.QueryRowContext(ctx, shortURL).Scan(
		&url.ID,
		&url.ShortURL,
		&url.OriginalURL,
		&url.UserID,
		&url.NeedDelete,
	)

	if err == sql.ErrNoRows {
		ps.sugarLog.Infof("GetByShortURL: was error storage.ErrorNotFound; shortURL= %s", shortURL)
		return nil, storage.ErrorNotFound
	}

	if err != nil {
		ps.sugarLog.Errorf("GetByShortURL in QueryRowContext: was error %w; shortURL= %s", err, shortURL)
		return nil, fmt.Errorf("failed to get by short url: %w", err)
	}

	return url, nil
}

func (ps *postgresStorage) GetByOringURL(origURL string) (*model.URL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// TODO проверить можно ли по условию задачи нескольким пользователям иметь одинаковые длинные ссылки(это логичнее)
	query := `
		SELECT urls.id, short_url, original_url, user_uuid, is_deleted
		FROM urls
		JOIN users ON urls.user_id = users.id
		WHERE original_url = $1;
	`

	stmt, err := ps.db.PrepareContext(ctx, query)
	if err != nil {
		ps.sugarLog.Errorf("GetByOringURL in PrepareContext: was error %w; origURL= %s", err, origURL)
		return nil, fmt.Errorf("failed to prepare query for GetByOringURL: %w", err)
	}
	defer stmt.Close()

	url := &model.URL{}
	err = stmt.QueryRowContext(ctx, origURL).Scan(
		&url.ID,
		&url.ShortURL,
		&url.OriginalURL,
		&url.UserID,
		&url.NeedDelete,
	)

	if err == sql.ErrNoRows {
		ps.sugarLog.Infof("GetByOringURL: was error storage.ErrorNotFound; origURL= %s", origURL)
		return nil, storage.ErrorNotFound
	}

	if err != nil {
		ps.sugarLog.Errorf("GetByOringURL in QueryRowContext: was error %w; origURL= %s", err, origURL)
		return nil, fmt.Errorf("failed to get by original url: %w", err)
	}

	return url, nil
}

func (ps *postgresStorage) GetUserURLs(userID string) ([]model.URL, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
		SELECT urls.id, short_url, original_url, user_uuid, is_deleted
		FROM urls
		JOIN users ON urls.user_id = users.id
		WHERE user_uuid=$1;
	`

	stmt, err := ps.db.PrepareContext(ctx, query)
	if err != nil {
		ps.sugarLog.Errorf("GetUserURLs in PrepareContext: was error %w; userID= %s", err, userID)
		return nil, fmt.Errorf("failed to prepare query for GetUserURLs: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, userID)
	if err != nil {
		ps.sugarLog.Errorf("GetUserURLs in QueryContext: was error %w; userID= %s", err, userID)
		return nil, fmt.Errorf("failed to get urls for user_id=%s: %w", userID, err)
	}
	defer rows.Close()

	userURLs := make([]model.URL, 0)
	for rows.Next() {
		var url model.URL

		err := rows.Scan(
			&url.ID,
			&url.ShortURL,
			&url.OriginalURL,
			&url.UserID,
			&url.NeedDelete,
		)
		if err != nil {
			ps.sugarLog.Errorf("GetUserURLs in Scan: was error %w; userID= %s", err, userID)
			return nil, fmt.Errorf("scan failed for user_id=%s: %w", userID, err)
		}

		userURLs = append(userURLs, url)
	}

	if err = rows.Err(); err != nil {
		ps.sugarLog.Errorf("GetUserURLs in Rows iteration: was error %w; userID= %s", err, userID)
		return nil, fmt.Errorf("rows iteration error: %w", err)
	}

	return userURLs, nil
}

func (ps *postgresStorage) MarkAsDeleted(shortURLs []string, userID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	query := `
        UPDATE urls 
        SET is_deleted = true
		FROM users 
		WHERE urls.user_id = users.id 
  		AND users.user_uuid = $1 
  		AND urls.short_url = ANY($2) 
  		AND urls.is_deleted = false;
    `

	stmt, err := ps.db.PrepareContext(ctx, query)
	if err != nil {
		ps.sugarLog.Errorf("MarkAsDeleted in PrepareContext: was error %w; userID= %s", err, userID)
		return fmt.Errorf("failed to prepare query in MarkAsDeleted: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, userID, shortURLs)
	if err != nil {
		ps.sugarLog.Errorf("MarkAsDeleted in ExecContext: was error %w; userID= %s", err, userID)
		return fmt.Errorf("failed to exec query in MarkAsDeleted: %w", err)
	}

	rowsAff, _ := result.RowsAffected()
	if rowsAff == 0 {
		ps.sugarLog.Infof("MarkAsDeleted was error storage.ErrNothingToDelete; userID= %s", userID)
		return storage.ErrNothingToDelete
	}

	return nil
}

// ----------------   Implement Closer   ----------------

func (ps *postgresStorage) Close() error {
	if ps.db != nil {
		return ps.db.Close()
	}

	return nil
}
