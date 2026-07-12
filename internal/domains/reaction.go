package domains

type Reaction struct {
	ID    uint64 `json:"id"`
	Emoji string `json:"emoji"`
	// SortOrder — глобальный порядок отображения в UI-пикере и на сообщении.
	// Позволяет клиенту стабильно упорядочивать реакции даже после ассинхронных
	// WS-событий reaction_added, где сервер не гарантирует порядок доставки.
	SortOrder uint64 `json:"sortOrder"`
}

type MessageReactionSummary struct {
	Reaction Reaction `json:"reaction"`
	UserIDs  []uint64 `json:"userIds"`
}

type MessageReactionDTO struct {
	MessageID  uint64 `json:"-"`
	ReactionID uint64 `json:"reactionId"`
	UserID     uint64 `json:"-"`
}
