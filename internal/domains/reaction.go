package domains

type Reaction struct {
	ID    uint64 `json:"id"`
	Emoji string `json:"emoji"`
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
