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

	userID := uint64(123)
	chatID := uint64(456)

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockMessagesUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful get chat messages - default pagination",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, chatID, 0, 0)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetChatMessages(gomock.Any(), userID, chatID, nil).
					Return(createTestMessages(chatID, userID, 3), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(
					t,
					common.ApplicationJSONContentType,
					rr.Header().Get(common.ContentTypeHeaderName),
				)

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 3)

				firstMessage := response[0]
				assert.Equal(t, float64(1), firstMessage["id"])
				assert.Equal(t, float64(chatID), firstMessage["chatId"])
				assert.Equal(t, "Message 1", firstMessage["text"])
				assert.NotNil(t, firstMessage["createdAt"])
				assert.NotNil(t, firstMessage["updatedAt"])

				sender, ok := firstMessage["sender"].(map[string]any)
				require.True(t, ok, "sender should be an object")
				assert.Equal(t, float64(userID+100), sender["id"])
				assert.Equal(t, "user"+strconv.Itoa(int(userID+100)), sender["username"])
			},
		},
		{
			name: "successful get chat messages - custom pagination",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, chatID, 20, 10)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetChatMessages(
						gomock.Any(),
						userID,
						chatID,
						&domains.Pagination{
							Limit:  pointers.New[uint64](20),
							Offset: pointers.New[uint64](10),
						},
					).
					Return(createTestMessages(chatID, userID, 2), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 2)
			},
		},
		{
			name: "successful get chat messages - empty result",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, chatID, 0, 0)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetChatMessages(gomock.Any(), userID, chatID, nil).
					Return([]domains.Message{}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Empty(t, response)
			},
		},
		{
			name: "unauthorized - no userID in context",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return httptest.NewRequest(http.MethodGet, "/chats/123/messages", http.NoBody)
			},
			setupMock:      func(_ *mockusecases.MockMessagesUseCases) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "context with value userID not found")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - invalid chat ID parameter",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				req := httptest.NewRequest(http.MethodGet, "/chats/invalid/messages", http.NoBody)
				ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)
				vars := map[string]string{
					common.IDRouteKey: "invalid",
				}
				req = mux.SetURLVars(req.WithContext(ctx), vars)

				return req
			},
			setupMock:      func(_ *mockusecases.MockMessagesUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "invalid syntax")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - empty chat ID parameter",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				req := httptest.NewRequest(http.MethodGet, "/chats//messages", http.NoBody)
				ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)
				vars := map[string]string{
					common.IDRouteKey: "",
				}
				req = mux.SetURLVars(req.WithContext(ctx), vars)

				return req
			},
			setupMock:      func(_ *mockusecases.MockMessagesUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "invalid syntax")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "forbidden - user is not chat member",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, chatID, 0, 0)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetChatMessages(gomock.Any(), userID, chatID, nil).
					Return(nil, customerrors.ErrUserIsNotChatMember)
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrUserIsNotChatMember.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "not found - chat not found",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, uint64(999), 0, 0)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetChatMessages(gomock.Any(), userID, uint64(999), nil).
					Return(nil, customerrors.ErrChatNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrChatNotFound.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "not found - user not found",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, uint64(999), chatID, 0, 0)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetChatMessages(gomock.Any(), uint64(999), chatID, nil).
					Return(nil, customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "internal server error - use case error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, chatID, 0, 0)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetChatMessages(gomock.Any(), userID, chatID, nil).
					Return(nil, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "database connection failed")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "internal server error - JSON encoding error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, chatID, 0, 0)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetChatMessages(gomock.Any(), userID, chatID, nil).
					Return(createTestMessages(chatID, userID, 1), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse:  nil,
		},
		{
			name: "pagination with invalid limit parameter",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

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

				return req
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetChatMessages(gomock.Any(), userID, chatID, nil).
					Return(createTestMessages(chatID, userID, 2), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 2)
			},
		},
		{
			name: "pagination with invalid offset parameter",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

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

				return req
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetChatMessages(
						gomock.Any(),
						userID,
						chatID,
						&domains.Pagination{
							Limit:  pointers.New[uint64](10),
							Offset: pointers.New[uint64](0),
						},
					).
					Return(createTestMessages(chatID, userID, 2), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 2)
			},
		},
		{
			name: "zero chat ID",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, uint64(0), 0, 0)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetChatMessages(gomock.Any(), userID, uint64(0), nil).
					Return([]domains.Message{}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Empty(t, response)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
			tt.setupMock(mockUseCase)

			handler := chat_messages.Handler(mockUseCase)
			req := tt.setupRequest(t)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}
}
