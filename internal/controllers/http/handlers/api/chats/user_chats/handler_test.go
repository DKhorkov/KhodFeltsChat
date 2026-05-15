package user_chats_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/chats/user_chats"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/DKhorkov/libs/pointers"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания запроса с контекстом и query параметрами
	createRequest := func(t *testing.T, userID uint64, limit, offset int) *http.Request {
		t.Helper()

		url := "/chats"
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

		return req.WithContext(ctx)
	}

	// Вспомогательная функция для создания тестовых чатов
	createTestChats := func(userID uint64, count int) []domains.Chat {
		now := time.Now()
		chats := make([]domains.Chat, count)

		for i := range count {
			chatID := uint64(i + 1)
			chats[i] = domains.Chat{
				ID:        chatID,
				Title:     pointers.New("Chat " + strconv.Itoa(int(chatID))),
				Type:      domains.ChatTypePrivate,
				CreatedAt: now.Add(time.Duration(i) * time.Hour),
				UpdatedAt: now.Add(time.Duration(i) * time.Hour),
				IsRead:    i%2 == 0,
				Members: []domains.User{
					{
						ID:       userID,
						Username: "user" + strconv.Itoa(int(userID)),
					},
					{
						ID:       userID + 100,
						Username: "other_user",
					},
				},
				Messages: []domains.Message{
					{
						ID:     uint64(i + 1000),
						ChatID: chatID,
						Sender: domains.User{
							ID:       userID,
							Username: "user" + strconv.Itoa(int(userID)),
						},
						Text:      "Hello from chat " + strconv.Itoa(int(chatID)),
						CreatedAt: now.Add(time.Duration(i) * time.Minute),
						UpdatedAt: now.Add(time.Duration(i) * time.Minute),
					},
				},
			}
		}

		return chats
	}

	userID := uint64(123)
	now := time.Now()

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockChatsUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful get user chats - default pagination",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, 0, 0)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				m.EXPECT().
					GetUserChats(gomock.Any(), userID, nil).
					Return(createTestChats(userID, 3), nil)
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

				firstChat := response[0]
				assert.Equal(t, float64(1), firstChat["id"])
				assert.Equal(t, "Chat 1", firstChat["title"])
				assert.Equal(t, "private", firstChat["type"])
				assert.True(t, firstChat["isRead"].(bool))
				assert.NotNil(t, firstChat["createdAt"])
				assert.NotNil(t, firstChat["updatedAt"])

				members, ok := firstChat["members"].([]any)
				require.True(t, ok, "members should be an array")
				assert.Len(t, members, 2)

				messages, ok := firstChat["messages"].([]any)
				require.True(t, ok, "messages should be an array")
				assert.Len(t, messages, 1)
			},
		},
		{
			name: "successful get user chats - custom pagination",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, 5, 10)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				m.EXPECT().
					GetUserChats(gomock.Any(), userID, &domains.Pagination{Limit: pointers.New[uint64](5), Offset: pointers.New[uint64](10)}).
					Return(createTestChats(userID, 2), nil)
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
			name: "successful get user chats - empty result",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, 0, 0)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				m.EXPECT().
					GetUserChats(gomock.Any(), userID, nil).
					Return([]domains.Chat{}, nil)
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

				return httptest.NewRequest(http.MethodGet, "/chats", http.NoBody)
			},
			setupMock:      func(_ *mockusecases.MockChatsUseCases) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "context with value userID not found")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "user not found",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, uint64(999), 0, 0)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				m.EXPECT().
					GetUserChats(gomock.Any(), uint64(999), nil).
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
			name: "internal server error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, 0, 0)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				m.EXPECT().
					GetUserChats(gomock.Any(), userID, nil).
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
			name: "pagination with invalid limit parameter",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				req := httptest.NewRequest(
					http.MethodGet,
					"/chats?limit=invalid&offset=0",
					http.NoBody,
				)
				ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

				return req.WithContext(ctx)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				m.EXPECT().
					GetUserChats(gomock.Any(), userID, nil).
					Return(createTestChats(userID, 2), nil)
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
					"/chats?limit=10&offset=invalid",
					http.NoBody,
				)
				ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

				return req.WithContext(ctx)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				m.EXPECT().
					GetUserChats(gomock.Any(), userID, &domains.Pagination{Limit: pointers.New[uint64](10), Offset: pointers.New[uint64](0)}).
					Return(createTestChats(userID, 2), nil)
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
			name: "get user chats with different chat types",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, 0, 0)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				expectedChats := []domains.Chat{
					{
						ID:        1,
						Title:     pointers.New("Private Chat"),
						Type:      domains.ChatTypePrivate,
						CreatedAt: now,
						UpdatedAt: now,
						IsRead:    true,
						Members:   []domains.User{{ID: userID}},
					},
					{
						ID:          2,
						Title:       pointers.New("Group Chat"),
						Description: pointers.New("Work group"),
						Type:        domains.ChatTypeGroup,
						CreatedAt:   now.Add(time.Hour),
						UpdatedAt:   now.Add(time.Hour),
						IsRead:      false,
						Members: []domains.User{
							{ID: userID},
							{ID: 456},
							{ID: 789},
						},
					},
				}

				m.EXPECT().
					GetUserChats(gomock.Any(), userID, nil).
					Return(expectedChats, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response []map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)
				assert.Len(t, response, 2)

				assert.Equal(t, "private", response[0]["type"])
				assert.Equal(t, "group", response[1]["type"])
				assert.Equal(t, "Work group", response[1]["description"])
			},
		},
		{
			name: "error on JSON encoding failure",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, userID, 0, 0)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				expectedChats := []domains.Chat{
					{
						ID:        1,
						Title:     pointers.New("Valid Chat"),
						Type:      domains.ChatTypePrivate,
						CreatedAt: now,
						UpdatedAt: now,
						IsRead:    true,
						Members:   []domains.User{{ID: userID}},
					},
				}

				m.EXPECT().
					GetUserChats(gomock.Any(), userID, nil).
					Return(expectedChats, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockChatsUseCases(ctrl)
			tt.setupMock(mockUseCase)

			handler := user_chats.Handler(mockUseCase)
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
