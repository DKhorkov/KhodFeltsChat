package update_avatar_test

import (
	"bytes"
	"errors"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/update_avatar"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func createMultipartRequest(t *testing.T) *http.Request {
	t.Helper()

	var buf bytes.Buffer

	writer := multipart.NewWriter(&buf)

	part, err := writer.CreateFormFile("avatar", "test.jpg")
	require.NoError(t, err)

	_, err = part.Write([]byte("fake image data"))
	require.NoError(t, err)

	require.NoError(t, writer.Close())

	req := httptest.NewRequest(http.MethodPut, "/api/users/me/avatar", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())

	ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, uint64(1))

	return req.WithContext(ctx)
}

func TestHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockUsersUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:         "successful avatar upload",
			setupRequest: createMultipartRequest,
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					UpdateAvatar(gomock.Any(), uint64(1), gomock.Any()).
					Return("http://localhost:8080/api/files/download/uuid.jpg", nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "text/plain", rr.Header().Get("Content-Type"))
				assert.Contains(t, rr.Body.String(), "uuid.jpg")
			},
		},
		{
			name: "unauthorized - no user id in context",
			setupRequest: func(t *testing.T) *http.Request {
				return httptest.NewRequest(http.MethodPut, "/api/users/me/avatar", http.NoBody)
			},
			setupMock:      func(m *mockusecases.MockUsersUseCases) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "missing avatar file",
			setupRequest: func(t *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodPut, "/api/users/me/avatar", http.NoBody)
				req.Header.Set("Content-Type", "multipart/form-data; boundary=xxx")
				ctx := contextlib.WithValue(
					req.Context(),
					authmiddleware.UserIDContextKey,
					uint64(1),
				)

				return req.WithContext(ctx)
			},
			setupMock:      func(m *mockusecases.MockUsersUseCases) {},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "invalid image format",
			setupRequest: createMultipartRequest,
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					UpdateAvatar(gomock.Any(), uint64(1), gomock.Any()).
					Return("", customerrors.ErrInvalidImageFormat)
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:         "file too large",
			setupRequest: createMultipartRequest,
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					UpdateAvatar(gomock.Any(), uint64(1), gomock.Any()).
					Return("", customerrors.ErrFileTooLarge)
			},
			expectedStatus: http.StatusRequestEntityTooLarge,
		},
		{
			name:         "internal server error",
			setupRequest: createMultipartRequest,
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					UpdateAvatar(gomock.Any(), uint64(1), gomock.Any()).
					Return("", errors.New("something went wrong"))
			},
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockUC := mockusecases.NewMockUsersUseCases(ctrl)

			tt.setupMock(mockUC)

			handler := update_avatar.Handler(mockUC)
			rr := httptest.NewRecorder()
			req := tt.setupRequest(t)

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}
}
