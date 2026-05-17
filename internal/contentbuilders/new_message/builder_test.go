package new_message_test

import (
	"context"
	"testing"

	newmessage "github.com/DKhorkov/kfc/internal/contentbuilders/new_message"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew(t *testing.T) {
	t.Parallel()

	builder := newmessage.New()
	assert.NotNil(t, builder)
}

func TestContentBuilder_Subject(t *testing.T) {
	t.Parallel()

	builder := newmessage.New()

	tests := []struct {
		name     string
		expected string
	}{
		{
			name:     "default subject",
			expected: "Новое сообщение в чате",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := builder.Subject()
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestContentBuilder_Body(t *testing.T) {
	t.Parallel()

	groupTitle := "Группа разработки"

	tests := []struct {
		name             string
		message          domains.Message
		chat             domains.Chat
		expectedContains []string
	}{
		{
			name: "private chat without title",
			message: domains.Message{
				ID:     1,
				ChatID: 5,
				Text:   "Привет!",
				Sender: domains.User{
					ID:       2,
					Username: "alice",
				},
			},
			chat: domains.Chat{
				ID: 5,
			},
			expectedContains: []string{
				"alice",
				"Личный чат",
				"Привет!",
			},
		},
		{
			name: "group chat with title",
			message: domains.Message{
				ID:     2,
				ChatID: 10,
				Text:   "Важное обновление",
				Sender: domains.User{
					ID:       3,
					Username: "bob",
				},
			},
			chat: domains.Chat{
				ID:    10,
				Title: &groupTitle,
			},
			expectedContains: []string{
				"bob",
				"Группа разработки",
				"Важное обновление",
			},
		},
		{
			name: "message with special HTML characters",
			message: domains.Message{
				ID:     3,
				ChatID: 1,
				Text:   "<script>alert('xss')</script>",
				Sender: domains.User{
					ID:       4,
					Username: "charlie",
				},
			},
			chat: domains.Chat{
				ID: 1,
			},
			expectedContains: []string{
				"charlie",
				"Личный чат",
				"<script>alert('xss')</script>",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			builder := newmessage.New()

			result, err := builder.Body(context.Background(), tt.message, tt.chat)
			require.NoError(t, err)

			for _, expected := range tt.expectedContains {
				assert.Contains(t, result, expected)
			}
		})
	}
}

func TestContentBuilder_Body_PrivateChatFallback(t *testing.T) {
	t.Parallel()

	builder := newmessage.New()

	result, err := builder.Body(
		context.Background(),
		domains.Message{
			Text:   "Тест",
			Sender: domains.User{Username: "user"},
		},
		domains.Chat{},
	)

	require.NoError(t, err)
	assert.Contains(t, result, "Личный чат")
	assert.NotContains(t, result, "<nil>")
}

func TestContentBuilder_Body_GroupChatUsesTitle(t *testing.T) {
	t.Parallel()

	builder := newmessage.New()
	title := "Мой чат"

	result, err := builder.Body(
		context.Background(),
		domains.Message{
			Text:   "Тест",
			Sender: domains.User{Username: "user"},
		},
		domains.Chat{Title: &title},
	)

	require.NoError(t, err)
	assert.Contains(t, result, "Мой чат")
	assert.NotContains(t, result, "Личный чат")
}
