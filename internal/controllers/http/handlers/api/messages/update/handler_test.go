package update_message_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	updatehandler "github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/messages/update"
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
	dto domains.UpdateMessageDTO,
) *http.Request {
	t.Helper()

	body, _ := json.Marshal(dto)
	req := httptest.NewRequest(http.MethodPut, "/api/messages/"+messageID, bytes.NewReader(body))

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
	chatID := uint64(100)

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockMessagesUseCases, b *mockcontrollers.MockWSBroadcaster)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful update (204)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					&userID,
					messageIDStr,
					domains.UpdateMessageDTO{Text: "new text"},
				)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases, b *mockcontrollers.MockWSBroadcaster) {
				m.EXPECT().
					UpdateMessage(gomock.Any(), domains.UpdateMessageDTO{
						MessageID: messageID,
						UserID:    userID,
						Text:      "new text",
					}).
					Return(&domains.Message{
						ID:     messageID,
						ChatID: chatID,
						Sender: domains.User{ID: userID},
					}, nil)

				b.EXPECT().
					BroadcastMessageEdited(gomock.Any(), chatID, messageID)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "UpdateMessage returns ErrMessageNotFound (404)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					&userID,
					messageIDStr,
					domains.UpdateMessageDTO{Text: "new text"},
				)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {
				m.EXPECT().
					UpdateMessage(gomock.Any(), domains.UpdateMessageDTO{
						MessageID: messageID,
						UserID:    userID,
						Text:      "new text",
					}).
					Return(nil, customerrors.ErrMessageNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrMessageNotFound.Error())
			},
		},
		{
			name: "UpdateMessage returns ErrNotMessageAuthor (403)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					&userID,
					messageIDStr,
					domains.UpdateMessageDTO{Text: "new text"},
				)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {
				m.EXPECT().
					UpdateMessage(gomock.Any(), domains.UpdateMessageDTO{
						MessageID: messageID,
						UserID:    userID,
						Text:      "new text",
					}).
					Return(nil, customerrors.ErrNotMessageAuthor)
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrNotMessageAuthor.Error())
			},
		},
		{
			name: "UpdateMessage returns generic error (500)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					&userID,
					messageIDStr,
					domains.UpdateMessageDTO{Text: "new text"},
				)
			},
			setupMock: func(m *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {
				m.EXPECT().
					UpdateMessage(gomock.Any(), domains.UpdateMessageDTO{
						MessageID: messageID,
						UserID:    userID,
						Text:      "new text",
					}).
					Return(nil, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "database connection failed")
			},
		},
		{
			name: "missing/invalid message ID (400)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					&userID,
					"invalid",
					domains.UpdateMessageDTO{Text: "new text"},
				)
			},
			setupMock:      func(_ *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "invalid syntax")
			},
		},
		{
			name: "invalid JSON body (400)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				req := httptest.NewRequest(
					http.MethodPut,
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
			name: "empty text (400)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					&userID,
					messageIDStr,
					domains.UpdateMessageDTO{Text: ""},
				)
			},
			setupMock:      func(_ *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), "text is required")
			},
		},
		{
			name: "unauthorized - no userID in context (401)",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					nil,
					messageIDStr,
					domains.UpdateMessageDTO{Text: "new text"},
				)
			},
			setupMock:      func(_ *mockusecases.MockMessagesUseCases, _ *mockcontrollers.MockWSBroadcaster) {},
			expectedStatus: http.StatusUnauthorized,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockUseCase := mockusecases.NewMockMessagesUseCases(ctrl)
			mockBroadcaster := mockcontrollers.NewMockWSBroadcaster(ctrl)

			tt.setupMock(mockUseCase, mockBroadcaster)

			handler := updatehandler.Handler(mockUseCase, mockBroadcaster)
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
