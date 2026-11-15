package sqlite

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/Rugved7/termai/internal/conversation"
	"github.com/Rugved7/termai/internal/storage"
)

// SaveExchange atomically persists a user message, an assistant response,
// and the updated session metadata.
func (db *DB) SaveExchange(
	ctx context.Context,
	session *conversation.Session,
	userMessage *conversation.Message,
	assistantMessage *conversation.Message,
) error {
	if session == nil {
		return errors.New("save exchange: session cannot be nil")
	}
	if userMessage == nil || assistantMessage == nil {
		return errors.New("save exchange: messages cannot be nil")
	}
	if userMessage.SessionID != session.ID ||
		assistantMessage.SessionID != session.ID {
		return errors.New("save exchange: session IDs do not match")
	}
	if userMessage.Role != "user" || assistantMessage.Role != "assistant" {
		return errors.New("save exchange: expected user and assistant messages")
	}

	tx, err := db.conn.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin exchange transaction: %w", err)
	}
	defer tx.Rollback()

	now := time.Now().UTC()

	// Insert the user message.
	userResult, err := tx.ExecContext(
		ctx,
		`INSERT INTO messages (session_id, role, content, created_at)
		 VALUES (?, ?, ?, ?)`,
		session.ID,
		userMessage.Role,
		userMessage.Content,
		now,
	)
	if err != nil {
		return fmt.Errorf("save user message: %w", err)
	}

	userID, err := userResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("retrieve user message ID: %w", err)
	}

	// Insert the assistant response.
	assistantResult, err := tx.ExecContext(
		ctx,
		`INSERT INTO messages (session_id, role, content, created_at)
		 VALUES (?, ?, ?, ?)`,
		session.ID,
		assistantMessage.Role,
		assistantMessage.Content,
		now,
	)
	if err != nil {
		return fmt.Errorf("save assistant message: %w", err)
	}

	assistantID, err := assistantResult.LastInsertId()
	if err != nil {
		return fmt.Errorf("retrieve assistant message ID: %w", err)
	}

	// Update session metadata within the same transaction.
	updateResult, err := tx.ExecContext(
		ctx,
		`UPDATE sessions
		 SET title = ?, updated_at = ?
		 WHERE id = ?`,
		session.Title,
		now,
		session.ID,
	)
	if err != nil {
		return fmt.Errorf("update session metadata: %w", err)
	}

	affected, err := updateResult.RowsAffected()
	if err != nil {
		return fmt.Errorf("check session update: %w", err)
	}
	if affected == 0 {
		return storage.ErrNotFound
	}

	// Commit all three operations together.
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("commit exchange transaction: %w", err)
	}

	// Update in-memory objects only after a successful commit.
	userMessage.ID = userID
	userMessage.CreatedAt = now

	assistantMessage.ID = assistantID
	assistantMessage.CreatedAt = now

	session.UpdatedAt = now

	return nil
}
