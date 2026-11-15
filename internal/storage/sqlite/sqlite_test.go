package sqlite

import (
	"context"
	"testing"
)

func TestNewAppliesMigrations(t *testing.T) {
	db, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("failed to initialize database: %v", err)
	}
	defer db.Close()

	var count int

	err = db.conn.QueryRowContext(
		context.Background(),
		`SELECT COUNT(*)
         FROM sqlite_master
         WHERE type = 'table'
         AND name IN ('sessions', 'messages')`,
	).Scan(&count)

	if err != nil {
		t.Fatalf("failed to query tables: %v", err)
	}

	if count != 2 {
		t.Fatalf("expected 2 tables, got %d", count)
	}
}

func TestForeignKeyCascade(t *testing.T) {
	db, err := New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()

	ctx := context.Background()

	_, err = db.conn.ExecContext(
		ctx,
		`INSERT INTO sessions
         (id, title, provider, model)
         VALUES ('test-session', 'Test', 'ollama', 'llama3.2:3b')`,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.conn.ExecContext(
		ctx,
		`INSERT INTO messages
         (session_id, role, content)
         VALUES ('test-session', 'user', 'Hello')`,
	)
	if err != nil {
		t.Fatal(err)
	}

	_, err = db.conn.ExecContext(
		ctx,
		`DELETE FROM sessions WHERE id = ?`,
		"test-session",
	)
	if err != nil {
		t.Fatal(err)
	}

	var count int

	err = db.conn.QueryRowContext(
		ctx,
		`SELECT COUNT(*) FROM messages`,
	).Scan(&count)

	if err != nil {
		t.Fatal(err)
	}

	if count != 0 {
		t.Fatalf("expected cascade deletion, got %d messages", count)
	}
}
