package storage

import (
	"context"
	"errors"

	conversation "github.com/Rugved7/termai/internal/conversation"
)

var ErrNotFound = errors.New("record not found")

type SessionRepository interface {
	CreateSession(
		ctx context.Context,
		session *conversation.Session,
	) error

	GetByID(
		ctx context.Context,
		id string,
	) (*conversation.Session, error)

	List(
		ctx context.Context,
		limit int,
	) ([]conversation.Session, error)

	Update(
		ctx context.Context,
		session *conversation.Session,
	) error

	Delete(
		ctx context.Context,
		id string,
	) error
}

type MessageRepository interface {
	CreateMessage(
		ctx context.Context,
		message *conversation.Message,
	) error

	ListBySession(
		ctx context.Context,
		sessionID string,
	) ([]conversation.Message, error)
}

type Store interface {
	SessionRepository
	MessageRepository

	SaveExchange(
		ctx context.Context,
		session *conversation.Session,
		userMessage *conversation.Message,
		assistantMessage *conversation.Message,
	) error

	Close() error
}
