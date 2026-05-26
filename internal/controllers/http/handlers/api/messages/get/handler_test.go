package get_message_test

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	get_message "github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/messages/get"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func createRequest(
	t *testing.T,
	userID *uint64,
	messageID string,
) *http.Request {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, "/api/messages/"+messageID, http.NoBody)

	if userID != nil {
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, *userID)
		req = req.WithContext(ctx)
	}

	vars := map[string]string{common.IDRouteKey: messageID}
	req = mux.SetURLVars(req, vars)

	return req
}

func TestHandler(t *testing.T) {
	t.Parallel()

	userID := uint64(123)
	messageID := uint64(456)
	messageIDStr := strconv.FormatUint(messageID, 10)
	now := time.Now()

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockMessagesUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful get message",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, &userID, messageIDStr)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetMessageByID(gomock.Any(), userID, messageID).
					Return(&domains.Message{
						ID:     messageID,
						ChatID: 100,
						Sender: domains.User{
							ID:       userID,
							Username: "testuser",
						},
						Text:      "hello world",
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, "application/json", rr.Header().Get("Content-Type"))

				var resp map[string]any

				err := json.NewDecoder(rr.Body).Decode(&resp)
				require.NoError(t, err)

				assert.Equal(t, float64(messageID), resp["id"])
				assert.Equal(t, float64(100), resp["chatId"])
				assert.Equal(t, "hello world", resp["text"])

				sender, ok := resp["sender"].(map[string]any)
				require.True(t, ok)
				assert.Equal(t, float64(userID), sender["id"])
				assert.Equal(t, "testuser", sender["username"])
			},
		},
		{
			name: "unauthorized - no userID in context",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, nil, messageIDStr)
			},
			setupMock:      func(_ *mockusecases.MockMessagesUseCases) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid messageID (not a number)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, &userID, "invalid")
			},
			setupMock:      func(_ *mockusecases.MockMessagesUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "invalid syntax")
			},
		},
		{
			name: "empty messageID",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, &userID, "")
			},
			setupMock:      func(_ *mockusecases.MockMessagesUseCases) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "message not found",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, &userID, messageIDStr)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetMessageByID(gomock.Any(), userID, messageID).
					Return(nil, customerrors.ErrMessageNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrMessageNotFound.Error())
			},
		},
		{
			name: "internal server error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, &userID, messageIDStr)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases) {
				m.EXPECT().
					GetMessageByID(gomock.Any(), userID, messageID).
					Return(nil, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "database connection failed")
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)

			tt.setupMock(mockUseCase)

			handler := get_message.Handler(mockUseCase)
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
