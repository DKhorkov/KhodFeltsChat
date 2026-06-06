package delete_avatar_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/delete_avatar"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/contextlib"
	authmiddleware "github.com/DKhorkov/libs/middlewares/http/auth"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockUsersUseCases)
		expectedStatus int
	}{
		{
			name: "successful avatar delete",
			setupRequest: func(t *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodDelete, "/api/users/me/avatar", http.NoBody)
				ctx := contextlib.WithValue(
					req.Context(),
					authmiddleware.UserIDContextKey,
					uint64(1),
				)

				return req.WithContext(ctx)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					DeleteAvatar(gomock.Any(), uint64(1)).
					Return(nil)
			},
			expectedStatus: http.StatusNoContent,
		},
		{
			name: "unauthorized - no user id in context",
			setupRequest: func(t *testing.T) *http.Request {
				return httptest.NewRequest(http.MethodDelete, "/api/users/me/avatar", http.NoBody)
			},
			setupMock:      func(m *mockusecases.MockUsersUseCases) {},
			expectedStatus: http.StatusUnauthorized,
		},
		{
			name: "user not found",
			setupRequest: func(t *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodDelete, "/api/users/me/avatar", http.NoBody)
				ctx := contextlib.WithValue(
					req.Context(),
					authmiddleware.UserIDContextKey,
					uint64(999),
				)

				return req.WithContext(ctx)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					DeleteAvatar(gomock.Any(), uint64(999)).
					Return(customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name: "internal server error",
			setupRequest: func(t *testing.T) *http.Request {
				req := httptest.NewRequest(http.MethodDelete, "/api/users/me/avatar", http.NoBody)
				ctx := contextlib.WithValue(
					req.Context(),
					authmiddleware.UserIDContextKey,
					uint64(1),
				)

				return req.WithContext(ctx)
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				m.EXPECT().
					DeleteAvatar(gomock.Any(), uint64(1)).
					Return(errors.New("something went wrong"))
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

			handler := delete_avatar.Handler(mockUC)
			rr := httptest.NewRecorder()
			req := tt.setupRequest(t)

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)
		})
	}
}
