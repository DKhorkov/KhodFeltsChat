package domains_test

import (
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/domains"
)

func TestMessage_NewMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		want *domains.Message
	}{
		{
			name: "create new empty message",
			want: &domains.Message{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := domains.NewMessage()
			if got == nil {
				t.Fatal("NewMessage() returned nil")
			}

			// Проверяем, что поля инициализированы нулевыми значениями
			if got.ID != 0 {
				t.Errorf("NewMessage().ID = %v, want 0", got.ID)
			}

			if got.ChatID != 0 {
				t.Errorf("NewMessage().ChatID = %v, want 0", got.ChatID)
			}

			if got.Text != "" {
				t.Errorf("NewMessage().Text = %v, want empty string", got.Text)
			}

			if got.Sender.ID != 0 {
				t.Errorf("NewMessage().Sender should be zero value")
			}

			if !got.CreatedAt.IsZero() {
				t.Errorf("NewMessage().CreatedAt should be zero time")
			}

			if !got.UpdatedAt.IsZero() {
				t.Errorf("NewMessage().UpdatedAt should be zero time")
			}
		})
	}
}

func TestMessage_From(t *testing.T) {
	t.Parallel()

	now := time.Now()

	users := []domains.User{
		{ID: 1, Username: "user1", CreatedAt: now},
		{ID: 2, Username: "user2", CreatedAt: now.Add(-time.Hour)},
		{ID: 0, Username: "anonymous"}, // zero value user
	}

	tests := []struct {
		name    string
		message *domains.Message
		user    domains.User
		want    domains.User
	}{
		{
			name:    "set sender for empty message",
			message: &domains.Message{},
			user:    users[0],
			want:    users[0],
		},
		{
			name: "set sender for existing message",
			message: &domains.Message{
				ID:        100,
				ChatID:    1,
				Text:      "Hello",
				CreatedAt: now.Add(-30 * time.Minute),
			},
			user: users[1],
			want: users[1],
		},
		{
			name:    "set zero value sender",
			message: &domains.Message{},
			user:    users[2],
			want:    users[2],
		},
		{
			name: "override existing sender",
			message: &domains.Message{
				Sender: users[0],
			},
			user: users[1],
			want: users[1],
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Сохраняем исходное состояние для проверки неизменности других полей
			originalMessage := *tt.message

			result := tt.message.From(tt.user)

			// Проверяем возвращаемое значение (должен быть тот же указатель)
			if result != tt.message {
				t.Error("From() should return the same message pointer")
			}

			// Проверяем, что отправитель установлен корректно
			if tt.message.Sender != tt.want {
				t.Errorf("From().Sender = %v, want %v", tt.message.Sender, tt.want)
			}

			// Проверяем, что другие поля не изменились
			if tt.message.ID != originalMessage.ID {
				t.Errorf("From() changed ID from %v to %v", originalMessage.ID, tt.message.ID)
			}

			if tt.message.ChatID != originalMessage.ChatID {
				t.Errorf(
					"From() changed ChatID from %v to %v",
					originalMessage.ChatID,
					tt.message.ChatID,
				)
			}

			if tt.message.Text != originalMessage.Text {
				t.Errorf("From() changed Text from %q to %q", originalMessage.Text, tt.message.Text)
			}

			if !tt.message.CreatedAt.Equal(originalMessage.CreatedAt) {
				t.Errorf(
					"From() changed CreatedAt from %v to %v",
					originalMessage.CreatedAt,
					tt.message.CreatedAt,
				)
			}

			if !tt.message.UpdatedAt.Equal(originalMessage.UpdatedAt) {
				t.Errorf(
					"From() changed UpdatedAt from %v to %v",
					originalMessage.UpdatedAt,
					tt.message.UpdatedAt,
				)
			}
		})
	}
}

