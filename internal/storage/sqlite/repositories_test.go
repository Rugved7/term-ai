package sqlite

import (
	"context"
	"errors"
	"testing"

	"github.com/Rugved7/termai/internal/conversation"
	"github.com/Rugved7/termai/internal/storage"
)

func newRepositoryTestDB(t *testing.T) *DB {
	t.Helper()

	db, err := New(t.TempDir())
	if err != nil {
		t.Fatalf("initialize database: %v", err)
	}

	t.Cleanup(func() {
		if err := db.Close(); err != nil {
			t.Errorf("close database: %v", err)
		}
	})

	return db
}

func TestSessionRepository(t *testing.T) {
	db := newRepositoryTestDB(t)
	ctx := context.Background()

	session := &conversation.Session{
		ID:       "session-1",
		Title:    "Go concurrency",
		Provider: "ollama",
		Model:    "llama3.2:3b",
	}

	// Create
	if err := db.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	if session.CreatedAt.IsZero() || session.UpdatedAt.IsZero() {
		t.Fatal("CreateSession() did not populate timestamps")
	}

	// Retrieve
	got, err := db.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if got.Title != session.Title || got.Model != session.Model {
		t.Fatalf("GetByID() = %+v, want title %q and model %q",
			got, session.Title, session.Model)
	}

	// Update
	got.Title = "Updated conversation"

	if err := db.Update(ctx, got); err != nil {
		t.Fatalf("Update() error = %v", err)
	}

	updated, err := db.GetByID(ctx, session.ID)
	if err != nil {
		t.Fatalf("GetByID() after update error = %v", err)
	}

	if updated.Title != "Updated conversation" {
		t.Fatalf("updated title = %q, want %q",
			updated.Title, "Updated conversation")
	}

	// List
	sessions, err := db.List(ctx, 10)
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}

	if len(sessions) != 1 {
		t.Fatalf("List() returned %d sessions, want 1", len(sessions))
	}

	// Delete
	if err := db.Delete(ctx, session.ID); err != nil {
		t.Fatalf("Delete() error = %v", err)
	}

	_, err = db.GetByID(ctx, session.ID)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Fatalf("GetByID() after delete error = %v, want ErrNotFound", err)
	}
}

func TestMessageRepository(t *testing.T) {
	db := newRepositoryTestDB(t)
	ctx := context.Background()

	session := &conversation.Session{
		ID:       "session-2",
		Title:    "Test conversation",
		Provider: "ollama",
		Model:    "llama3.2:3b",
	}

	if err := db.CreateSession(ctx, session); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	first := &conversation.Message{
		SessionID: session.ID,
		Role:      "user",
		Content:   "Explain goroutines",
	}

	second := &conversation.Message{
		SessionID: session.ID,
		Role:      "assistant",
		Content:   "Goroutines are lightweight concurrent functions.",
	}

	if err := db.CreateMessage(ctx, first); err != nil {
		t.Fatalf("CreateMessage(first) error = %v", err)
	}

	if err := db.CreateMessage(ctx, second); err != nil {
		t.Fatalf("CreateMessage(second) error = %v", err)
	}

	if first.ID == 0 || second.ID == 0 {
		t.Fatal("CreateMessage() did not populate generated IDs")
	}

	messages, err := db.ListBySession(ctx, session.ID)
	if err != nil {
		t.Fatalf("ListBySession() error = %v", err)
	}

	if len(messages) != 2 {
		t.Fatalf("ListBySession() returned %d messages, want 2", len(messages))
	}

	if messages[0].Content != first.Content {
		t.Errorf("first message = %q, want %q",
			messages[0].Content, first.Content)
	}

	if messages[1].Content != second.Content {
		t.Errorf("second message = %q, want %q",
			messages[1].Content, second.Content)
	}

	if messages[0].ID >= messages[1].ID {
		t.Error("messages are not returned in insertion order")
	}
}

func TestRepositoryNotFound(t *testing.T) {
	db := newRepositoryTestDB(t)
	ctx := context.Background()

	_, err := db.GetByID(ctx, "missing-session")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("GetByID() error = %v, want ErrNotFound", err)
	}

	err = db.Delete(ctx, "missing-session")
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Delete() error = %v, want ErrNotFound", err)
	}

	session := &conversation.Session{
		ID:       "missing-session",
		Title:    "Missing",
		Provider: "ollama",
		Model:    "llama3.2:3b",
	}

	err = db.Update(ctx, session)
	if !errors.Is(err, storage.ErrNotFound) {
		t.Errorf("Update() error = %v, want ErrNotFound", err)
	}
}
