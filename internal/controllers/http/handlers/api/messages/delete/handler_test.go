package delete_message_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	deletehandler "github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/messages/delete"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockcontrollers "github.com/DKhorkov/kfc/mocks/controllers"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func createRequest(
	t *testing.T,
	userID *uint64,
	messageID string,
	dto domains.DeleteMessageDTO,
) *http.Request {
	t.Helper()

	body, _ := json.Marshal(dto)
	req := httptest.NewRequest(http.MethodDelete, "/api/messages/"+messageID, bytes.NewReader(body))

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

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockMessagesUseCases, b *mockcontrollers.MockWSBroadcaster)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful delete for self (no broadcast)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					&userID,
					messageIDStr,
					domains.DeleteMessageDTO{ForAll: false},
				)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {
				m.EXPECT().
					DeleteMessage(gomock.Any(), domains.DeleteMessageDTO{
						MessageID: messageID,
						UserID:    userID,
						ForAll:    false,
					}).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "successful delete for all (with broadcast)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					&userID,
					messageIDStr,
					domains.DeleteMessageDTO{ForAll: true},
				)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases, b *mockcontrollers.MockWSBroadcaster) {
				m.EXPECT().
					GetMessageByID(gomock.Any(), userID, messageID).
					Return(&domains.Message{
						ID:     messageID,
						ChatID: 100,
						Sender: domains.User{ID: userID},
					}, nil)

				m.EXPECT().
					DeleteMessage(gomock.Any(), domains.DeleteMessageDTO{
						MessageID: messageID,
						UserID:    userID,
						ForAll:    true,
					}).
					Return(nil)

				b.EXPECT().
					BroadcastMessageDeleted(gomock.Any(), uint64(100), messageID)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "unauthorized - no userID in context",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, nil, messageIDStr, domains.DeleteMessageDTO{ForAll: false})
			},
			setupMock:      func(_ *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "invalid messageID (not a number)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, &userID, "invalid", domains.DeleteMessageDTO{ForAll: false})
			},
			setupMock:      func(_ *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "invalid syntax")
			},
		},
		{
			name: "invalid body JSON",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				req := httptest.NewRequest(
					http.MethodDelete,
					"/api/messages/"+messageIDStr,
					bytes.NewReader([]byte("invalid json")),
				)
				ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)
				req = req.WithContext(ctx)
				vars := map[string]string{common.IDRouteKey: messageIDStr}
				req = mux.SetURLVars(req, vars)

				return req
			},
			setupMock:      func(_ *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "message not found on delete",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					&userID,
					messageIDStr,
					domains.DeleteMessageDTO{ForAll: false},
				)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {
				m.EXPECT().
					DeleteMessage(gomock.Any(), domains.DeleteMessageDTO{
						MessageID: messageID,
						UserID:    userID,
						ForAll:    false,
					}).
					Return(customerrors.ErrMessageNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrMessageNotFound.Error())
			},
		},
		{
			name: "not message author",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					&userID,
					messageIDStr,
					domains.DeleteMessageDTO{ForAll: true},
				)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {
				m.EXPECT().
					GetMessageByID(gomock.Any(), userID, messageID).
					Return(&domains.Message{
						ID:     messageID,
						ChatID: 100,
						Sender: domains.User{ID: userID},
					}, nil)

				m.EXPECT().
					DeleteMessage(gomock.Any(), domains.DeleteMessageDTO{
						MessageID: messageID,
						UserID:    userID,
						ForAll:    true,
					}).
					Return(customerrors.ErrNotMessageAuthor)
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrNotMessageAuthor.Error())
			},
		},
		{
			name: "internal server error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					&userID,
					messageIDStr,
					domains.DeleteMessageDTO{ForAll: false},
				)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {
				m.EXPECT().
					DeleteMessage(gomock.Any(), domains.DeleteMessageDTO{
						MessageID: messageID,
						UserID:    userID,
						ForAll:    false,
					}).
					Return(errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "database connection failed")
			},
		},
		{
			name: "GetMessageByID error for ForAll",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					&userID,
					messageIDStr,
					domains.DeleteMessageDTO{ForAll: true},
				)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {
				m.EXPECT().
					GetMessageByID(gomock.Any(), userID, messageID).
					Return(nil, errors.New("message not found"))
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrMessageNotFound.Error())
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
			mockBroadcaster := mockcontrollers.NewMockWSBroadcaster(ctrl)

			tt.setupMock(mockUseCase, mockBroadcaster)

			handler := deletehandler.Handler(mockUseCase, mockBroadcaster)
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
