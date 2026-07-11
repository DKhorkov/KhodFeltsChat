package domains_test

import (
	"encoding/json"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/stretchr/testify/assert"
)

func TestReaction_JSON(t *testing.T) {
	r := domains.Reaction{ID: 1, Emoji: "👍"}

	data, err := json.Marshal(r)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"id":1,"emoji":"👍"}`, string(data))
}

func TestMessageReactionSummary_JSON(t *testing.T) {
	s := domains.MessageReactionSummary{
		Reaction: domains.Reaction{ID: 1, Emoji: "👍"},
		UserIDs:  []uint64{10, 20},
	}

	data, err := json.Marshal(s)
	assert.NoError(t, err)
	assert.JSONEq(
		t,
		`{"reaction":{"id":1,"emoji":"👍"},"userIds":[10,20]}`,
		string(data),
	)
}

func TestMessageReactionDTO_JSONDecoding(t *testing.T) {
	body := `{"reactionId":5}`

	var dto domains.MessageReactionDTO

	err := json.Unmarshal([]byte(body), &dto)
	assert.NoError(t, err)
	assert.Equal(t, uint64(5), dto.ReactionID)
	assert.Equal(t, uint64(0), dto.MessageID)
	assert.Equal(t, uint64(0), dto.UserID)
}

func TestMessageReactionDTO_MessageIDAndUserIDNotSerialized(t *testing.T) {
	dto := domains.MessageReactionDTO{
		MessageID:  10,
		ReactionID: 5,
		UserID:     7,
	}

	data, err := json.Marshal(dto)
	assert.NoError(t, err)
	assert.JSONEq(t, `{"reactionId":5}`, string(data))
}
