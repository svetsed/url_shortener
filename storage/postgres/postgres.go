package postgres

// import (
// 	"database/sql"
// 	"fmt"

// 	_ "github.com/jackc/pgx/v5/stdlib"
// )

// type Storage struct {
// 	conn *sql.Conn
// }

// func New(storagePath string) (*Storage, error) {
// 	dsn := "postgres://user:password@localhost:5432/mydb?sslmode=disable"

// 	db, err := sql.Open("pgx", dsn)
// 	if err != nil {
// 		return nil, fmt.Errorf("Failed to load driver: %w", err)
// 	}

// 	defer db.Close()

// 	if err = db.Ping(); err != nil {
// 		return nil, fmt.Errorf("Failed to ping database: %w", err)
// 	}

	
// }