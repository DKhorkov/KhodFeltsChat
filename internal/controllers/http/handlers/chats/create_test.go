package chats_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/chats"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	middlewares "github.com/DKhorkov/libs/middlewares/http"
	"github.com/DKhorkov/libs/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestCreateChatHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания запроса с контекстом
	createRequest := func(t *testing.T, body []byte, userID uint64) *http.Request {
		t.Helper()

		req := httptest.NewRequest(http.MethodPost, "/chats", bytes.NewBuffer(body))
		ctx := contextlib.WithValue(req.Context(), middlewares.UserIDContextKey, userID)

		return req.WithContext(ctx)
	}

	t.Run("successful chat creation - private chat", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockChatsUseCases(ctrl)
		handler := chats.CreateChatHandler(mockUseCase)

		userID := uint64(123)
		now := time.Now()

		// Ожидаемый ответ от usecase
		expectedChat := &domains.Chat{
			ID:          1,
			Title:       pointers.New("Private Chat"),
			Description: nil,
			Type:        domains.ChatTypePrivate,
			CreatedAt:   now,
			UpdatedAt:   now,
			IsRead:      true,
			Members: []domains.User{
				{
					ID:             userID,
					Username:       "testuser",
					Email:          "test@example.com",
					EmailConfirmed: true,
					CreatedAt:      now,
					UpdatedAt:      now,
				},
			},
			Messages: []domains.Message{},
		}

		// Настройка ожиданий мока
		mockUseCase.EXPECT().
			CreateChat(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, chat domains.Chat) (*domains.Chat, error) {
				// Проверяем, что пользователь добавлен в участники
				assert.Equal(t, "Private Chat", *chat.Title)
				assert.Equal(t, domains.ChatTypePrivate, chat.Type)
				assert.Len(t, chat.Members, 1)
				assert.Equal(t, userID, chat.Members[0].ID)

				return expectedChat, nil
			})

		chatRequest := map[string]any{
			"title": "Private Chat",
			"type":  "private",
		}
		body, _ := json.Marshal(chatRequest)

		req := createRequest(t, body, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusCreated, rr.Code)
		assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, float64(1), response["id"])
		assert.Equal(t, "Private Chat", response["title"])
		assert.Equal(t, "private", response["type"])
		assert.NotNil(t, response["createdAt"])
		assert.NotNil(t, response["updatedAt"])
		assert.True(t, response["isRead"].(bool))

		// Проверяем members
		members, ok := response["members"].([]any)
		require.True(t, ok, "members should be an array")
		assert.Len(t, members, 1)
	})

	t.Run("successful chat creation - group chat", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockChatsUseCases(ctrl)
		handler := chats.CreateChatHandler(mockUseCase)

		userID := uint64(123)
		now := time.Now()

		expectedChat := &domains.Chat{
			ID:          2,
			Title:       pointers.New("Group Chat"),
			Description: pointers.New("Test group chat"),
			Type:        domains.ChatTypeGroup,
			CreatedAt:   now,
			UpdatedAt:   now,
			IsRead:      false,
			Members: []domains.User{
				{
					ID:             userID,
					Username:       "testuser",
					EmailConfirmed: true,
				},
				{
					ID:       456,
					Username: "anotheruser",
				},
			},
			Messages: []domains.Message{},
		}

		// Настройка ожиданий мока
		mockUseCase.EXPECT().
			CreateChat(gomock.Any(), gomock.Any()).
			Return(expectedChat, nil)

		chatRequest := map[string]any{
			"title":       "Group Chat",
			"description": "Test group chat",
			"type":        "group",
		}
		body, _ := json.Marshal(chatRequest)

		req := createRequest(t, body, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusCreated, rr.Code)

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Group Chat", response["title"])
		assert.Equal(t, "Test group chat", response["description"])
		assert.Equal(t, "group", response["type"])
		assert.False(t, response["isRead"].(bool))
	})

	t.Run("unauthorized - no userID in context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockChatsUseCases(ctrl)
		handler := chats.CreateChatHandler(mockUseCase)

		chatRequest := map[string]any{
			"title": "Test Chat",
			"type":  "private",
		}
		body, _ := json.Marshal(chatRequest)

		// Запрос БЕЗ userID в контексте (не используем contextlib.WithValue)
		req := httptest.NewRequest(http.MethodPost, "/chats", bytes.NewBuffer(body))
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "context with value userID not found\n")
	})

	t.Run("bad request - invalid JSON", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockChatsUseCases(ctrl)
		handler := chats.CreateChatHandler(mockUseCase)

		// Невалидный JSON
		body := []byte(`{"title": "Test Chat", "type": invalid}`)
		req := createRequest(t, body, 123)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid character")
	})

	t.Run("bad request - empty body", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockChatsUseCases(ctrl)
		handler := chats.CreateChatHandler(mockUseCase)

		// Пустое тело
		req := createRequest(t, []byte{}, 123)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "unexpected end of JSON input\n")
	})

	t.Run("bad request - invalid chat data", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockChatsUseCases(ctrl)
		handler := chats.CreateChatHandler(mockUseCase)

		userID := uint64(123)

		// Настройка мока для возврата ошибки валидации
		mockUseCase.EXPECT().
			CreateChat(gomock.Any(), gomock.Any()).
			Return(nil, customerrors.ErrInvalidChat)

		// Пустое имя чата
		chatRequest := map[string]any{
			"title": "",
			"type":  "private",
		}
		body, _ := json.Marshal(chatRequest)

		req := createRequest(t, body, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidChat.Error())
	})

	t.Run("internal server error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockChatsUseCases(ctrl)
		handler := chats.CreateChatHandler(mockUseCase)

		userID := uint64(123)

		// Настройка мока для возврата общей ошибки
		mockUseCase.EXPECT().
			CreateChat(gomock.Any(), gomock.Any()).
			Return(nil, errors.New("database error"))

		chatRequest := map[string]any{
			"title": "Test Chat",
			"type":  "private",
		}
		body, _ := json.Marshal(chatRequest)

		req := createRequest(t, body, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database error")
	})

	t.Run("chat with members from request", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockChatsUseCases(ctrl)
		handler := chats.CreateChatHandler(mockUseCase)

		userID := uint64(123)
		now := time.Now()

		expectedChat := &domains.Chat{
			ID:        1,
			Title:     pointers.New("Group with Members"),
			Type:      domains.ChatTypeGroup,
			CreatedAt: now,
			UpdatedAt: now,
			IsRead:    true,
			Members: []domains.User{
				{ID: 456, Username: "user1"},
				{ID: 789, Username: "user2"},
				{ID: userID, Username: "currentuser"},
			},
		}

		// Настройка ожиданий мока
		mockUseCase.EXPECT().
			CreateChat(gomock.Any(), gomock.Any()).
			DoAndReturn(func(_ context.Context, chat domains.Chat) (*domains.Chat, error) {
				// Проверяем, что текущий пользователь добавлен к существующим участникам
				assert.Len(t, chat.Members, 3)
				assert.Equal(t, userID, chat.Members[2].ID)
				assert.Equal(t, uint64(456), chat.Members[0].ID)
				assert.Equal(t, uint64(789), chat.Members[1].ID)

				return expectedChat, nil
			})

		// Запрос с участниками
		chatRequest := map[string]any{
			"title": "Group with Members",
			"type":  "group",
			"members": []map[string]any{
				{"id": 456, "username": "user1"},
				{"id": 789, "username": "user2"},
			},
		}
		body, _ := json.Marshal(chatRequest)

		req := createRequest(t, body, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusCreated, rr.Code)

		var response map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.Equal(t, "Group with Members", response["title"])

		members := response["members"].([]any)
		assert.Len(t, members, 3)
	})

	t.Run("missing required fields", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockChatsUseCases(ctrl)
		handler := chats.CreateChatHandler(mockUseCase)

		userID := uint64(123)

		// Настройка мока для возврата ошибки валидации
		mockUseCase.EXPECT().
			CreateChat(gomock.Any(), gomock.Any()).
			Return(nil, customerrors.ErrInvalidChat)

		// Отсутствует обязательное поле type
		chatRequest := map[string]any{
			"title": "Test Chat",
			// type отсутствует
		}
		body, _ := json.Marshal(chatRequest)

		req := createRequest(t, body, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidChat.Error())
	})

	t.Run("invalid chat type", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockChatsUseCases(ctrl)
		handler := chats.CreateChatHandler(mockUseCase)

		userID := uint64(123)

		// Настройка мока для возврата ошибки валидации
		mockUseCase.EXPECT().
			CreateChat(gomock.Any(), gomock.Any()).
			Return(nil, customerrors.ErrInvalidChat)

		// Невалидный тип чата
		chatRequest := map[string]any{
			"title": "Test Chat",
			"type":  "invalid_type",
		}
		body, _ := json.Marshal(chatRequest)

		req := createRequest(t, body, userID)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidChat.Error())
	})
}
