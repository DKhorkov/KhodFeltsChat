package schemas

// Reaction represents an emoji reaction available in the dictionary.
// swagger:model
type Reaction struct {
	// Unique identifier of the reaction.
	// required: true
	// nullable: false
	// minimum: 1
	ID uint64 `json:"id"`

	// Emoji character to display.
	// required: true
	// nullable: false
	// example: 👍
	Emoji string `json:"emoji"`
}

// MessageReaction represents an aggregated reaction on a message: the reaction
// itself and the list of user IDs who set it. Count is computed by the client
// as len(userIds).
// swagger:model
type MessageReaction struct {
	// The reaction (id + emoji).
	// required: true
	// nullable: false
	Reaction Reaction `json:"reaction"`

	// IDs of users who set this reaction.
	// required: true
	// nullable: false
	UserIDs []uint64 `json:"userIds"`
}

// SetReactionInput is the request body for POST /messages/{id}/reactions.
// swagger:parameters SetMessageReaction
type SetReactionInput struct {
	// Reaction ID from the dictionary.
	// in: body
	// required: true
	Body struct {
		ReactionID uint64 `json:"reactionId"`
	}
}
