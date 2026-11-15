package conversation

import "time"

type Session struct {
	ID        string
	Title     string
	Provider  string
	Model     string
	CreatedAt time.Time
	UpdatedAt time.Time
}

type Message struct {
	ID        int64
	SessionID string
	Role      string
	Content   string
	CreatedAt time.Time
}
