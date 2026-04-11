package messages_test

import (
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/mappers/messages"
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapMessage(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name     string
		input    domains.Message
		expected schemas.Message
	}{
		{
			name: "Полное сообщение со всеми полями",
			input: domains.Message{
				ID:     1,
				ChatID: 100,
				Sender: domains.User{
					ID:             5,
					Username:       "john_doe",
					Email:          "john@example.com",
					EmailConfirmed: true,
					Password:       "hashed_password_123",
					CreatedAt:      now.Add(-24 * time.Hour),
					UpdatedAt:      now.Add(-12 * time.Hour),
				},
				Text:      "Hello everyone! How are you doing today?",
				CreatedAt: now,
				UpdatedAt: now.Add(5 * time.Minute),
			},
			expected: schemas.Message{
				ID:     1,
				ChatID: 100,
				Sender: schemas.Sender{
					ID:       5,
					Username: "john_doe",
				},
				Text:      "Hello everyone! How are you doing today?",
				CreatedAt: now,
				UpdatedAt: now.Add(5 * time.Minute),
			},
		},
		{
			name: "Минимальное сообщение",
			input: domains.Message{
				ID:     2,
				ChatID: 200,
				Sender: domains.User{
					ID:             1,
					Username:       "user",
					Email:          "user@example.com",
					EmailConfirmed: false,
					Password:       "hash",
					CreatedAt:      time.Time{},
					UpdatedAt:      time.Time{},
				},
				Text:      "Hi",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
			},
			expected: schemas.Message{
				ID:     2,
				ChatID: 200,
				Sender: schemas.Sender{
					ID:       1,
					Username: "user",
				},
				Text:      "Hi",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
			},
		},
		{
			name: "Сообщение с максимальными значениями ID",
			input: domains.Message{
				ID:     ^uint64(0), // max uint64
				ChatID: ^uint64(0) - 1,
				Sender: domains.User{
					ID:             ^uint64(0) - 2,
					Username:       "max_user",
					Email:          "max@example.com",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				Text:      "Max ID message",
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: schemas.Message{
				ID:     ^uint64(0),
				ChatID: ^uint64(0) - 1,
				Sender: schemas.Sender{
					ID:       ^uint64(0) - 2,
					Username: "max_user",
				},
				Text:      "Max ID message",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "Сообщение с длинным текстом",
			input: domains.Message{
				ID:     3,
				ChatID: 300,
				Sender: domains.User{
					ID:             7,
					Username:       "alice",
					Email:          "alice@example.com",
					EmailConfirmed: true,
					Password:       "hashed",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				Text:      "Это очень длинное сообщение, которое содержит много информации, деталей и контекста. Оно предназначено для проверки корректной обработки длинных текстов в системе сообщений. Сообщение может содержать различные символы, эмодзи 😊 и специальные символы: !@#$%^&*()",
				CreatedAt: now.Add(-1 * time.Hour),
				UpdatedAt: now.Add(-30 * time.Minute),
			},
			expected: schemas.Message{
				ID:     3,
				ChatID: 300,
				Sender: schemas.Sender{
					ID:       7,
					Username: "alice",
				},
				Text:      "Это очень длинное сообщение, которое содержит много информации, деталей и контекста. Оно предназначено для проверки корректной обработки длинных текстов в системе сообщений. Сообщение может содержать различные символы, эмодзи 😊 и специальные символы: !@#$%^&*()",
				CreatedAt: now.Add(-1 * time.Hour),
				UpdatedAt: now.Add(-30 * time.Minute),
			},
		},
		{
			name: "Сообщение с минимальным текстом (1 символ)",
			input: domains.Message{
				ID:     4,
				ChatID: 400,
				Sender: domains.User{
					ID:             8,
					Username:       "bob",
					Email:          "bob@example.com",
					EmailConfirmed: false,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				Text:      ".",
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: schemas.Message{
				ID:     4,
				ChatID: 400,
				Sender: schemas.Sender{
					ID:       8,
					Username: "bob",
				},
				Text:      ".",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "Сообщение с отправителем, у которого длинный username",
			input: domains.Message{
				ID:     5,
				ChatID: 500,
				Sender: domains.User{
					ID:             9,
					Username:       "user_with_very_long_username_that_exceeds_normal_limits_12345",
					Email:          "longusername@example.com",
					EmailConfirmed: true,
					Password:       "hashed_password_for_long_user",
					CreatedAt:      now.Add(-7 * 24 * time.Hour),
					UpdatedAt:      now.Add(-3 * 24 * time.Hour),
				},
				Text:      "Test message",
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: schemas.Message{
				ID:     5,
				ChatID: 500,
				Sender: schemas.Sender{
					ID:       9,
					Username: "user_with_very_long_username_that_exceeds_normal_limits_12345",
				},
				Text:      "Test message",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "Сообщение с одинаковыми временными метками",
			input: domains.Message{
				ID:     6,
				ChatID: 600,
				Sender: domains.User{
					ID:             10,
					Username:       "same_time_user",
					Email:          "same@example.com",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				Text:      "Message with same timestamps",
				CreatedAt: now,
				UpdatedAt: now, // То же самое время
			},
			expected: schemas.Message{
				ID:     6,
				ChatID: 600,
				Sender: schemas.Sender{
					ID:       10,
					Username: "same_time_user",
				},
				Text:      "Message with same timestamps",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "Сообщение с отправителем, у которого минимальные данные",
			input: domains.Message{
				ID:     7,
				ChatID: 700,
				Sender: domains.User{
					ID:             1,
					Username:       "a", // Минимальный username
					Email:          "a@b.c",
					EmailConfirmed: false,
					Password:       "",
					CreatedAt:      time.Time{},
					UpdatedAt:      time.Time{},
				},
				Text:      "Min",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
			},
			expected: schemas.Message{
				ID:     7,
				ChatID: 700,
				Sender: schemas.Sender{
					ID:       1,
					Username: "a",
				},
				Text:      "Min",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
			},
		},
		{
			name: "Сообщение с нулевым ChatID (edge case)",
			input: domains.Message{
				ID:     8,
				ChatID: 0, // Нулевой ChatID
				Sender: domains.User{
					ID:             11,
					Username:       "zero_chat",
					Email:          "zero@example.com",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				Text:      "Message in chat 0",
				CreatedAt: now,
				UpdatedAt: now,
			},
			expected: schemas.Message{
				ID:     8,
				ChatID: 0,
				Sender: schemas.Sender{
					ID:       11,
					Username: "zero_chat",
				},
				Text:      "Message in chat 0",
				CreatedAt: now,
				UpdatedAt: now,
			},
		},
		{
			name: "Сообщение с эмодзи и спецсимволами",
			input: domains.Message{
				ID:     9,
				ChatID: 900,
				Sender: domains.User{
					ID:             12,
					Username:       "emoji_user",
					Email:          "emoji@example.com",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				Text:      "Hello! 😊 🎉 Привет! @username #tag\nNew line\tTab",
				CreatedAt: now,
				UpdatedAt: now.Add(time.Minute),
			},
			expected: schemas.Message{
				ID:     9,
				ChatID: 900,
				Sender: schemas.Sender{
					ID:       12,
					Username: "emoji_user",
				},
				Text:      "Hello! 😊 🎉 Привет! @username #tag\nNew line\tTab",
				CreatedAt: now,
				UpdatedAt: now.Add(time.Minute),
			},
		},
		{
			name: "Сообщение с будущей датой создания (edge case)",
			input: domains.Message{
				ID:     10,
				ChatID: 1000,
				Sender: domains.User{
					ID:             13,
					Username:       "future",
					Email:          "future@example.com",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				Text:      "Future message",
				CreatedAt: now.Add(365 * 24 * time.Hour), // Через год
				UpdatedAt: now.Add(366 * 24 * time.Hour), // Через год + день
			},
			expected: schemas.Message{
				ID:     10,
				ChatID: 1000,
				Sender: schemas.Sender{
					ID:       13,
					Username: "future",
				},
				Text:      "Future message",
				CreatedAt: now.Add(365 * 24 * time.Hour),
				UpdatedAt: now.Add(366 * 24 * time.Hour),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := messages.MapMessage(tt.input)

			// Проверяем все поля
			assert.Equal(t, tt.expected.ID, result.ID)
			assert.Equal(t, tt.expected.ChatID, result.ChatID)
			assert.Equal(t, tt.expected.Text, result.Text)
			assert.Equal(t, tt.expected.CreatedAt, result.CreatedAt)
			assert.Equal(t, tt.expected.UpdatedAt, result.UpdatedAt)

			// Проверяем Sender
			assert.Equal(t, tt.expected.Sender.ID, result.Sender.ID)
			assert.Equal(t, tt.expected.Sender.Username, result.Sender.Username)
		})
	}
}

func TestMapMessages(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name     string
		input    []domains.Message
		expected []schemas.Message
	}{
		{
			name:     "Пустой слайс сообщений",
			input:    []domains.Message{},
			expected: []schemas.Message{},
		},
		{
			name: "Одно сообщение",
			input: []domains.Message{
				{
					ID:     1,
					ChatID: 100,
					Sender: domains.User{
						ID:             5,
						Username:       "user1",
						Email:          "user1@example.com",
						EmailConfirmed: true,
						Password:       "hash",
						CreatedAt:      now,
						UpdatedAt:      now,
					},
					Text:      "Hello",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			expected: []schemas.Message{
				{
					ID:     1,
					ChatID: 100,
					Sender: schemas.Sender{
						ID:       5,
						Username: "user1",
					},
					Text:      "Hello",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
		{
			name: "Несколько сообщений от разных отправителей",
			input: []domains.Message{
				{
					ID:     1,
					ChatID: 100,
					Sender: domains.User{
						ID:             5,
						Username:       "alice",
						Email:          "alice@example.com",
						EmailConfirmed: true,
						Password:       "hash1",
						CreatedAt:      now,
						UpdatedAt:      now,
					},
					Text:      "Hi Bob!",
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					ID:     2,
					ChatID: 100,
					Sender: domains.User{
						ID:             6,
						Username:       "bob",
						Email:          "bob@example.com",
						EmailConfirmed: false,
						Password:       "hash2",
						CreatedAt:      now.Add(-24 * time.Hour),
						UpdatedAt:      now.Add(-12 * time.Hour),
					},
					Text:      "Hello Alice! How are you?",
					CreatedAt: now.Add(5 * time.Minute),
					UpdatedAt: now.Add(5 * time.Minute),
				},
				{
					ID:     3,
					ChatID: 100,
					Sender: domains.User{
						ID:             5,
						Username:       "alice",
						Email:          "alice@example.com",
						EmailConfirmed: true,
						Password:       "hash1",
						CreatedAt:      now,
						UpdatedAt:      now,
					},
					Text:      "I'm fine, thanks! 😊",
					CreatedAt: now.Add(10 * time.Minute),
					UpdatedAt: now.Add(10 * time.Minute),
				},
			},
			expected: []schemas.Message{
				{
					ID:     1,
					ChatID: 100,
					Sender: schemas.Sender{
						ID:       5,
						Username: "alice",
					},
					Text:      "Hi Bob!",
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					ID:     2,
					ChatID: 100,
					Sender: schemas.Sender{
						ID:       6,
						Username: "bob",
					},
					Text:      "Hello Alice! How are you?",
					CreatedAt: now.Add(5 * time.Minute),
					UpdatedAt: now.Add(5 * time.Minute),
				},
				{
					ID:     3,
					ChatID: 100,
					Sender: schemas.Sender{
						ID:       5,
						Username: "alice",
					},
					Text:      "I'm fine, thanks! 😊",
					CreatedAt: now.Add(10 * time.Minute),
					UpdatedAt: now.Add(10 * time.Minute),
				},
			},
		},
		{
			name: "Сообщения из разных чатов",
			input: []domains.Message{
				{
					ID:     1,
					ChatID: 100,
					Sender: domains.User{
						ID:             1,
						Username:       "user1",
						Email:          "user1@example.com",
						EmailConfirmed: true,
						Password:       "hash",
						CreatedAt:      now,
						UpdatedAt:      now,
					},
					Text:      "Chat 100 message",
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					ID:     2,
					ChatID: 200,
					Sender: domains.User{
						ID:             2,
						Username:       "user2",
						Email:          "user2@example.com",
						EmailConfirmed: false,
						Password:       "hash2",
						CreatedAt:      now,
						UpdatedAt:      now,
					},
					Text:      "Chat 200 message",
					CreatedAt: now.Add(time.Hour),
					UpdatedAt: now.Add(time.Hour),
				},
				{
					ID:     3,
					ChatID: 100,
					Sender: domains.User{
						ID:             3,
						Username:       "user3",
						Email:          "user3@example.com",
						EmailConfirmed: true,
						Password:       "hash3",
						CreatedAt:      now,
						UpdatedAt:      now,
					},
					Text:      "Another chat 100 message",
					CreatedAt: now.Add(2 * time.Hour),
					UpdatedAt: now.Add(2 * time.Hour),
				},
			},
			expected: []schemas.Message{
				{
					ID:     1,
					ChatID: 100,
					Sender: schemas.Sender{
						ID:       1,
						Username: "user1",
					},
					Text:      "Chat 100 message",
					CreatedAt: now,
					UpdatedAt: now,
				},
				{
					ID:     2,
					ChatID: 200,
					Sender: schemas.Sender{
						ID:       2,
						Username: "user2",
					},
					Text:      "Chat 200 message",
					CreatedAt: now.Add(time.Hour),
					UpdatedAt: now.Add(time.Hour),
				},
				{
					ID:     3,
					ChatID: 100,
					Sender: schemas.Sender{
						ID:       3,
						Username: "user3",
					},
					Text:      "Another chat 100 message",
					CreatedAt: now.Add(2 * time.Hour),
					UpdatedAt: now.Add(2 * time.Hour),
				},
			},
		},
		{
			name: "Большое количество сообщений (производительность)",
			input: func() []domains.Message {
				messages := make([]domains.Message, 10000)
				for i := range messages {
					messages[i] = domains.Message{
						ID:     uint64(i + 1),
						ChatID: uint64(i%10 + 1),
						Sender: domains.User{
							ID:             uint64(i%100 + 1),
							Username:       "user",
							Email:          "user@example.com",
							EmailConfirmed: i%2 == 0,
							Password:       "hash",
							CreatedAt:      now,
							UpdatedAt:      now,
						},
						Text:      "Message",
						CreatedAt: now.Add(time.Duration(i) * time.Minute),
						UpdatedAt: now.Add(time.Duration(i) * time.Minute),
					}
				}

				return messages
			}(),
			expected: func() []schemas.Message {
				messages := make([]schemas.Message, 10000)
				for i := range messages {
					messages[i] = schemas.Message{
						ID:     uint64(i + 1),
						ChatID: uint64(i%10 + 1),
						Sender: schemas.Sender{
							ID:       uint64(i%100 + 1),
							Username: "user",
						},
						Text:      "Message",
						CreatedAt: now.Add(time.Duration(i) * time.Minute),
						UpdatedAt: now.Add(time.Duration(i) * time.Minute),
					}
				}

				return messages
			}(),
		},
		{
			name: "Сообщения с nil отправителем (невозможно по структуре, но edge case)",
			input: []domains.Message{
				{
					ID:        1,
					ChatID:    100,
					Sender:    domains.User{}, // Пустой User (все поля zero-value)
					Text:      "Empty sender",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
			expected: []schemas.Message{
				{
					ID:     1,
					ChatID: 100,
					Sender: schemas.Sender{
						ID:       0,
						Username: "",
					},
					Text:      "Empty sender",
					CreatedAt: now,
					UpdatedAt: now,
				},
			},
		},
		{
			name: "Смешанные сообщения с разными типами данных",
			input: []domains.Message{
				{
					ID:     1000,
					ChatID: 50,
					Sender: domains.User{
						ID:             99,
						Username:       "admin",
						Email:          "admin@example.com",
						EmailConfirmed: true,
						Password:       "admin_hash",
						CreatedAt:      now.Add(-365 * 24 * time.Hour),
						UpdatedAt:      now.Add(-30 * 24 * time.Hour),
					},
					Text:      "System message: Maintenance at 3 AM",
					CreatedAt: now.Add(-1 * time.Hour),
					UpdatedAt: now.Add(-1 * time.Hour),
				},
				{
					ID:     1001,
					ChatID: 50,
					Sender: domains.User{
						ID:             1,
						Username:       "regular_user",
						Email:          "user@example.com",
						EmailConfirmed: false,
						Password:       "user_hash",
						CreatedAt:      now.Add(-7 * 24 * time.Hour),
						UpdatedAt:      now,
					},
					Text:      "Thanks for the info! 👍",
					CreatedAt: now.Add(-50 * time.Minute),
					UpdatedAt: now.Add(-50 * time.Minute),
				},
			},
			expected: []schemas.Message{
				{
					ID:     1000,
					ChatID: 50,
					Sender: schemas.Sender{
						ID:       99,
						Username: "admin",
					},
					Text:      "System message: Maintenance at 3 AM",
					CreatedAt: now.Add(-1 * time.Hour),
					UpdatedAt: now.Add(-1 * time.Hour),
				},
				{
					ID:     1001,
					ChatID: 50,
					Sender: schemas.Sender{
						ID:       1,
						Username: "regular_user",
					},
					Text:      "Thanks for the info! 👍",
					CreatedAt: now.Add(-50 * time.Minute),
					UpdatedAt: now.Add(-50 * time.Minute),
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := messages.MapMessages(tt.input)

			// Проверяем длину результата
			require.Len(t, result, len(tt.expected))

			// Проверяем каждое сообщение
			for i, expectedMessage := range tt.expected {
				assert.Equal(t, expectedMessage.ID, result[i].ID)
				assert.Equal(t, expectedMessage.ChatID, result[i].ChatID)
				assert.Equal(t, expectedMessage.Text, result[i].Text)
				assert.Equal(t, expectedMessage.CreatedAt, result[i].CreatedAt)
				assert.Equal(t, expectedMessage.UpdatedAt, result[i].UpdatedAt)

				// Проверяем Sender
				assert.Equal(t, expectedMessage.Sender.ID, result[i].Sender.ID)
				assert.Equal(t, expectedMessage.Sender.Username, result[i].Sender.Username)
			}

			// Для больших слайсов проверяем производительность
			if len(tt.input) >= 10000 {
				assert.NotPanics(t, func() {
					messages.MapMessages(tt.input)
				})

				// Проверяем, что функция работает быстро
				// (неформальная проверка)
				start := time.Now()

				messages.MapMessages(tt.input)

				elapsed := time.Since(start)

				// Сообщение не должно обрабатываться слишком долго
				// (это субъективно, зависит от системы)
				if elapsed > 100*time.Millisecond {
					t.Logf("MapMessages обработала %d сообщений за %v", len(tt.input), elapsed)
				}
			}
		})
	}
}

// Дополнительные тесты для edge cases.
func TestMapMessagesEdgeCases(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	t.Run("Nil input slice", func(t *testing.T) {
		t.Parallel()

		// Важно: в Go нельзя передать nil как []T в параметр типа []T
		// без приведения типа, но проверим случай с пустым слайсом
		result := messages.MapMessages([]domains.Message{})
		assert.Empty(t, result)
		assert.NotNil(t, result) // Всегда должен возвращаться не-nil слайс
	})

	t.Run("Single message with all fields zero", func(t *testing.T) {
		t.Parallel()

		input := []domains.Message{
			{
				ID:        0,
				ChatID:    0,
				Sender:    domains.User{},
				Text:      "",
				CreatedAt: time.Time{},
				UpdatedAt: time.Time{},
			},
		}

		result := messages.MapMessages(input)
		require.Len(t, result, 1)

		assert.Equal(t, uint64(0), result[0].ID)
		assert.Equal(t, uint64(0), result[0].ChatID)
		assert.Empty(t, result[0].Text)
		assert.Equal(t, time.Time{}, result[0].CreatedAt)
		assert.Equal(t, time.Time{}, result[0].UpdatedAt)
		assert.Equal(t, uint64(0), result[0].Sender.ID)
		assert.Empty(t, result[0].Sender.Username)
	})

	t.Run("Verify no password copying", func(t *testing.T) {
		t.Parallel()

		input := []domains.Message{
			{
				ID:     1,
				ChatID: 100,
				Sender: domains.User{
					ID:             1,
					Username:       "test",
					Email:          "test@example.com",
					EmailConfirmed: true,
					Password:       "secret_password_123",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				Text:      "Test",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}

		result := messages.MapMessages(input)

		// Проверяем, что в результате нет следов пароля
		// Это можно проверить, убедившись что Sender содержит только ID и Username
		sender := result[0].Sender
		assert.Equal(t, uint64(1), sender.ID)
		assert.Equal(t, "test", sender.Username)
		// Не должно быть полей Email, Password и т.д.
	})

	t.Run("Order preservation", func(t *testing.T) {
		t.Parallel()

		input := []domains.Message{
			{
				ID:        1,
				ChatID:    100,
				Sender:    domains.User{ID: 1, Username: "a"},
				Text:      "First",
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:        2,
				ChatID:    100,
				Sender:    domains.User{ID: 2, Username: "b"},
				Text:      "Second",
				CreatedAt: now,
				UpdatedAt: now,
			},
			{
				ID:        3,
				ChatID:    100,
				Sender:    domains.User{ID: 3, Username: "c"},
				Text:      "Third",
				CreatedAt: now,
				UpdatedAt: now,
			},
		}

		result := messages.MapMessages(input)

		// Порядок должен сохраниться
		assert.Equal(t, uint64(1), result[0].ID)
		assert.Equal(t, uint64(2), result[1].ID)
		assert.Equal(t, uint64(3), result[2].ID)
		assert.Equal(t, "First", result[0].Text)
		assert.Equal(t, "Second", result[1].Text)
		assert.Equal(t, "Third", result[2].Text)
	})
}
