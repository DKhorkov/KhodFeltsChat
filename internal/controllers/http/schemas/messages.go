package schemas

import "time"

type Message struct {
	ID        uint64    `json:"id"`
	ChatID    uint64    `json:"chatId"`
	Sender    Sender    `json:"sender"`
	Text      string    `json:"text"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type Sender struct {
	ID       uint64 `json:"id"`
	Username string `json:"username"`
}