func TestMessage_Received(t *testing.T) {
	t.Parallel()

	beforeCall := time.Now()

	tests := []struct {
		name    string
		message *domains.Message
	}{
		{
			name:    "set received time for empty message",
			message: &domains.Message{},
		},
		{
			name: "set received time for message with existing time",
			message: &domains.Message{
				ID:        1,
				CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "set received time for fully populated message",
			message: &domains.Message{
				ID:        2,
				ChatID:    100,
				Sender:    domains.User{ID: 1, Username: "test"},
				Text:      "Test message",
				CreatedAt: beforeCall.Add(-time.Hour),
				UpdatedAt: beforeCall.Add(-30 * time.Minute),
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Сохраняем исходное состояние
			originalMessage := *tt.message

			result := tt.message.Received()

			// Проверяем возвращаемое значение
			if result != tt.message {
				t.Error("Received() should return the same message pointer")
			}

			// Проверяем, что время установлено (должно быть после вызова)
			if tt.message.CreatedAt.IsZero() {
				t.Error("Received() should set CreatedAt")
			}

			if tt.message.CreatedAt.Before(beforeCall) {
				t.Errorf(
					"CreatedAt %v should be after test start %v",
					tt.message.CreatedAt,
					beforeCall,
				)
			}

			// Проверяем, что другие поля не изменились
			if tt.message.ID != originalMessage.ID {
				t.Errorf("Received() changed ID from %v to %v", originalMessage.ID, tt.message.ID)
			}

			if tt.message.ChatID != originalMessage.ChatID {
				t.Errorf(
					"Received() changed ChatID from %v to %v",
					originalMessage.ChatID,
					tt.message.ChatID,
				)
			}

			if tt.message.Sender != originalMessage.Sender {
				t.Errorf(
					"Received() changed Sender from %v to %v",
					originalMessage.Sender,
					tt.message.Sender,
				)
			}

			if tt.message.Text != originalMessage.Text {
				t.Errorf(
					"Received() changed Text from %q to %q",
					originalMessage.Text,
					tt.message.Text,
				)
			}

			if !tt.message.UpdatedAt.Equal(originalMessage.UpdatedAt) {
				t.Errorf(
					"Received() changed UpdatedAt from %v to %v",
					originalMessage.UpdatedAt,
					tt.message.UpdatedAt,
				)
			}
		})
	}
}

func TestMessage_Updated(t *testing.T) {
	t.Parallel()

	beforeCall := time.Now()

	tests := []struct {
		name    string
		message *domains.Message
	}{
		{
			name:    "set updated time for empty message",
			message: &domains.Message{},
		},
		{
			name: "set updated time for message with existing time",
			message: &domains.Message{
				ID:        1,
				UpdatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
			},
		},
		{
			name: "set updated time for fully populated message",
			message: &domains.Message{
				ID:        2,
				ChatID:    100,
				Sender:    domains.User{ID: 1, Username: "test"},
				Text:      "Test message",
				CreatedAt: beforeCall.Add(-time.Hour),
				UpdatedAt: beforeCall.Add(-30 * time.Minute),
			},
		},
		{
			name: "set updated time when CreatedAt is zero",
			message: &domains.Message{
				ID:     3,
				Text:   "New message",
				Sender: domains.User{ID: 2, Username: "user2"},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			// Сохраняем исходное состояние
			originalMessage := *tt.message

			result := tt.message.Updated()

			// Проверяем возвращаемое значение
			if result != tt.message {
				t.Error("Updated() should return the same message pointer")
			}

			// Проверяем, что время обновления установлено
			if tt.message.UpdatedAt.IsZero() {
				t.Error("Updated() should set UpdatedAt")
			}

			if tt.message.UpdatedAt.Before(beforeCall) {
				t.Errorf(
					"UpdatedAt %v should be after test start %v",
					tt.message.UpdatedAt,
					beforeCall,
				)
			}

			// Проверяем, что другие поля не изменились
			if tt.message.ID != originalMessage.ID {
				t.Errorf("Updated() changed ID from %v to %v", originalMessage.ID, tt.message.ID)
			}

			if tt.message.ChatID != originalMessage.ChatID {
				t.Errorf(
					"Updated() changed ChatID from %v to %v",
					originalMessage.ChatID,
					tt.message.ChatID,
				)
			}

			if tt.message.Sender != originalMessage.Sender {
				t.Errorf(
					"Updated() changed Sender from %v to %v",
					originalMessage.Sender,
					tt.message.Sender,
				)
			}

			if tt.message.Text != originalMessage.Text {
				t.Errorf(
					"Updated() changed Text from %q to %q",
					originalMessage.Text,
					tt.message.Text,
				)
			}

			if !tt.message.CreatedAt.Equal(originalMessage.CreatedAt) {
				t.Errorf(
					"Updated() changed CreatedAt from %v to %v",
					originalMessage.CreatedAt,
					tt.message.CreatedAt,
				)
			}
		})
	}
}

func TestMessage_MethodChaining(t *testing.T) {
	t.Parallel()

	user := domains.User{ID: 1, Username: "testuser"}

	tests := []struct {
		name     string
		chain    func() *domains.Message
		validate func(*domains.Message) bool
	}{
		{
			name: "chain From and Received",
			chain: func() *domains.Message {
				return domains.NewMessage().
					From(user).
					Received()
			},
			validate: func(m *domains.Message) bool {
				return m.Sender == user && !m.CreatedAt.IsZero() && m.UpdatedAt.IsZero()
			},
		},
		{
			name: "chain all methods",
			chain: func() *domains.Message {
				return domains.NewMessage().
					From(user).
					Received().
					Updated()
			},
			validate: func(m *domains.Message) bool {
				return m.Sender == user && !m.CreatedAt.IsZero() && !m.UpdatedAt.IsZero()
			},
		},
		{
			name: "chain with multiple updates",
			chain: func() *domains.Message {
				msg := domains.NewMessage().
					From(user).
					Received()

				// Имитируем изменение сообщения
				msg.Text = "Updated text"

				return msg.Updated()
			},
			validate: func(m *domains.Message) bool {
				return m.Sender == user && !m.CreatedAt.IsZero() && !m.UpdatedAt.IsZero() &&
					m.Text == "Updated text"
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			msg := tt.chain()

			if msg == nil {
				t.Fatal("Method chain returned nil")
			}

			if !tt.validate(msg) {
				t.Errorf("Method chain validation failed for message: %+v", msg)
			}
		})
	}
}
