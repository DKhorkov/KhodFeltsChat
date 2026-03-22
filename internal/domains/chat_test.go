package domains_test

import (
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
)

func TestChat_IsValid(t *testing.T) {
	t.Parallel()

	now := time.Now()

	// Вспомогательные данные
	defaultTitle := "Test Chat"
	defaultDescription := "Test Description"
	defaultUser := domains.User{ID: 1, Username: "testuser"}
	anotherUser := domains.User{ID: 2, Username: "anotheruser"}

	tests := []struct {
		name     string
		chat     domains.Chat
		expected bool
	}{
		{
			name: "valid private chat with two members",
			chat: domains.Chat{
				ID:        1,
				Title:     &defaultTitle,
				Type:      domains.ChatTypePrivate,
				CreatedAt: now,
				UpdatedAt: now,
				IsRead:    true,
				Members:   []domains.User{defaultUser, anotherUser},
			},
			expected: true,
		},
		{
			name: "valid group chat with multiple members",
			chat: domains.Chat{
				ID:          2,
				Title:       &defaultTitle,
				Description: &defaultDescription,
				Type:        domains.ChatTypeGroup,
				CreatedAt:   now,
				UpdatedAt:   now,
				IsRead:      false,
				Members: []domains.User{
					defaultUser,
					anotherUser,
					{ID: 3, Username: "thirduser"},
				},
			},
			expected: true,
		},
		{
			name: "valid chat with minimum members (one member) - group",
			chat: domains.Chat{
				ID:        3,
				Type:      domains.ChatTypeGroup,
				CreatedAt: now,
				UpdatedAt: now,
				Members:   []domains.User{defaultUser},
			},
			expected: true,
		},
		{
			name: "valid chat without title and description",
			chat: domains.Chat{
				ID:        4,
				Type:      domains.ChatTypeGroup,
				CreatedAt: now,
				UpdatedAt: now,
				Members:   []domains.User{defaultUser, anotherUser},
			},
			expected: true,
		},
		{
			name: "invalid chat type",
			chat: domains.Chat{
				ID:        5,
				Type:      "invalid_type",
				CreatedAt: now,
				UpdatedAt: now,
				Members:   []domains.User{defaultUser},
			},
			expected: false,
		},
		{
			name: "empty chat type",
			chat: domains.Chat{
				ID:        6,
				Type:      "",
				CreatedAt: now,
				UpdatedAt: now,
				Members:   []domains.User{defaultUser},
			},
			expected: false,
		},
		{
			name: "no members",
			chat: domains.Chat{
				ID:        7,
				Type:      domains.ChatTypePrivate,
				CreatedAt: now,
				UpdatedAt: now,
				Members:   []domains.User{},
			},
			expected: false,
		},
		{
			name: "nil members slice",
			chat: domains.Chat{
				ID:        8,
				Type:      domains.ChatTypeGroup,
				CreatedAt: now,
				UpdatedAt: now,
				Members:   nil,
			},
			expected: false,
		},
		{
			name: "valid chat with messages",
			chat: domains.Chat{
				ID:        9,
				Type:      domains.ChatTypePrivate,
				CreatedAt: now,
				UpdatedAt: now,
				Members:   []domains.User{defaultUser, anotherUser},
				Messages: []domains.Message{
					{ID: 1, Text: "Hello"},
				},
			},
			expected: true,
		},
		{
			name: "case sensitivity test for chat type",
			chat: domains.Chat{
				ID:        10,
				Type:      "PRIVATE", // должен быть "private" (в нижнем регистре)
				CreatedAt: now,
				UpdatedAt: now,
				Members:   []domains.User{defaultUser},
			},
			expected: false,
		},
		{
			name: "valid chat with all fields populated",
			chat: domains.Chat{
				ID:          11,
				Title:       &defaultTitle,
				Description: &defaultDescription,
				Type:        domains.ChatTypeGroup,
				CreatedAt:   now.Add(-24 * time.Hour),
				UpdatedAt:   now,
				IsRead:      true,
				Members:     []domains.User{defaultUser, anotherUser},
				Messages: []domains.Message{
					{ID: 1, Text: "First message"},
					{ID: 2, Text: "Second message"},
				},
			},
			expected: true,
		},
		{
			name: "invalid private chat - one member",
			chat: domains.Chat{
				ID:          11,
				Title:       &defaultTitle,
				Description: &defaultDescription,
				Type:        domains.ChatTypePrivate,
				CreatedAt:   now.Add(-24 * time.Hour),
				UpdatedAt:   now,
				IsRead:      true,
				Members:     []domains.User{defaultUser},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := tt.chat.IsValid()
			if result != tt.expected {
				t.Errorf("IsValid() = %v, want %v", result, tt.expected)
			}
		})
	}
}

func TestChatTypeConstants(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		chatType domains.ChatType
		expected string
	}{
		{
			name:     "private chat type",
			chatType: domains.ChatTypePrivate,
			expected: "private",
		},
		{
			name:     "group chat type",
			chatType: domains.ChatTypeGroup,
			expected: "group",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			if string(tt.chatType) != tt.expected {
				t.Errorf("ChatType = %q, want %q", tt.chatType, tt.expected)
			}
		})
	}
}
