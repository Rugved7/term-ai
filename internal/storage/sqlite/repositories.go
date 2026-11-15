package sqlite

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"time"

	"github.com/Rugved7/termai/internal/conversation"
	"github.com/Rugved7/termai/internal/storage"
)

var _ storage.Store = (*DB)(nil)

// CreateSession inserts a new conversation session.
func (db *DB) CreateSession(ctx context.Context, session *conversation.Session) error {
	if session == nil {
		return errors.New("create session: session cannot be nil")
	}

	now := time.Now().UTC()
	session.CreatedAt = now
	session.UpdatedAt = now

	_, err := db.conn.ExecContext(
		ctx,
		`INSERT INTO sessions (id, title, provider, model, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		session.ID,
		session.Title,
		session.Provider,
		session.Model,
		session.CreatedAt,
		session.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("create session: %w", err)
	}

	return nil
}

// GetByID retrieves a session by its ID.
func (db *DB) GetByID(ctx context.Context, id string) (*conversation.Session, error) {
	var session conversation.Session

	err := db.conn.QueryRowContext(
		ctx,
		`SELECT id, title, provider, model, created_at, updated_at
		 FROM sessions
		 WHERE id = ?`,
		id,
	).Scan(
		&session.ID,
		&session.Title,
		&session.Provider,
		&session.Model,
		&session.CreatedAt,
		&session.UpdatedAt,
	)

	if errors.Is(err, sql.ErrNoRows) {
		return nil, storage.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get session: %w", err)
	}

	return &session, nil
}

// List retrieves the most recently updated sessions.
func (db *DB) List(ctx context.Context, limit int) ([]conversation.Session, error) {
	if limit <= 0 {
		return nil, errors.New("list sessions: limit must be positive")
	}

	rows, err := db.conn.QueryContext(
		ctx,
		`SELECT id, title, provider, model, created_at, updated_at
		 FROM sessions
		 ORDER BY updated_at DESC, id DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list sessions: %w", err)
	}
	defer rows.Close()

	sessions := make([]conversation.Session, 0)

	for rows.Next() {
		var session conversation.Session

		if err := rows.Scan(
			&session.ID,
			&session.Title,
			&session.Provider,
			&session.Model,
			&session.CreatedAt,
			&session.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan session: %w", err)
		}

		sessions = append(sessions, session)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate sessions: %w", err)
	}

	return sessions, nil
}

// Update modifies a session's metadata and update timestamp.
func (db *DB) Update(ctx context.Context, session *conversation.Session) error {
	if session == nil {
		return errors.New("update session: session cannot be nil")
	}

	now := time.Now().UTC()

	result, err := db.conn.ExecContext(
		ctx,
		`UPDATE sessions
		 SET title = ?, provider = ?, model = ?, updated_at = ?
		 WHERE id = ?`,
		session.Title,
		session.Provider,
		session.Model,
		now,
		session.ID,
	)
	if err != nil {
		return fmt.Errorf("update session: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("update session: check affected rows: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}

	session.UpdatedAt = now
	return nil
}

// Delete removes a session. Its messages are deleted by the foreign-key cascade.
func (db *DB) Delete(ctx context.Context, id string) error {
	result, err := db.conn.ExecContext(
		ctx,
		`DELETE FROM sessions WHERE id = ?`,
		id,
	)
	if err != nil {
		return fmt.Errorf("delete session: %w", err)
	}

	affected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete session: check affected rows: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}

	return nil
}

// CreateMessage persists a message and populates its generated ID and timestamp.
func (db *DB) CreateMessage(ctx context.Context, message *conversation.Message) error {
	if message == nil {
		return errors.New("create message: message cannot be nil")
	}

	now := time.Now().UTC()
	message.CreatedAt = now

	result, err := db.conn.ExecContext(
		ctx,
		`INSERT INTO messages (session_id, role, content, created_at)
		 VALUES (?, ?, ?, ?)`,
		message.SessionID,
		message.Role,
		message.Content,
		message.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("create message: %w", err)
	}

	message.ID, err = result.LastInsertId()
	if err != nil {
		return fmt.Errorf("create message: retrieve generated ID: %w", err)
	}

	return nil
}

// ListBySession retrieves a session's messages in conversation order.
func (db *DB) ListBySession(ctx context.Context, sessionID string) ([]conversation.Message, error) {
	rows, err := db.conn.QueryContext(
		ctx,
		`SELECT id, session_id, role, content, created_at
		 FROM messages
		 WHERE session_id = ?
		 ORDER BY id ASC`,
		sessionID,
	)
	if err != nil {
		return nil, fmt.Errorf("list messages: %w", err)
	}
	defer rows.Close()

	messages := make([]conversation.Message, 0)

	for rows.Next() {
		var message conversation.Message

		if err := rows.Scan(
			&message.ID,
			&message.SessionID,
			&message.Role,
			&message.Content,
			&message.CreatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan message: %w", err)
		}

		messages = append(messages, message)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate messages: %w", err)
	}

	return messages, nil
}
