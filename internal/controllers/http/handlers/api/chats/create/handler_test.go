package create_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/chats/create"
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

	// Вспомогательная функция для создания запроса с контекстом
	createRequest := func(t *testing.T, body []byte, userID uint64) *http.Request {
		t.Helper()

		req := httptest.NewRequest(http.MethodPost, "/chats", bytes.NewBuffer(body))
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

		return req.WithContext(ctx)
	}

	now := time.Now()

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockChatsUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful chat creation - private chat",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				chatRequest := map[string]any{
					"title": "Private Chat",
					"type":  "private",
				}
				body, _ := json.Marshal(chatRequest)

				return createRequest(t, body, 123)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				expectedChat := &domains.Chat{
					ID:          1,
					Title:       pointers.New("Private Chat"),
					Description: nil,
					Type:        domains.ChatTypePrivate,
					CreatedAt:   now,
					UpdatedAt:   now,
					Members: []domains.User{
						{
							ID:             123,
							Username:       "testuser",
							Email:          "test@example.com",
							EmailConfirmed: true,
							CreatedAt:      now,
							UpdatedAt:      now,
						},
					},
					Messages: []domains.Message{},
				}

				m.EXPECT().
					CreateChat(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, chat domains.Chat) (*domains.Chat, error) {
						assert.Equal(t, "Private Chat", *chat.Title)
						assert.Equal(t, domains.ChatTypePrivate, chat.Type)
						assert.Len(t, chat.Members, 1)
						assert.Equal(t, uint64(123), chat.Members[0].ID)

						return expectedChat, nil
					})
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, float64(1), response["id"])
				assert.Equal(t, "Private Chat", response["title"])
				assert.Equal(t, "private", response["type"])
				assert.NotNil(t, response["createdAt"])
				assert.NotNil(t, response["updatedAt"])

				members, ok := response["members"].([]any)
				require.True(t, ok, "members should be an array")
				assert.Len(t, members, 1)
			},
		},
		{
			name: "successful chat creation - group chat",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				chatRequest := map[string]any{
					"title":       "Group Chat",
					"description": "Test group chat",
					"type":        "group",
				}
				body, _ := json.Marshal(chatRequest)

				return createRequest(t, body, 123)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				expectedChat := &domains.Chat{
					ID:          2,
					Title:       pointers.New("Group Chat"),
					Description: pointers.New("Test group chat"),
					Type:        domains.ChatTypeGroup,
					CreatedAt:   now,
					UpdatedAt:   now,
					Members: []domains.User{
						{
							ID:             123,
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

				m.EXPECT().
					CreateChat(gomock.Any(), gomock.Any()).
					Return(expectedChat, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, "Group Chat", response["title"])
				assert.Equal(t, "Test group chat", response["description"])
				assert.Equal(t, "group", response["type"])
			},
		},
		{
			name: "unauthorized - no userID in context",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				chatRequest := map[string]any{
					"title": "Test Chat",
					"type":  "private",
				}
				body, _ := json.Marshal(chatRequest)

				return httptest.NewRequest(http.MethodPost, "/chats", bytes.NewBuffer(body))
			},
			setupMock:      func(_ *mockusecases.MockChatsUseCases) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "context with value userID not found\n")
			},
		},
		{
			name: "bad request - invalid JSON",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				body := []byte(`{"title": "Test Chat", "type": invalid}`)

				return createRequest(t, body, 123)
			},
			setupMock:      func(_ *mockusecases.MockChatsUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "invalid character")
			},
		},
		{
			name: "bad request - empty body",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, []byte{}, 123)
			},
			setupMock:      func(_ *mockusecases.MockChatsUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "unexpected end of JSON input\n")
			},
		},
		{
			name: "bad request - invalid chat data",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				chatRequest := map[string]any{
					"title": "",
					"type":  "private",
				}
				body, _ := json.Marshal(chatRequest)

				return createRequest(t, body, 123)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				m.EXPECT().
					CreateChat(gomock.Any(), gomock.Any()).
					Return(nil, customerrors.ErrInvalidChat)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidChat.Error())
			},
		},
		{
			name: "internal server error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				chatRequest := map[string]any{
					"title": "Test Chat",
					"type":  "private",
				}
				body, _ := json.Marshal(chatRequest)

				return createRequest(t, body, 123)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				m.EXPECT().
					CreateChat(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "database error")
			},
		},
		{
			name: "chat with members from request",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				chatRequest := map[string]any{
					"title": "Group with Members",
					"type":  "group",
					"members": []map[string]any{
						{"id": 456, "username": "user1"},
						{"id": 789, "username": "user2"},
					},
				}
				body, _ := json.Marshal(chatRequest)

				return createRequest(t, body, 123)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				expectedChat := &domains.Chat{
					ID:        1,
					Title:     pointers.New("Group with Members"),
					Type:      domains.ChatTypeGroup,
					CreatedAt: now,
					UpdatedAt: now,
					Members: []domains.User{
						{ID: 456, Username: "user1"},
						{ID: 789, Username: "user2"},
						{ID: 123, Username: "currentuser"},
					},
				}

				m.EXPECT().
					CreateChat(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, chat domains.Chat) (*domains.Chat, error) {
						assert.Len(t, chat.Members, 3)
						assert.Equal(t, uint64(123), chat.Members[2].ID)
						assert.Equal(t, uint64(456), chat.Members[0].ID)
						assert.Equal(t, uint64(789), chat.Members[1].ID)

						return expectedChat, nil
					})
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, "Group with Members", response["title"])

				members := response["members"].([]any)
				assert.Len(t, members, 3)
			},
		},
		{
			name: "missing required fields",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				chatRequest := map[string]any{
					"title": "Test Chat",
				}
				body, _ := json.Marshal(chatRequest)

				return createRequest(t, body, 123)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				m.EXPECT().
					CreateChat(gomock.Any(), gomock.Any()).
					Return(nil, customerrors.ErrInvalidChat)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidChat.Error())
			},
		},
		{
			name: "invalid chat type",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				chatRequest := map[string]any{
					"title": "Test Chat",
					"type":  "invalid_type",
				}
				body, _ := json.Marshal(chatRequest)

				return createRequest(t, body, 123)
			},
			setupMock: func(m *mockusecases.MockChatsUseCases) {
				m.EXPECT().
					CreateChat(gomock.Any(), gomock.Any()).
					Return(nil, customerrors.ErrInvalidChat)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidChat.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockChatsUseCases(ctrl)
			tt.setupMock(mockUseCase)

			handler := create.Handler(mockUseCase)
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
