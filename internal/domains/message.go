package domains

type Message struct {
	ChatID string  `json:"chatId"`
	Text   string  `json:"text"`
	Sender *Sender `json:"sender,omitempty"`
}

type Sender struct {
	UserID   uint64 `json:"userId"`
	Username string `json:"username"`
}
