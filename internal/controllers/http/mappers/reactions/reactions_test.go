package reactions_test

import (
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/mappers/reactions"
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/stretchr/testify/assert"
)

func TestMapReaction(t *testing.T) {
	t.Parallel()

	got := reactions.MapReaction(domains.Reaction{ID: 1, Emoji: "👍"})
	assert.Equal(t, schemas.Reaction{ID: 1, Emoji: "👍"}, got)
}

func TestMapReactions_PreservesOrder(t *testing.T) {
	t.Parallel()

	src := []domains.Reaction{
		{ID: 1, Emoji: "👍"},
		{ID: 2, Emoji: "❤️"},
	}
	got := reactions.MapReactions(src)
	assert.Equal(t, []schemas.Reaction{
		{ID: 1, Emoji: "👍"},
		{ID: 2, Emoji: "❤️"},
	}, got)
}

func TestMapMessageReaction_CopiesUserIDs(t *testing.T) {
	t.Parallel()

	src := domains.MessageReactionSummary{
		Reaction: domains.Reaction{ID: 1, Emoji: "👍"},
		UserIDs:  []uint64{10, 20},
	}
	got := reactions.MapMessageReaction(src)
	assert.Equal(t, schemas.MessageReaction{
		Reaction: schemas.Reaction{ID: 1, Emoji: "👍"},
		UserIDs:  []uint64{10, 20},
	}, got)

	// Убедимся, что копия — мутация исходника не влияет
	src.UserIDs[0] = 999

	assert.Equal(t, uint64(10), got.UserIDs[0])
}

func TestMapMessageReactions_EmptyInput_ReturnsNil(t *testing.T) {
	t.Parallel()

	assert.Nil(t, reactions.MapMessageReactions(nil))
	assert.Nil(t, reactions.MapMessageReactions([]domains.MessageReactionSummary{}))
}

func TestMapMessageReactions_MapsAllElements(t *testing.T) {
	t.Parallel()

	src := []domains.MessageReactionSummary{
		{Reaction: domains.Reaction{ID: 1, Emoji: "👍"}, UserIDs: []uint64{1}},
		{Reaction: domains.Reaction{ID: 2, Emoji: "❤️"}, UserIDs: []uint64{2, 3}},
	}
	got := reactions.MapMessageReactions(src)
	assert.Len(t, got, 2)
	assert.Equal(t, "👍", got[0].Reaction.Emoji)
	assert.Equal(t, []uint64{2, 3}, got[1].UserIDs)
}
