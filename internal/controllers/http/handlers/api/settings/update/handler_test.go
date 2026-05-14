package update_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/settings/update"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	createRequest := func(t *testing.T, userID uint64, body []byte) *http.Request {
		t.Helper()

		req := httptest.NewRequest(http.MethodPut, "/api/users/me/settings", bytes.NewReader(body))
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

		return req.WithContext(ctx)
	}

	now := time.Now()

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockSettingsUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful update settings to dark theme",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				body, _ := json.Marshal(map[string]any{"theme": 1})

				return createRequest(t, 123, body)
			},
			setupMock: func(m *mockusecases.MockSettingsUseCases) {
				m.EXPECT().
					UpdateSettings(gomock.Any(), domains.Settings{UserID: 123, Theme: domains.ThemeDark}).
					Return(&domains.Settings{
						ID:        1,
						UserID:    123,
						Theme:     domains.ThemeDark,
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(
					t,
					common.ApplicationJSONContentType,
					rr.Header().Get(common.ContentTypeHeaderName),
				)

				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, float64(1), response["theme"])
			},
		},
		{
			name: "successful update settings to light theme",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				body, _ := json.Marshal(map[string]any{"theme": 0})

				return createRequest(t, 456, body)
			},
			setupMock: func(m *mockusecases.MockSettingsUseCases) {
				m.EXPECT().
					UpdateSettings(gomock.Any(), domains.Settings{UserID: 456, Theme: domains.ThemeLight}).
					Return(&domains.Settings{
						ID:        2,
						UserID:    456,
						Theme:     domains.ThemeLight,
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, float64(0), response["theme"])
			},
		},
		{
			name: "unauthorized - no userID in context",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				body, _ := json.Marshal(map[string]any{"theme": 1})

				return httptest.NewRequest(http.MethodPut, "/api/users/me/settings", bytes.NewReader(body))
			},
			setupMock:      func(_ *mockusecases.MockSettingsUseCases) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "bad request - invalid JSON",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, 123, []byte("invalid json"))
			},
			setupMock:      func(_ *mockusecases.MockSettingsUseCases) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "settings not found",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				body, _ := json.Marshal(map[string]any{"theme": 1})

				return createRequest(t, 999, body)
			},
			setupMock: func(m *mockusecases.MockSettingsUseCases) {
				m.EXPECT().
					UpdateSettings(gomock.Any(), gomock.Any()).
					Return(nil, customerrors.ErrSettingsNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "internal server error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				body, _ := json.Marshal(map[string]any{"theme": 1})

				return createRequest(t, 123, body)
			},
			setupMock: func(m *mockusecases.MockSettingsUseCases) {
				m.EXPECT().
					UpdateSettings(gomock.Any(), gomock.Any()).
					Return(nil, errors.New("database error"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), "database error")
			},
		},
		{
			name: "userID from context is used, not from body",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				body, _ := json.Marshal(map[string]any{"theme": 1, "userID": 999})

				return createRequest(t, 123, body)
			},
			setupMock: func(m *mockusecases.MockSettingsUseCases) {
				m.EXPECT().
					UpdateSettings(gomock.Any(), domains.Settings{UserID: 123, Theme: domains.ThemeDark}).
					Return(&domains.Settings{
						ID:        1,
						UserID:    123,
						Theme:     domains.ThemeDark,
						CreatedAt: now,
						UpdatedAt: now,
					}, nil)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockSettingsUseCases(ctrl)
			tt.setupMock(mockUseCase)

			handler := update.Handler(mockUseCase)
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
