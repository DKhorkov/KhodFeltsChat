package push_subscriptions_test

import (
	"testing"
	"time"

	pushsubscriptions "github.com/DKhorkov/kfc/internal/controllers/http/mappers/push_subscriptions"
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/stretchr/testify/assert"
)

func TestMapCreateResponse(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		input    domains.PushSubscription
		expected schemas.CreatePushSubscriptionResponse
	}{
		{
			name: "Полная подписка",
			input: domains.PushSubscription{
				ID:            42,
				UserID:        1,
				Endpoint:      "https://push.example.com/sub/123",
				EncryptionKey: "BEl0...",
				Auth:          "auth-secret",
				CreatedAt:     time.Now().UTC(),
			},
			expected: schemas.CreatePushSubscriptionResponse{
				ID: 42,
			},
		},
		{
			name: "Подписка с нулевым ID",
			input: domains.PushSubscription{
				ID:            0,
				UserID:        1,
				Endpoint:      "https://push.example.com/sub/0",
				EncryptionKey: "key",
				Auth:          "auth",
				CreatedAt:     time.Time{},
			},
			expected: schemas.CreatePushSubscriptionResponse{
				ID: 0,
			},
		},
		{
			name: "Подписка с максимальным ID",
			input: domains.PushSubscription{
				ID:            ^uint64(0),
				UserID:        ^uint64(0),
				Endpoint:      "https://push.example.com/sub/max",
				EncryptionKey: "key",
				Auth:          "auth",
				CreatedAt:     time.Now().UTC(),
			},
			expected: schemas.CreatePushSubscriptionResponse{
				ID: ^uint64(0),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := pushsubscriptions.MapCreateResponse(tt.input)
			assert.Equal(t, tt.expected.ID, result.ID)
		})
	}
}
