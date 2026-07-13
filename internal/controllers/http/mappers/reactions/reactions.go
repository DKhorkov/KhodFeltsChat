package reactions

import (
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
)

func MapReaction(r domains.Reaction) schemas.Reaction {
	return schemas.Reaction{ID: r.ID, Emoji: r.Emoji, SortOrder: r.SortOrder}
}

func MapReactions(rs []domains.Reaction) []schemas.Reaction {
	result := make([]schemas.Reaction, len(rs))
	for i := range rs {
		result[i] = MapReaction(rs[i])
	}

	return result
}

func MapMessageReaction(s domains.MessageReactionSummary) schemas.MessageReaction {
	userIDs := make([]uint64, len(s.UserIDs))
	copy(userIDs, s.UserIDs)

	return schemas.MessageReaction{
		Reaction: MapReaction(s.Reaction),
		UserIDs:  userIDs,
	}
}

func MapMessageReactions(ss []domains.MessageReactionSummary) []schemas.MessageReaction {
	if len(ss) == 0 {
		return nil
	}

	result := make([]schemas.MessageReaction, len(ss))
	for i := range ss {
		result[i] = MapMessageReaction(ss[i])
	}

	return result
}
