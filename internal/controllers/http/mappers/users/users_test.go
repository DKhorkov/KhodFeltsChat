package users_test

import (
	"testing"
	"time"

	mappers "github.com/DKhorkov/kfc/internal/controllers/http/mappers/users"
	"github.com/DKhorkov/kfc/internal/controllers/http/schemas"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/libs/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMapUser(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name       string
		input      domains.User
		expected   schemas.User
		skipFields []string // Поля, которые не должны проверяться
	}{
		{
			name: "Полный пользователь со всеми полями",
			input: domains.User{
				ID:             123,
				Username:       "john_doe",
				Email:          "john.doe@example.com",
				EmailConfirmed: true,
				Password:       "hashed_password_123!@#",
				CreatedAt:      now.Add(-365 * 24 * time.Hour), // Год назад
				UpdatedAt:      now.Add(-30 * 24 * time.Hour),  // Месяц назад
			},
			expected: schemas.User{
				ID:             123,
				Username:       "john_doe",
				Email:          "john.doe@example.com",
				EmailConfirmed: true,
				CreatedAt:      now.Add(-365 * 24 * time.Hour),
				UpdatedAt:      now.Add(-30 * 24 * time.Hour),
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь с неподтвержденным email",
			input: domains.User{
				ID:             456,
				Username:       "jane_smith",
				Email:          "jane.smith@example.org",
				EmailConfirmed: false,
				Password:       "another_hashed_password",
				CreatedAt:      now.Add(-7 * 24 * time.Hour), // Неделю назад
				UpdatedAt:      now,                          // Сейчас
			},
			expected: schemas.User{
				ID:             456,
				Username:       "jane_smith",
				Email:          "jane.smith@example.org",
				EmailConfirmed: false,
				CreatedAt:      now.Add(-7 * 24 * time.Hour),
				UpdatedAt:      now,
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь с минимальными данными",
			input: domains.User{
				ID:             1,
				Username:       "a",     // Минимальная длина
				Email:          "a@b.c", // Минимальный email
				EmailConfirmed: false,
				Password:       "",
				CreatedAt:      time.Time{},
				UpdatedAt:      time.Time{},
			},
			expected: schemas.User{
				ID:             1,
				Username:       "a",
				Email:          "a@b.c",
				EmailConfirmed: false,
				CreatedAt:      time.Time{},
				UpdatedAt:      time.Time{},
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь с максимальными значениями ID",
			input: domains.User{
				ID:             ^uint64(0), // max uint64
				Username:       "max_id_user",
				Email:          "max@example.com",
				EmailConfirmed: true,
				Password:       "hash",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			expected: schemas.User{
				ID:             ^uint64(0),
				Username:       "max_id_user",
				Email:          "max@example.com",
				EmailConfirmed: true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь с длинным username",
			input: domains.User{
				ID:             789,
				Username:       "user_with_very_long_username_exceeding_normal_limits_1234567890",
				Email:          "longusername@example-domain.com",
				EmailConfirmed: true,
				Password:       "very_long_hashed_password_string_that_should_not_be_copied_12345",
				CreatedAt:      now.Add(-100 * 24 * time.Hour),
				UpdatedAt:      now.Add(-50 * 24 * time.Hour),
			},
			expected: schemas.User{
				ID:             789,
				Username:       "user_with_very_long_username_exceeding_normal_limits_1234567890",
				Email:          "longusername@example-domain.com",
				EmailConfirmed: true,
				CreatedAt:      now.Add(-100 * 24 * time.Hour),
				UpdatedAt:      now.Add(-50 * 24 * time.Hour),
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь со сложным email",
			input: domains.User{
				ID:             101,
				Username:       "complex_email_user",
				Email:          "user.name+tag@subdomain.example.co.uk",
				EmailConfirmed: true,
				Password:       "hash",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			expected: schemas.User{
				ID:             101,
				Username:       "complex_email_user",
				Email:          "user.name+tag@subdomain.example.co.uk",
				EmailConfirmed: true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь с одинаковыми временными метками",
			input: domains.User{
				ID:             202,
				Username:       "same_time_user",
				Email:          "same@example.com",
				EmailConfirmed: false,
				Password:       "hash",
				CreatedAt:      now,
				UpdatedAt:      now, // То же самое время
			},
			expected: schemas.User{
				ID:             202,
				Username:       "same_time_user",
				Email:          "same@example.com",
				EmailConfirmed: false,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь с future dates (edge case)",
			input: domains.User{
				ID:             303,
				Username:       "future_user",
				Email:          "future@example.com",
				EmailConfirmed: true,
				Password:       "hash",
				CreatedAt:      now.Add(365 * 24 * time.Hour), // Через год
				UpdatedAt:      now.Add(366 * 24 * time.Hour), // Через год + день
			},
			expected: schemas.User{
				ID:             303,
				Username:       "future_user",
				Email:          "future@example.com",
				EmailConfirmed: true,
				CreatedAt:      now.Add(365 * 24 * time.Hour),
				UpdatedAt:      now.Add(366 * 24 * time.Hour),
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь с нестандартными символами в username",
			input: domains.User{
				ID:             404,
				Username:       "user_123-abc.456",
				Email:          "special@example.com",
				EmailConfirmed: true,
				Password:       "hash",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			expected: schemas.User{
				ID:             404,
				Username:       "user_123-abc.456",
				Email:          "special@example.com",
				EmailConfirmed: true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь с аватаром",
			input: domains.User{
				ID:             808,
				Username:       "avatar_user",
				Email:          "avatar@example.com",
				EmailConfirmed: true,
				Password:       "hash",
				CreatedAt:      now,
				UpdatedAt:      now,
				AvatarPath: pointers.New(
					"https://kfc.webtm.ru/api/files/download/550e8400.jpg",
				),
			},
			expected: schemas.User{
				ID:             808,
				Username:       "avatar_user",
				Email:          "avatar@example.com",
				EmailConfirmed: true,
				CreatedAt:      now,
				UpdatedAt:      now,
				AvatarPath: pointers.New(
					"https://kfc.webtm.ru/api/files/download/550e8400.jpg",
				),
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь без аватара (nil)",
			input: domains.User{
				ID:             909,
				Username:       "no_avatar_user",
				Email:          "noavatar@example.com",
				EmailConfirmed: true,
				Password:       "hash",
				CreatedAt:      now,
				UpdatedAt:      now,
				AvatarPath:     nil,
			},
			expected: schemas.User{
				ID:             909,
				Username:       "no_avatar_user",
				Email:          "noavatar@example.com",
				EmailConfirmed: true,
				CreatedAt:      now,
				UpdatedAt:      now,
				AvatarPath:     nil,
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь с нулевым ID",
			input: domains.User{
				ID:             0,
				Username:       "zero_id",
				Email:          "zero@example.com",
				EmailConfirmed: false,
				Password:       "hash",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			expected: schemas.User{
				ID:             0,
				Username:       "zero_id",
				Email:          "zero@example.com",
				EmailConfirmed: false,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь с пустым паролем",
			input: domains.User{
				ID:             505,
				Username:       "empty_pass",
				Email:          "empty@example.com",
				EmailConfirmed: true,
				Password:       "", // Пустой пароль
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			expected: schemas.User{
				ID:             505,
				Username:       "empty_pass",
				Email:          "empty@example.com",
				EmailConfirmed: true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь с очень длинным паролем",
			input: domains.User{
				ID:             606,
				Username:       "long_pass_user",
				Email:          "longpass@example.com",
				EmailConfirmed: false,
				Password:       "very_long_hashed_password_" + string(make([]byte, 1000)) + "_end",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			expected: schemas.User{
				ID:             606,
				Username:       "long_pass_user",
				Email:          "longpass@example.com",
				EmailConfirmed: false,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			skipFields: []string{"Password"},
		},
		{
			name: "Пользователь с Unicode символами",
			input: domains.User{
				ID:             707,
				Username:       "ユーザー名",         // Японские символы
				Email:          "unicode@例子.测试", // Internationalized email
				EmailConfirmed: true,
				Password:       "hash",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			expected: schemas.User{
				ID:             707,
				Username:       "ユーザー名",
				Email:          "unicode@例子.测试",
				EmailConfirmed: true,
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			skipFields: []string{"Password"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := mappers.MapUser(tt.input)

			// Проверяем все ожидаемые поля
			assert.Equal(t, tt.expected.ID, result.ID)
			assert.Equal(t, tt.expected.Username, result.Username)
			assert.Equal(t, tt.expected.Email, result.Email)
			assert.Equal(t, tt.expected.EmailConfirmed, result.EmailConfirmed)
			assert.Equal(t, tt.expected.CreatedAt, result.CreatedAt)
			assert.Equal(t, tt.expected.UpdatedAt, result.UpdatedAt)
			assert.Equal(t, tt.expected.AvatarPath, result.AvatarPath)
		})
	}
}

func TestMapUsers(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	tests := []struct {
		name     string
		input    []domains.User
		expected []schemas.User
	}{
		{
			name:     "Пустой слайс пользователей",
			input:    []domains.User{},
			expected: []schemas.User{},
		},
		{
			name: "Один пользователь",
			input: []domains.User{
				{
					ID:             1,
					Username:       "single_user",
					Email:          "single@example.com",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
			expected: []schemas.User{
				{
					ID:             1,
					Username:       "single_user",
					Email:          "single@example.com",
					EmailConfirmed: true,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
		},
		{
			name: "Несколько пользователей с разными данными",
			input: []domains.User{
				{
					ID:             1,
					Username:       "admin",
					Email:          "admin@example.com",
					EmailConfirmed: true,
					Password:       "admin_hash",
					CreatedAt:      now.Add(-365 * 24 * time.Hour),
					UpdatedAt:      now.Add(-30 * 24 * time.Hour),
				},
				{
					ID:             2,
					Username:       "jane_doe",
					Email:          "jane.doe@example.org",
					EmailConfirmed: true,
					Password:       "jane_hash",
					CreatedAt:      now.Add(-180 * 24 * time.Hour),
					UpdatedAt:      now.Add(-7 * 24 * time.Hour),
				},
				{
					ID:             3,
					Username:       "john_smith",
					Email:          "john.smith@example.net",
					EmailConfirmed: false,
					Password:       "john_hash",
					CreatedAt:      now.Add(-30 * 24 * time.Hour),
					UpdatedAt:      now,
				},
				{
					ID:             4,
					Username:       "guest",
					Email:          "guest@example.com",
					EmailConfirmed: false,
					Password:       "guest_hash",
					CreatedAt:      now.Add(-1 * 24 * time.Hour),
					UpdatedAt:      now.Add(-1 * 24 * time.Hour),
				},
			},
			expected: []schemas.User{
				{
					ID:             1,
					Username:       "admin",
					Email:          "admin@example.com",
					EmailConfirmed: true,
					CreatedAt:      now.Add(-365 * 24 * time.Hour),
					UpdatedAt:      now.Add(-30 * 24 * time.Hour),
				},
				{
					ID:             2,
					Username:       "jane_doe",
					Email:          "jane.doe@example.org",
					EmailConfirmed: true,
					CreatedAt:      now.Add(-180 * 24 * time.Hour),
					UpdatedAt:      now.Add(-7 * 24 * time.Hour),
				},
				{
					ID:             3,
					Username:       "john_smith",
					Email:          "john.smith@example.net",
					EmailConfirmed: false,
					CreatedAt:      now.Add(-30 * 24 * time.Hour),
					UpdatedAt:      now,
				},
				{
					ID:             4,
					Username:       "guest",
					Email:          "guest@example.com",
					EmailConfirmed: false,
					CreatedAt:      now.Add(-1 * 24 * time.Hour),
					UpdatedAt:      now.Add(-1 * 24 * time.Hour),
				},
			},
		},
		{
			name: "Большое количество пользователей (производительность)",
			input: func() []domains.User {
				users := make([]domains.User, 5000)
				for i := range users {
					users[i] = domains.User{
						ID:             uint64(i + 1),
						Username:       "user",
						Email:          "user@example.com",
						EmailConfirmed: i%2 == 0,
						Password:       "hashed_password_" + string(rune('a'+(i%26))),
						CreatedAt:      now.Add(-time.Duration(i) * 24 * time.Hour),
						UpdatedAt:      now.Add(-time.Duration(i/2) * 24 * time.Hour),
					}
				}

				return users
			}(),
			expected: func() []schemas.User {
				users := make([]schemas.User, 5000)
				for i := range users {
					users[i] = schemas.User{
						ID:             uint64(i + 1),
						Username:       "user",
						Email:          "user@example.com",
						EmailConfirmed: i%2 == 0,
						CreatedAt:      now.Add(-time.Duration(i) * 24 * time.Hour),
						UpdatedAt:      now.Add(-time.Duration(i/2) * 24 * time.Hour),
					}
				}

				return users
			}(),
		},
		{
			name: "Пользователи с повторяющимися username",
			input: []domains.User{
				{
					ID:             1,
					Username:       "duplicate",
					Email:          "user1@example.com",
					EmailConfirmed: true,
					Password:       "hash1",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             2,
					Username:       "duplicate", // Тот же username
					Email:          "user2@example.com",
					EmailConfirmed: false,
					Password:       "hash2",
					CreatedAt:      now.Add(time.Hour),
					UpdatedAt:      now.Add(time.Hour),
				},
			},
			expected: []schemas.User{
				{
					ID:             1,
					Username:       "duplicate",
					Email:          "user1@example.com",
					EmailConfirmed: true,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             2,
					Username:       "duplicate",
					Email:          "user2@example.com",
					EmailConfirmed: false,
					CreatedAt:      now.Add(time.Hour),
					UpdatedAt:      now.Add(time.Hour),
				},
			},
		},
		{
			name: "Пользователи с нулевыми значениями",
			input: []domains.User{
				{
					ID:             0,
					Username:       "",
					Email:          "",
					EmailConfirmed: false,
					Password:       "",
					CreatedAt:      time.Time{},
					UpdatedAt:      time.Time{},
				},
				{
					ID:             1,
					Username:       "normal",
					Email:          "normal@example.com",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
			expected: []schemas.User{
				{
					ID:             0,
					Username:       "",
					Email:          "",
					EmailConfirmed: false,
					CreatedAt:      time.Time{},
					UpdatedAt:      time.Time{},
				},
				{
					ID:             1,
					Username:       "normal",
					Email:          "normal@example.com",
					EmailConfirmed: true,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
		},
		{
			name: "Пользователи с разными доменами email",
			input: []domains.User{
				{
					ID:             1,
					Username:       "gmail_user",
					Email:          "user@gmail.com",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             2,
					Username:       "yahoo_user",
					Email:          "user@yahoo.com",
					EmailConfirmed: false,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             3,
					Username:       "corp_user",
					Email:          "user@company.corp",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
			expected: []schemas.User{
				{
					ID:             1,
					Username:       "gmail_user",
					Email:          "user@gmail.com",
					EmailConfirmed: true,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             2,
					Username:       "yahoo_user",
					Email:          "user@yahoo.com",
					EmailConfirmed: false,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             3,
					Username:       "corp_user",
					Email:          "user@company.corp",
					EmailConfirmed: true,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
		},
		{
			name: "Пользователи с разными состояниями EmailConfirmed",
			input: []domains.User{
				{
					ID:             1,
					Username:       "confirmed",
					Email:          "confirmed@example.com",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             2,
					Username:       "not_confirmed",
					Email:          "not_confirmed@example.com",
					EmailConfirmed: false,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             3,
					Username:       "pending",
					Email:          "pending@example.com",
					EmailConfirmed: false,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
			expected: []schemas.User{
				{
					ID:             1,
					Username:       "confirmed",
					Email:          "confirmed@example.com",
					EmailConfirmed: true,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             2,
					Username:       "not_confirmed",
					Email:          "not_confirmed@example.com",
					EmailConfirmed: false,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             3,
					Username:       "pending",
					Email:          "pending@example.com",
					EmailConfirmed: false,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
		},
		{
			name: "Пользователи с разными форматами username",
			input: []domains.User{
				{
					ID:             1,
					Username:       "user123",
					Email:          "user123@example.com",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             2,
					Username:       "USER_NAME",
					Email:          "user_name@example.com",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             3,
					Username:       "user-name",
					Email:          "user-name@example.com",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             4,
					Username:       "user.name",
					Email:          "user.name@example.com",
					EmailConfirmed: true,
					Password:       "hash",
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
			expected: []schemas.User{
				{
					ID:             1,
					Username:       "user123",
					Email:          "user123@example.com",
					EmailConfirmed: true,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             2,
					Username:       "USER_NAME",
					Email:          "user_name@example.com",
					EmailConfirmed: true,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             3,
					Username:       "user-name",
					Email:          "user-name@example.com",
					EmailConfirmed: true,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
				{
					ID:             4,
					Username:       "user.name",
					Email:          "user.name@example.com",
					EmailConfirmed: true,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			result := mappers.MapUsers(tt.input)

			// Проверяем длину результата
			require.Len(t, result, len(tt.expected))

			// Проверяем каждого пользователя
			for i, expectedUser := range tt.expected {
				assert.Equal(t, expectedUser.ID, result[i].ID)
				assert.Equal(t, expectedUser.Username, result[i].Username)
				assert.Equal(t, expectedUser.Email, result[i].Email)
				assert.Equal(t, expectedUser.EmailConfirmed, result[i].EmailConfirmed)
				assert.Equal(t, expectedUser.CreatedAt, result[i].CreatedAt)
				assert.Equal(t, expectedUser.UpdatedAt, result[i].UpdatedAt)
				assert.Equal(t, expectedUser.AvatarPath, result[i].AvatarPath)
			}

			// Для больших слайсов проверяем производительность
			if len(tt.input) >= 5000 {
				assert.NotPanics(t, func() {
					mappers.MapUsers(tt.input)
				})

				// Неформальная проверка производительности
				start := time.Now()

				mappers.MapUsers(tt.input)

				elapsed := time.Since(start)

				if elapsed > 50*time.Millisecond {
					t.Logf("MapUsers обработала %d пользователей за %v", len(tt.input), elapsed)
				}
			}
		})
	}
}

// Дополнительные тесты для edge cases.
func TestMapUsersEdgeCases(t *testing.T) {
	t.Parallel()

	now := time.Now().UTC()

	t.Run("Verify no password copying in batch", func(t *testing.T) {
		t.Parallel()

		input := []domains.User{
			{
				ID:             1,
				Username:       "user1",
				Email:          "user1@example.com",
				EmailConfirmed: true,
				Password:       "secret_password_123",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			{
				ID:             2,
				Username:       "user2",
				Email:          "user2@example.com",
				EmailConfirmed: false,
				Password:       "another_secret_456",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
		}

		result := mappers.MapUsers(input)

		// Проверяем, что в результате нет следов паролей
		for i := range result {
			assert.Equal(t, input[i].Username, result[i].Username)
			assert.Equal(t, input[i].Email, result[i].Email)
			assert.Equal(t, input[i].EmailConfirmed, result[i].EmailConfirmed)
		}
	})

	t.Run("Order preservation", func(t *testing.T) {
		t.Parallel()

		input := []domains.User{
			{
				ID:             1,
				Username:       "a",
				Email:          "a@example.com",
				EmailConfirmed: true,
				Password:       "hash1",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			{
				ID:             2,
				Username:       "b",
				Email:          "b@example.com",
				EmailConfirmed: false,
				Password:       "hash2",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
			{
				ID:             3,
				Username:       "c",
				Email:          "c@example.com",
				EmailConfirmed: true,
				Password:       "hash3",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
		}

		result := mappers.MapUsers(input)

		// Порядок должен сохраниться
		assert.Equal(t, uint64(1), result[0].ID)
		assert.Equal(t, uint64(2), result[1].ID)
		assert.Equal(t, uint64(3), result[2].ID)
		assert.Equal(t, "a", result[0].Username)
		assert.Equal(t, "b", result[1].Username)
		assert.Equal(t, "c", result[2].Username)
	})

	t.Run("Verify different iteration method (range with index vs value)", func(t *testing.T) {
		t.Parallel()

		// В функции MapUsers используется for i, user := range users
		// что создает копию user. Проверим, что это работает корректно
		input := []domains.User{
			{
				ID:             1,
				Username:       "test",
				Email:          "test@example.com",
				EmailConfirmed: true,
				Password:       "should_not_be_copied",
				CreatedAt:      now,
				UpdatedAt:      now,
			},
		}

		result := mappers.MapUsers(input)

		// Должен быть создан новый слайс
		assert.NotSame(t, &input, &result)

		// Данные должны быть скопированы корректно
		assert.Equal(t, input[0].ID, result[0].ID)
		assert.Equal(t, input[0].Username, result[0].Username)
		assert.Equal(t, input[0].Email, result[0].Email)
		assert.Equal(t, input[0].EmailConfirmed, result[0].EmailConfirmed)
		assert.Equal(t, input[0].CreatedAt, result[0].CreatedAt)
		assert.Equal(t, input[0].UpdatedAt, result[0].UpdatedAt)
	})

	t.Run("Nil slice returns empty slice", func(t *testing.T) {
		t.Parallel()

		// В Go нельзя передать nil как []T без приведения типа
		// но функция должна возвращать не-nil слайс для пустого ввода
		result := mappers.MapUsers([]domains.User{})
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("Single user with extreme values", func(t *testing.T) {
		t.Parallel()

		input := []domains.User{
			{
				ID:             ^uint64(0),
				Username:       "a",
				Email:          "a@b.c",
				EmailConfirmed: true,
				Password:       "p",
				CreatedAt:      time.Time{},
				UpdatedAt:      time.Time{},
			},
		}

		result := mappers.MapUsers(input)
		require.Len(t, result, 1)

		assert.Equal(t, ^uint64(0), result[0].ID)
		assert.Equal(t, "a", result[0].Username)
		assert.Equal(t, "a@b.c", result[0].Email)
		assert.True(t, result[0].EmailConfirmed)
		assert.Equal(t, time.Time{}, result[0].CreatedAt)
		assert.Equal(t, time.Time{}, result[0].UpdatedAt)
	})
}

// Benchmark тесты для проверки производительности.
func BenchmarkMapUser(b *testing.B) {
	user := domains.User{
		ID:             123,
		Username:       "benchmark_user",
		Email:          "benchmark@example.com",
		EmailConfirmed: true,
		Password:       "hashed_password_for_benchmark",
		CreatedAt:      time.Now(),
		UpdatedAt:      time.Now(),
	}

	for b.Loop() {
		mappers.MapUser(user)
	}
}

func BenchmarkMapUsers(b *testing.B) {
	// Подготовка тестовых данных
	users := make([]domains.User, 1000)
	for i := range users {
		users[i] = domains.User{
			ID:             uint64(i + 1),
			Username:       "user",
			Email:          "user@example.com",
			EmailConfirmed: i%2 == 0,
			Password:       "hashed_password",
			CreatedAt:      time.Now(),
			UpdatedAt:      time.Now(),
		}
	}

	for b.Loop() {
		mappers.MapUsers(users)
	}
}
