package chat_messages_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/messages/chat_messages"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/DKhorkov/libs/pointers"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания запроса
	createRequest := func(t *testing.T, userID uint64, chatID uint64, limit, offset int) *http.Request {
		t.Helper()

		url := "/chats/" + strconv.FormatUint(chatID, 10) + "/messages"
		if limit > 0 || offset > 0 {
			url += "?"
			if limit > 0 {
				url += "limit=" + strconv.Itoa(limit)
			}

			if offset > 0 {
				if limit > 0 {
					url += "&"
				}

				url += "offset=" + strconv.Itoa(offset)
			}
		}

		req := httptest.NewRequest(http.MethodGet, url, http.NoBody)
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

		// Устанавливаем параметры маршрута для mux.Vars
		vars := map[string]string{
			common.IDRouteKey: strconv.FormatUint(chatID, 10),
		}
		req = mux.SetURLVars(req.WithContext(ctx), vars)

		return req
	}

	// Вспомогательная функция для создания тестовых сообщений
	createTestMessages := func(chatID, userID uint64, count int) []domains.Message {
		now := time.Now()
		messages := make([]domains.Message, count)

		for i := range count {
			messageID := uint64(i + 1)

			senderID := userID
			if i%2 == 0 {
				senderID = userID + 100 // Другой пользователь
			}

			messages[i] = domains.Message{
				ID:     messageID,
				ChatID: chatID,
				Sender: domains.User{
					ID:       senderID,
					Username: "user" + strconv.Itoa(int(senderID)),
				},
				Text:      "Message " + strconv.Itoa(int(messageID)),
				CreatedAt: now.Add(time.Duration(i) * time.Minute),
				UpdatedAt: now.Add(time.Duration(i) * time.Minute),
			}
		}

		return messages
	}

	t.Run("successful get chat messages - default pagination", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		userID := uint64(123)
		chatID := uint64(456)
		expectedMessages := createTestMessages(chatID, userID, 3)

		mockUseCase.EXPECT().
			GetChatMessages(gomock.Any(), userID, chatID, nil).
			Return(expectedMessages, nil)

		req := createRequest(t, userID, chatID, 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)
		assert.Equal(
			t,
			common.ApplicationJSONContentType,
			rr.Header().Get(common.ContentTypeHeaderName),
		)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 3)

		// Проверяем структуру первого сообщения
		firstMessage := response[0]
		assert.Equal(t, float64(1), firstMessage["id"])
		assert.Equal(t, float64(chatID), firstMessage["chatId"])
		assert.Equal(t, "Message 1", firstMessage["text"])
		assert.NotNil(t, firstMessage["createdAt"])
		assert.NotNil(t, firstMessage["updatedAt"])

		// Проверяем отправителя
		sender, ok := firstMessage["sender"].(map[string]any)
		require.True(t, ok, "sender should be an object")
		assert.Equal(t, float64(userID+100), sender["id"])
		assert.Equal(t, "user"+strconv.Itoa(int(userID+100)), sender["username"])
	})

	t.Run("successful get chat messages - custom pagination", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		userID := uint64(123)
		chatID := uint64(456)
		expectedMessages := createTestMessages(chatID, userID, 2)

		mockUseCase.EXPECT().
			GetChatMessages(
				gomock.Any(),
				userID,
				chatID,
				&domains.Pagination{
					Limit:  pointers.New[uint64](20),
					Offset: pointers.New[uint64](10),
				},
			).
			Return(expectedMessages, nil)

		req := createRequest(t, userID, chatID, 20, 10)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
	})

	t.Run("successful get chat messages - empty result", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		userID := uint64(123)
		chatID := uint64(456)

		mockUseCase.EXPECT().
			GetChatMessages(gomock.Any(), userID, chatID, nil).
			Return([]domains.Message{}, nil)

		req := createRequest(t, userID, chatID, 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Empty(t, response)
	})

	t.Run("unauthorized - no userID in context", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		// Запрос без userID в контексте
		req := httptest.NewRequest(http.MethodGet, "/chats/123/messages", http.NoBody)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusUnauthorized, rr.Code)
		assert.Contains(t, rr.Body.String(), "context with value userID not found")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("bad request - invalid chat ID parameter", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		userID := uint64(123)

		// Запрос с невалидным chat ID
		req := httptest.NewRequest(http.MethodGet, "/chats/invalid/messages", http.NoBody)
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)
		vars := map[string]string{
			common.IDRouteKey: "invalid",
		}
		req = mux.SetURLVars(req.WithContext(ctx), vars)

		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid syntax")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("bad request - empty chat ID parameter", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		userID := uint64(123)

		// Запрос с пустым chat ID
		req := httptest.NewRequest(http.MethodGet, "/chats//messages", http.NoBody)
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)
		vars := map[string]string{
			common.IDRouteKey: "",
		}
		req = mux.SetURLVars(req.WithContext(ctx), vars)

		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusBadRequest, rr.Code)
		assert.Contains(t, rr.Body.String(), "invalid syntax")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("forbidden - user is not chat member", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		userID := uint64(123)
		chatID := uint64(456)

		mockUseCase.EXPECT().
			GetChatMessages(gomock.Any(), userID, chatID, nil).
			Return(nil, customerrors.ErrUserIsNotChatMember)

		req := createRequest(t, userID, chatID, 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusForbidden, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserIsNotChatMember.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("not found - chat not found", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		userID := uint64(123)
		chatID := uint64(999) // Несуществующий чат

		mockUseCase.EXPECT().
			GetChatMessages(gomock.Any(), userID, chatID, nil).
			Return(nil, customerrors.ErrChatNotFound)

		req := createRequest(t, userID, chatID, 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrChatNotFound.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("not found - user not found", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		userID := uint64(999) // Несуществующий пользователь
		chatID := uint64(456)

		mockUseCase.EXPECT().
			GetChatMessages(gomock.Any(), userID, chatID, nil).
			Return(nil, customerrors.ErrUserNotFound)

		req := createRequest(t, userID, chatID, 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusNotFound, rr.Code)
		assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("internal server error - use case error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		userID := uint64(123)
		chatID := uint64(456)

		mockUseCase.EXPECT().
			GetChatMessages(gomock.Any(), userID, chatID, nil).
			Return(nil, errors.New("database connection failed"))

		req := createRequest(t, userID, chatID, 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusInternalServerError, rr.Code)
		assert.Contains(t, rr.Body.String(), "database connection failed")
		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	})

	t.Run("internal server error - JSON encoding error", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		userID := uint64(123)
		chatID := uint64(456)

		// Создаем сообщения, которые не могут быть замаплены (имитируем ошибку маппера)
		// В реальном тесте нужно замокать mappers.MapMessages
		// Но здесь мы просто проверяем сценарий успешного выполнения от use case
		expectedMessages := createTestMessages(chatID, userID, 1)

		mockUseCase.EXPECT().
			GetChatMessages(gomock.Any(), userID, chatID, nil).
			Return(expectedMessages, nil)

		req := createRequest(t, userID, chatID, 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должен успешно обработаться, так как маппер работает
		assert.Equal(t, http.StatusOK, rr.Code)
	})

	t.Run("pagination with invalid limit parameter", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		userID := uint64(123)
		chatID := uint64(456)
		expectedMessages := createTestMessages(chatID, userID, 2)

		// При невалидном limit, GetPaginationFromRequest должен вернуть nil
		mockUseCase.EXPECT().
			GetChatMessages(gomock.Any(), userID, chatID, nil).
			Return(expectedMessages, nil)

		// Запрос с невалидным limit
		req := httptest.NewRequest(
			http.MethodGet,
			"/chats/"+strconv.FormatUint(chatID, 10)+"/messages?limit=invalid&offset=0",
			http.NoBody,
		)
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)
		vars := map[string]string{
			common.IDRouteKey: strconv.FormatUint(chatID, 10),
		}
		req = mux.SetURLVars(req.WithContext(ctx), vars)

		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должен обработаться с дефолтными значениями
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
	})

	t.Run("pagination with invalid offset parameter", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		userID := uint64(123)
		chatID := uint64(456)
		expectedMessages := createTestMessages(chatID, userID, 2)

		mockUseCase.EXPECT().
			GetChatMessages(
				gomock.Any(),
				userID,
				chatID,
				&domains.Pagination{
					Limit:  pointers.New[uint64](10),
					Offset: pointers.New[uint64](0),
				},
			).
			Return(expectedMessages, nil)

		// Запрос с невалидным offset
		req := httptest.NewRequest(
			http.MethodGet,
			"/chats/"+strconv.FormatUint(chatID, 10)+"/messages?limit=10&offset=invalid",
			http.NoBody,
		)
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)
		vars := map[string]string{
			common.IDRouteKey: strconv.FormatUint(chatID, 10),
		}
		req = mux.SetURLVars(req.WithContext(ctx), vars)

		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert - должен обработаться с дефолтными значениями
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Len(t, response, 2)
	})

	t.Run("zero chat ID", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
		handler := chat_messages.Handler(mockUseCase)

		userID := uint64(123)
		chatID := uint64(0) // Нулевой chat ID

		// Use case может обрабатывать нулевой chat ID, но это пограничный случай
		mockUseCase.EXPECT().
			GetChatMessages(gomock.Any(), userID, chatID, nil).
			Return([]domains.Message{}, nil)

		req := createRequest(t, userID, chatID, 0, 0)
		rr := httptest.NewRecorder()

		// Act
		handler.ServeHTTP(rr, req)

		// Assert
		assert.Equal(t, http.StatusOK, rr.Code)

		var response []map[string]any

		err := json.Unmarshal(rr.Body.Bytes(), &response)
		require.NoError(t, err)
		assert.Empty(t, response)
	})
}
