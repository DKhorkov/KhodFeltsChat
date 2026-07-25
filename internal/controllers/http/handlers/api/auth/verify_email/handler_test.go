package verify_email_test

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/verify_email"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

const (
	validCode uint64 = 123456
)

func createRequest(t *testing.T, rawCode string) *http.Request {
	t.Helper()

	url := "/verify-email/" + rawCode
	req := httptest.NewRequest(http.MethodGet, url, http.NoBody)
	req = mux.SetURLVars(req, map[string]string{verify_email.TokenRouteKey: rawCode})

	return req
}

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	tests := []struct {
		name           string
		rawCode        string
		setupMock      func(m *mockusecases.MockAuthUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:    "successful email verification",
			rawCode: strconv.FormatUint(validCode, 10),
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().VerifyEmail(gomock.Any(), validCode).Return(nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Empty(t, rr.Body.String())
			},
		},
		{
			name:    "unauthorized - non-numeric code",
			rawCode: "not-a-number",
			setupMock: func(m *mockusecases.MockAuthUseCases) {
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
			},
		},
		{
			name:    "unauthorized - use case returns invalid JWT",
			rawCode: "654321",
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), uint64(654321)).
					Return(customerrors.ErrInvalidJWT)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
			},
		},
		{
			name:    "not found - user not found",
			rawCode: "222222",
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), uint64(222222)).
					Return(customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
			},
		},
		{
			name:    "conflict - email already confirmed",
			rawCode: "333333",
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), uint64(333333)).
					Return(customerrors.ErrEmailAlreadyConfirmed)
			},
			expectedStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrEmailAlreadyConfirmed.Error())
			},
		},
		{
			name:    "internal server error",
			rawCode: "444444",
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					VerifyEmail(gomock.Any(), uint64(444444)).
					Return(errors.New("database connection failed"))
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

			mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
			tt.setupMock(mockUseCase)

			handler := verify_email.Handler(mockUseCase)
			req := createRequest(t, tt.rawCode)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}
}
