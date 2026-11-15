package sqlite

import (
	"database/sql"
	"embed"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/pressly/goose/v3"
	_ "modernc.org/sqlite"
)

//go:embed migrations/*.sql
var migrationFS embed.FS

type DB struct {
	conn *sql.DB
}

// New initializes the SQLite database and applies migrations.
func New(dataDir string) (*DB, error) {

	if err := os.MkdirAll(dataDir, 0700); err != nil {
		return nil, fmt.Errorf(
			"create data directory: %w", err,
		)
	}

	dbPath := filepath.Join(dataDir, "termai.db")

	dsn := fmt.Sprintf(
		"file:%s?_pragma=foreign_keys(1)&_pragma=busy_timeout(5000)",
		filepath.ToSlash(dbPath),
	)

	conn, err := sql.Open("sqlite", dsn)
	if err != nil {
		return nil, fmt.Errorf(
			"open database: %w", err,
		)
	}

	// SQLite is embedded and local. One connection avoids
	// inconsistent per-connection foreign-key settings.
	conn.SetMaxOpenConns(1)
	conn.SetMaxIdleConns(1)

	if err := conn.Ping(); err != nil {
		conn.Close()
		return nil, fmt.Errorf(
			"connect to database: %w", err,
		)
	}

	if err := runMigrations(conn); err != nil {
		conn.Close()
		return nil, fmt.Errorf(
			"run database migrations: %w", err,
		)
	}

	return &DB{conn: conn}, nil
}

func runMigrations(conn *sql.DB) error {
	goose.SetLogger(log.New(io.Discard, "", 0))

	goose.SetBaseFS(migrationFS)
	defer goose.SetBaseFS(nil)

	if err := goose.SetDialect("sqlite3"); err != nil {
		return err
	}

	if err := goose.Up(conn, "migrations"); err != nil {
		return err
	}

	return nil
}

// Close closes the underlying database connection.
func (db *DB) Close() error {
	return db.conn.Close()
}
