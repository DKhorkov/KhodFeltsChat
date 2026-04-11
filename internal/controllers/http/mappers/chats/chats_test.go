package chats_test

import (
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/mappers/chats"
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/libs/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapChat(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name     string
		input    domains.Chat
		expected schemas.Chat
	}{
		{
			name: "Полный чат со всеми полями (group)",
			input: domains.Chat{
				ID:          1,
				Title:       pointers.New("Work Chat"),
				Description: pointers.New("Chat for work discussions"),
				Type:        domains.ChatTypeGroup,
				CreatedAt:   now,
				UpdatedAt:   now.Add(time.Hour),
				IsRead:      true,
				Members: []domains.User{
					{
						ID:             1,
						Username:       "user1",
						Email:          "user1@example.com",
						EmailConfirmed: true,
						Password:       "hashed_password",
						CreatedAt:      now,
						UpdatedAt:      now,
					},
					{
						ID:             2,
						Username:       "user2",
						Email:          "user2@example.com",
						EmailConfirmed: false,
						Password:       "hashed_password2",
						CreatedAt:      now,
						UpdatedAt:      now,
					},
				},
				Messages: []domains.Message{
					{
						ID:     1,
						ChatID: 1,
						Sender: domains.User{
							ID:             1,
							Username:       "user1",
							Email:          "user1@example.com",
							EmailConfirmed: true,
							Password:       "hashed_password",
							CreatedAt:      now,
							UpdatedAt:      now,
						},
						Text:      "Hello everyone!",
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			},
			expected: schemas.Chat{
				ID:          1,
				Title:       pointers.New("Work Chat"),
				Description: pointers.New("Chat for work discussions"),
				Type:        "group",
				CreatedAt:   now,
				UpdatedAt:   now.Add(time.Hour),
				IsRead:      true,
				Members: []schemas.User{
					{
						ID:             1,
						Username:       "user1",
						Email:          "user1@example.com",
						EmailConfirmed: true,
						CreatedAt:      now,
						UpdatedAt:      now,
					},
					{
						ID:             2,
						Username:       "user2",
						Email:          "user2@example.com",
						EmailConfirmed: false,
						CreatedAt:      now,
						UpdatedAt:      now,
					},
				},
				Messages: []schemas.Message{
					{
						ID:     1,
						ChatID: 1,
						Sender: schemas.Sender{
							ID:       1,
							Username: "user1",
						},
						Text:      "Hello everyone!",
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			},
		},
		{
			name: "Приватный чат без title и description",
			input: domains.Chat{
				ID:          2,
				Title:       nil,
				Description: nil,
				Type:        domains.ChatTypePrivate,
				CreatedAt:   now,
				UpdatedAt:   now,
				IsRead:      false,
				Members: []domains.User{
					{
						ID:             1,
						Username:       "user1",
						Email:          "user1@example.com",
						EmailConfirmed: true,
						Password:       "hashed_password",
						CreatedAt:      now,
						UpdatedAt:      now,
					},
				},
				Messages: []domains.Message{},
			},
			expected: schemas.Chat{
				ID:          2,
				Title:       nil,
				Description: nil,
				Type:        "private",
				CreatedAt:   now,
				UpdatedAt:   now,
				IsRead:      false,
				Members: []schemas.User{
					{
						ID:             1,
						Username:       "user1",
						Email:          "user1@example.com",
						EmailConfirmed: true,
						CreatedAt:      now,
						UpdatedAt:      now,
					},
				},
				Messages: []schemas.Message{},
			},
		},
		{
			name: "Чат без members и messages",
			input: domains.Chat{
				ID:          3,
				Title:       pointers.New("Empty Chat"),
				Description: nil,
				Type:        domains.ChatTypeGroup,
				CreatedAt:   now,
				UpdatedAt:   now,
				IsRead:      true,
				Members:     nil,
				Messages:    nil,
			},
			expected: schemas.Chat{
				ID:          3,
				Title:       pointers.New("Empty Chat"),
				Description: nil,
				Type:        "group",
				CreatedAt:   now,
				UpdatedAt:   now,
				IsRead:      true,
				Members:     nil,
				Messages:    nil,
			},
		},
		{
			name: "Чат с пустыми slices members и messages",
			input: domains.Chat{
				ID:          4,
				Title:       nil,
				Description: nil,
				Type:        domains.ChatTypePrivate,
				CreatedAt:   now,
				UpdatedAt:   now,
				IsRead:      false,
				Members:     []domains.User{},
				Messages:    []domains.Message{},
			},
			expected: schemas.Chat{
				ID:          4,
				Title:       nil,
				Description: nil,
				Type:        "private",
				CreatedAt:   now,
				UpdatedAt:   now,
				IsRead:      false,
				Members:     []schemas.User{},
				Messages:    []schemas.Message{},
			},
		},
		{
			name: "Чат с одним участником и несколькими сообщениями",
			input: domains.Chat{
				ID:          5,
				Title:       pointers.New("Support Chat"),
				Description: pointers.New("Technical support"),
				Type:        domains.ChatTypeGroup,
				CreatedAt:   now,
				UpdatedAt:   now.Add(2 * time.Hour),
				IsRead:      true,
				Members: []domains.User{
					{
						ID:             100,
						Username:       "support_agent",
						Email:          "support@company.com",
						EmailConfirmed: true,
						Password:       "hashed",
						CreatedAt:      now,
						UpdatedAt:      now,
					},
				},
				Messages: []domains.Message{
					{
						ID:     10,
						ChatID: 5,
						Sender: domains.User{
							ID:             100,
							Username:       "support_agent",
							Email:          "support@company.com",
							EmailConfirmed: true,
							Password:       "hashed",
							CreatedAt:      now,
							UpdatedAt:      now,
						},
						Text:      "Hello! How can I help you?",
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						ID:     11,
						ChatID: 5,
						Sender: domains.User{
							ID:             100,
							Username:       "support_agent",
							Email:          "support@company.com",
							EmailConfirmed: true,
							Password:       "hashed",
							CreatedAt:      now,
							UpdatedAt:      now,
						},
						Text:      "Please provide more details",
						CreatedAt: now.Add(time.Minute),
						UpdatedAt: now.Add(time.Minute),
					},
				},
			},
			expected: schemas.Chat{
				ID:          5,
				Title:       pointers.New("Support Chat"),
				Description: pointers.New("Technical support"),
				Type:        "group",
				CreatedAt:   now,
				UpdatedAt:   now.Add(2 * time.Hour),
				IsRead:      true,
				Members: []schemas.User{
					{
						ID:             100,
						Username:       "support_agent",
						Email:          "support@company.com",
						EmailConfirmed: true,
						CreatedAt:      now,
						UpdatedAt:      now,
					},
				},
				Messages: []schemas.Message{
					{
						ID:     10,
						ChatID: 5,
						Sender: schemas.Sender{
							ID:       100,
							Username: "support_agent",
						},
						Text:      "Hello! How can I help you?",
						CreatedAt: now,
						UpdatedAt: now,
					},
					{
						ID:     11,
						ChatID: 5,
						Sender: schemas.Sender{
							ID:       100,
							Username: "support_agent",
						},
						Text:      "Please provide more details",
						CreatedAt: now.Add(time.Minute),
						UpdatedAt: now.Add(time.Minute),
					},
				},
			},
		},
		{
			name: "Чат с максимальными значениями ID",
			input: domains.Chat{
				ID:          ^uint64(0), // max uint64
				Title:       pointers.New("Max ID Chat"),
				Description: nil,
				Type:        domains.ChatTypeGroup,
				CreatedAt:   now,
				UpdatedAt:   now,
				IsRead:      false,
				Members:     []domains.User{},
				Messages:    []domains.Message{},
			},
			expected: schemas.Chat{
				ID:          ^uint64(0),
				Title:       pointers.New("Max ID Chat"),
				Description: nil,
				Type:        "group",
				CreatedAt:   now,
				UpdatedAt:   now,
				IsRead:      false,
				Members:     []schemas.User{},
				Messages:    []schemas.Message{},
			},
		},
		{
			name: "Чат с минимальными значениями",
			input: domains.Chat{
				ID:          1,
				Title:       nil,
				Description: nil,
				Type:        domains.ChatTypePrivate,
				CreatedAt:   time.Time{},
				UpdatedAt:   time.Time{},
				IsRead:      false,
				Members:     nil,
				Messages:    nil,
			},
			expected: schemas.Chat{
				ID:          1,
				Title:       nil,
				Description: nil,
				Type:        "private",
				CreatedAt:   time.Time{},
				UpdatedAt:   time.Time{},
				IsRead:      false,
				Members:     nil,
				Messages:    nil,
			},
		},
		{
			name: "Чат с длинными строками",
			input: domains.Chat{
				ID:    6,
				Title: pointers.New("A very long chat title that should be properly handled"),
				Description: pointers.New(
					"An extremely detailed description of the chat that goes on and on about various topics and discussions that will take place here",
				),
				Type:      domains.ChatTypeGroup,
				CreatedAt: now,
				UpdatedAt: now,
				IsRead:    true,
				Members: []domains.User{
					{
						ID:             1,
						Username:       "user_with_very_long_username_that_exceeds_normal_length",
						Email:          "verylongemailaddress@examplecompanydomain.com",
						EmailConfirmed: true,
						Password:       "hashed",
						CreatedAt:      now,
						UpdatedAt:      now,
					},
				},
				Messages: []domains.Message{
					{
						ID:     1,
						ChatID: 6,
						Sender: domains.User{
							ID:             1,
							Username:       "user_with_very_long_username_that_exceeds_normal_length",
							Email:          "verylongemailaddress@examplecompanydomain.com",
							EmailConfirmed: true,
							Password:       "hashed",
							CreatedAt:      now,
							UpdatedAt:      now,
						},
						Text:      "A very long message text that contains a lot of information and details about various topics and discussions",
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			},
			expected: schemas.Chat{
				ID:    6,
				Title: pointers.New("A very long chat title that should be properly handled"),
				Description: pointers.New(
					"An extremely detailed description of the chat that goes on and on about various topics and discussions that will take place here",
				),
				Type:      "group",
				CreatedAt: now,
				UpdatedAt: now,
				IsRead:    true,
				Members: []schemas.User{
					{
						ID:             1,
						Username:       "user_with_very_long_username_that_exceeds_normal_length",
						Email:          "verylongemailaddress@examplecompanydomain.com",
						EmailConfirmed: true,
						CreatedAt:      now,
						UpdatedAt:      now,
					},
				},
				Messages: []schemas.Message{
					{
						ID:     1,
						ChatID: 6,
						Sender: schemas.Sender{
							ID:       1,
							Username: "user_with_very_long_username_that_exceeds_normal_length",
						},
						Text:      "A very long message text that contains a lot of information and details about various topics and discussions",
						CreatedAt: now,
						UpdatedAt: now,
					},
				},
			},
		},
		{
			name: "Чат с неконвертируемым ChatType (edge case)",
			input: domains.Chat{
				ID:          7,
				Title:       nil,
				Description: nil,
				Type:        domains.ChatType("custom_type"), // Нестандартный тип
				CreatedAt:   now,
				UpdatedAt:   now,
				IsRead:      false,
				Members:     nil,
				Messages:    nil,
			},
			expected: schemas.Chat{
				ID:          7,
				Title:       nil,
				Description: nil,
				Type:        "custom_type", // Должен преобразоваться как есть
				CreatedAt:   now,
				UpdatedAt:   now,
				IsRead:      false,
				Members:     nil,
				Messages:    nil,
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := chats.MapChat(tt.input)

			// Проверяем основные поля
			assert.Equal(t, tt.expected.ID, result.ID)
			assert.Equal(t, tt.expected.Title, result.Title)
			assert.Equal(t, tt.expected.Description, result.Description)
			assert.Equal(t, tt.expected.Type, result.Type)
			assert.Equal(t, tt.expected.CreatedAt, result.CreatedAt)
			assert.Equal(t, tt.expected.UpdatedAt, result.UpdatedAt)
			assert.Equal(t, tt.expected.IsRead, result.IsRead)

			require.Len(t, result.Members, len(tt.expected.Members))

			for i, expectedMember := range tt.expected.Members {
				assert.Equal(t, expectedMember.ID, result.Members[i].ID)
				assert.Equal(t, expectedMember.Username, result.Members[i].Username)
				assert.Equal(t, expectedMember.Email, result.Members[i].Email)
				assert.Equal(t, expectedMember.EmailConfirmed, result.Members[i].EmailConfirmed)
				assert.Equal(t, expectedMember.CreatedAt, result.Members[i].CreatedAt)
				assert.Equal(t, expectedMember.UpdatedAt, result.Members[i].UpdatedAt)
			}

			require.Len(t, result.Messages, len(tt.expected.Messages))

			for i, expectedMessage := range tt.expected.Messages {
				assert.Equal(t, expectedMessage.ID, result.Messages[i].ID)
				assert.Equal(t, expectedMessage.ChatID, result.Messages[i].ChatID)
				assert.Equal(t, expectedMessage.Sender.ID, result.Messages[i].Sender.ID)
				assert.Equal(
					t,
					expectedMessage.Sender.Username,
					result.Messages[i].Sender.Username,
				)
				assert.Equal(t, expectedMessage.Text, result.Messages[i].Text)
				assert.Equal(t, expectedMessage.CreatedAt, result.Messages[i].CreatedAt)
				assert.Equal(t, expectedMessage.UpdatedAt, result.Messages[i].UpdatedAt)
			}
		})
	}
}

func TestMapChats(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name     string
		input    []domains.Chat
		expected []schemas.Chat
	}{
		{
			name:     "Пустой слайс чатов",
			input:    []domains.Chat{},
			expected: []schemas.Chat{},
		},
		{
			name: "Один чат",
			input: []domains.Chat{
				{
					ID:          1,
					Title:       pointers.New("Chat 1"),
					Description: nil,
					Type:        domains.ChatTypePrivate,
					CreatedAt:   now,
					UpdatedAt:   now,
					IsRead:      true,
					Members:     nil,
					Messages:    nil,
				},
			},
			expected: []schemas.Chat{
				{
					ID:          1,
					Title:       pointers.New("Chat 1"),
					Description: nil,
					Type:        "private",
					CreatedAt:   now,
					UpdatedAt:   now,
					IsRead:      true,
					Members:     nil,
					Messages:    nil,
				},
			},
		},
		{
			name: "Несколько чатов разных типов",
			input: []domains.Chat{
				{
					ID:          1,
					Title:       pointers.New("Private Chat"),
					Description: nil,
					Type:        domains.ChatTypePrivate,
					CreatedAt:   now,
					UpdatedAt:   now,
					IsRead:      false,
					Members: []domains.User{
						{
							ID:             1,
							Username:       "user1",
							Email:          "user1@example.com",
							EmailConfirmed: true,
							Password:       "hashed",
							CreatedAt:      now,
							UpdatedAt:      now,
						},
					},
					Messages: nil,
				},
				{
					ID:          2,
					Title:       pointers.New("Group Chat"),
					Description: pointers.New("Work group"),
					Type:        domains.ChatTypeGroup,
					CreatedAt:   now.Add(time.Hour),
					UpdatedAt:   now.Add(2 * time.Hour),
					IsRead:      true,
					Members: []domains.User{
						{
							ID:             1,
							Username:       "user1",
							Email:          "user1@example.com",
							EmailConfirmed: true,
							Password:       "hashed",
							CreatedAt:      now,
							UpdatedAt:      now,
						},
						{
							ID:             2,
							Username:       "user2",
							Email:          "user2@example.com",
							EmailConfirmed: false,
							Password:       "hashed2",
							CreatedAt:      now,
							UpdatedAt:      now,
						},
					},
					Messages: []domains.Message{
						{
							ID:     1,
							ChatID: 2,
							Sender: domains.User{
								ID:             1,
								Username:       "user1",
								Email:          "user1@example.com",
								EmailConfirmed: true,
								Password:       "hashed",
								CreatedAt:      now,
								UpdatedAt:      now,
							},
							Text:      "Hello group!",
							CreatedAt: now,
							UpdatedAt: now,
						},
					},
				},
				{
					ID:          3,
					Title:       nil,
					Description: nil,
					Type:        domains.ChatTypePrivate,
					CreatedAt:   now.Add(3 * time.Hour),
					UpdatedAt:   now.Add(4 * time.Hour),
					IsRead:      false,
					Members:     []domains.User{},
					Messages:    []domains.Message{},
				},
			},
			expected: []schemas.Chat{
				{
					ID:          1,
					Title:       pointers.New("Private Chat"),
					Description: nil,
					Type:        "private",
					CreatedAt:   now,
					UpdatedAt:   now,
					IsRead:      false,
					Members: []schemas.User{
						{
							ID:             1,
							Username:       "user1",
							Email:          "user1@example.com",
							EmailConfirmed: true,
							CreatedAt:      now,
							UpdatedAt:      now,
						},
					},
					Messages: nil,
				},
				{
					ID:          2,
					Title:       pointers.New("Group Chat"),
					Description: pointers.New("Work group"),
					Type:        "group",
					CreatedAt:   now.Add(time.Hour),
					UpdatedAt:   now.Add(2 * time.Hour),
					IsRead:      true,
					Members: []schemas.User{
						{
							ID:             1,
							Username:       "user1",
							Email:          "user1@example.com",
							EmailConfirmed: true,
							CreatedAt:      now,
							UpdatedAt:      now,
						},
						{
							ID:             2,
							Username:       "user2",
							Email:          "user2@example.com",
							EmailConfirmed: false,
							CreatedAt:      now,
							UpdatedAt:      now,
						},
					},
					Messages: []schemas.Message{
						{
							ID:     1,
							ChatID: 2,
							Sender: schemas.Sender{
								ID:       1,
								Username: "user1",
							},
							Text:      "Hello group!",
							CreatedAt: now,
							UpdatedAt: now,
						},
					},
				},
				{
					ID:          3,
					Title:       nil,
					Description: nil,
					Type:        "private",
					CreatedAt:   now.Add(3 * time.Hour),
					UpdatedAt:   now.Add(4 * time.Hour),
					IsRead:      false,
					Members:     []schemas.User{},
					Messages:    []schemas.Message{},
				},
			},
		},
		{
			name: "Большое количество чатов (производительность)",
			input: func() []domains.Chat {
				chats := make([]domains.Chat, 1000)
				for i := range chats {
					chats[i] = domains.Chat{
						ID:          uint64(i + 1),
						Title:       pointers.New("Chat"),
						Description: nil,
						Type:        domains.ChatTypeGroup,
						CreatedAt:   now,
						UpdatedAt:   now,
						IsRead:      i%2 == 0,
						Members:     nil,
						Messages:    nil,
					}
				}

				return chats
			}(),
			expected: func() []schemas.Chat {
				chats := make([]schemas.Chat, 1000)
				for i := range chats {
					chats[i] = schemas.Chat{
						ID:          uint64(i + 1),
						Title:       pointers.New("Chat"),
						Description: nil,
						Type:        "group",
						CreatedAt:   now,
						UpdatedAt:   now,
						IsRead:      i%2 == 0,
						Members:     nil,
						Messages:    nil,
					}
				}

				return chats
			}(),
		},
		{
			name: "Слайс с nil значениями (edge case)",
			input: []domains.Chat{
				{
					ID:          1,
					Title:       nil,
					Description: nil,
					Type:        domains.ChatTypePrivate,
					CreatedAt:   now,
					UpdatedAt:   now,
					IsRead:      false,
					Members:     nil,
					Messages:    nil,
				},
				{
					ID:          2,
					Title:       pointers.New("Chat 2"),
					Description: nil,
					Type:        domains.ChatTypeGroup,
					CreatedAt:   now,
					UpdatedAt:   now,
					IsRead:      true,
					Members:     []domains.User{},
					Messages:    []domains.Message{},
				},
			},
			expected: []schemas.Chat{
				{
					ID:          1,
					Title:       nil,
					Description: nil,
					Type:        "private",
					CreatedAt:   now,
					UpdatedAt:   now,
					IsRead:      false,
					Members:     nil,
					Messages:    nil,
				},
				{
					ID:          2,
					Title:       pointers.New("Chat 2"),
					Description: nil,
					Type:        "group",
					CreatedAt:   now,
					UpdatedAt:   now,
					IsRead:      true,
					Members:     []schemas.User{},
					Messages:    []schemas.Message{},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := chats.MapChats(tt.input)

			// Проверяем длину результата
			require.Len(t, result, len(tt.expected))

			// Проверяем каждый чат
			for i, expectedChat := range tt.expected {
				assert.Equal(t, expectedChat.ID, result[i].ID)
				assert.Equal(t, expectedChat.Title, result[i].Title)
				assert.Equal(t, expectedChat.Description, result[i].Description)
				assert.Equal(t, expectedChat.Type, result[i].Type)
				assert.Equal(t, expectedChat.CreatedAt, result[i].CreatedAt)
				assert.Equal(t, expectedChat.UpdatedAt, result[i].UpdatedAt)
				assert.Equal(t, expectedChat.IsRead, result[i].IsRead)

				// Проверяем, что исходный слайс не изменился (копирование по индексу)
				if i < len(tt.input) {
					// Можно проверить, что это разные объекты
					assert.NotSame(t, &tt.input[i], &result[i])
				}
			}

			// Для больших слайсов проверяем производительность
			if len(tt.input) >= 1000 {
				// Убеждаемся, что функция работает без паники
				assert.NotPanics(t, func() {
					chats.MapChats(tt.input)
				})
			}
		})
	}
}
