package forget_password_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/forget_password"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/gorilla/mux"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

const (
	validCode uint64 = 123456
	newPass          = "NewSecurePassword123!"
)

func createForgetPasswordRequest(t *testing.T, rawCode string, body io.Reader) *http.Request {
	t.Helper()

	url := "/forget-password/" + rawCode
	req := httptest.NewRequest(http.MethodPost, url, body)
	req = mux.SetURLVars(req, map[string]string{forget_password.TokenRouteKey: rawCode})

	return req
}

func marshalDTO(t *testing.T, dto domains.ForgetPasswordDTO) io.Reader {
	t.Helper()

	data, err := json.Marshal(dto)
	require.NoError(t, err)

	return bytes.NewReader(data)
}

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	tests := []struct {
		name           string
		rawCode        string
		body           func(t *testing.T) io.Reader
		setupMock      func(m *mockusecases.MockAuthUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:    "successful password reset",
			rawCode: strconv.FormatUint(validCode, 10),
			body: func(t *testing.T) io.Reader {
				return marshalDTO(t, domains.ForgetPasswordDTO{NewPassword: newPass})
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), validCode, newPass).
					Return(nil)
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
			body: func(t *testing.T) io.Reader {
				return marshalDTO(t, domains.ForgetPasswordDTO{NewPassword: newPass})
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrInvalidJWT.Error())
			},
		},
		{
			name:    "bad request - malformed JSON body",
			rawCode: strconv.FormatUint(validCode, 10),
			body: func(t *testing.T) io.Reader {
				return bytes.NewReader([]byte("{not json"))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:    "validation failed",
			rawCode: strconv.FormatUint(validCode, 10),
			body: func(t *testing.T) io.Reader {
				return marshalDTO(t, domains.ForgetPasswordDTO{NewPassword: "weak"})
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), validCode, "weak").
					Return(customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
			},
		},
		{
			name:    "unauthorized - use case returns invalid JWT",
			rawCode: "654321",
			body: func(t *testing.T) io.Reader {
				return marshalDTO(t, domains.ForgetPasswordDTO{NewPassword: newPass})
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), uint64(654321), newPass).
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
			body: func(t *testing.T) io.Reader {
				return marshalDTO(t, domains.ForgetPasswordDTO{NewPassword: newPass})
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), uint64(222222), newPass).
					Return(customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()
				assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
			},
		},
		{
			name:    "internal server error",
			rawCode: "333333",
			body: func(t *testing.T) io.Reader {
				return marshalDTO(t, domains.ForgetPasswordDTO{NewPassword: newPass})
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					ForgetPassword(gomock.Any(), uint64(333333), newPass).
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

			handler := forget_password.Handler(mockUseCase)
			req := createForgetPasswordRequest(t, tt.rawCode, tt.body(t))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}
}
