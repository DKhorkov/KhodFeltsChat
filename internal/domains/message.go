package domains

import (
	"time"
)

type Message struct {
	ID        uint64    `json:"id"`
	ChatID    uint64    `json:"chatId"`
	Sender    User      `json:"sender"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
	IsRead    bool      `json:"isRead"`
}

func NewMessage() *Message {
	return &Message{}
}

func (m *Message) From(user User) *Message {
	m.Sender = user

	return m
}

func (m *Message) Received() *Message {
	m.CreatedAt = time.Now()

	return m
}

func (m *Message) Updated() *Message {
	m.UpdatedAt = time.Now()

	return m
}
