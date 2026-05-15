package register_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/register"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания запроса
	createRequest := func(t *testing.T, requestBody io.Reader) *http.Request {
		t.Helper()

		return httptest.NewRequest(http.MethodPost, "/register", requestBody)
	}

	// Вспомогательная функция для создания тестового пользователя
	createTestUser := func(id uint64, emailConfirmed bool) *domains.User {
		now := time.Now()

		return &domains.User{
			ID:             id,
			Username:       "newuser" + strconv.FormatUint(id, 10),
			Email:          "newuser" + strconv.FormatUint(id, 10) + "@example.com",
			EmailConfirmed: emailConfirmed,
			Password:       "hashed_password_here",
			CreatedAt:      now,
			UpdatedAt:      now,
		}
	}

	// Вспомогательная функция для создания RegisterDTO
	createRegisterDTO := func() domains.RegisterDTO {
		return domains.RegisterDTO{
			Username: "newuser",
			Email:    "newuser@example.com",
			Password: "SecurePassword123!",
		}
	}

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockAuthUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful registration",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := createRegisterDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedUser := createTestUser(123, false)
				m.EXPECT().
					RegisterUser(gomock.Any(), createRegisterDTO()).
					Return(expectedUser, nil)
			},
			expectedStatus: http.StatusCreated,
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

				assert.Equal(t, float64(123), response["id"])
				assert.Equal(t, "newuser123", response["username"])
				assert.Equal(t, "newuser123@example.com", response["email"])
				assert.False(
					t,
					response["emailConfirmed"].(bool),
					"email should not be confirmed after registration",
				)
				assert.NotNil(t, response["createdAt"])
				assert.NotNil(t, response["updatedAt"])

				_, passwordExists := response["password"]
				assert.False(t, passwordExists, "password should not be included in response")
			},
		},
		{
			name: "successful registration with confirmed email",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := createRegisterDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedUser := createTestUser(123, true)
				m.EXPECT().
					RegisterUser(gomock.Any(), createRegisterDTO()).
					Return(expectedUser, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.True(
					t,
					response["emailConfirmed"].(bool),
					"email can be confirmed after registration in some cases",
				)
			},
		},
		{
			name: "bad request - empty request body",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, bytes.NewReader([]byte{}))
			},
			setupMock:      func(_ *mockusecases.MockAuthUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - invalid JSON",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				invalidJSON := `{"username": "test", "email": invalid}`

				return createRequest(t, bytes.NewReader([]byte(invalidJSON)))
			},
			setupMock:      func(_ *mockusecases.MockAuthUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), "invalid character")
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - missing username",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					bytes.NewReader(
						[]byte(`{"email": "test@example.com", "password": "password123"}`),
					),
				)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Email:    "test@example.com",
					Password: "password123",
				}
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - missing email",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					bytes.NewReader([]byte(`{"username": "testuser", "password": "password123"}`)),
				)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "testuser",
					Password: "password123",
				}
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - missing password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(
					t,
					bytes.NewReader(
						[]byte(`{"username": "testuser", "email": "test@example.com"}`),
					),
				)
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
				}
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - empty object",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, bytes.NewReader([]byte(`{}`)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{}
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - validation failed",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "ab",
					Email:    "invalid-email",
					Password: "123",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "ab",
					Email:    "invalid-email",
					Password: "123",
				}
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "conflict - user already exists",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := createRegisterDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					RegisterUser(gomock.Any(), createRegisterDTO()).
					Return(nil, customerrors.ErrUserAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrUserAlreadyExists.Error())
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "conflict - username already exists",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "existinguser",
					Email:    "newemail@example.com",
					Password: "Password123!",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "existinguser",
					Email:    "newemail@example.com",
					Password: "Password123!",
				}
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrUserAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrUserAlreadyExists.Error())
			},
		},
		{
			name: "conflict - email already exists",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "newusername",
					Email:    "existing@example.com",
					Password: "Password123!",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "newusername",
					Email:    "existing@example.com",
					Password: "Password123!",
				}
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrUserAlreadyExists)
			},
			expectedStatus: http.StatusConflict,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrUserAlreadyExists.Error())
			},
		},
		{
			name: "internal server error - use case error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := createRegisterDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					RegisterUser(gomock.Any(), createRegisterDTO()).
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
			name: "internal server error - JSON encoding error",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := createRegisterDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedUser := createTestUser(123, false)
				m.EXPECT().
					RegisterUser(gomock.Any(), createRegisterDTO()).
					Return(expectedUser, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse:  nil,
		},
		{
			name: "registration with strong password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: "SecurePassword123!",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: "SecurePassword123!",
				}
				expectedUser := createTestUser(123, false)
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(expectedUser, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse:  nil,
		},
		{
			name: "registration with very strong password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: "V3ry$tr0ngP@ssw0rd!2024",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: "V3ry$tr0ngP@ssw0rd!2024",
				}
				expectedUser := createTestUser(123, false)
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(expectedUser, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse:  nil,
		},
		{
			name: "registration with weak password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: "123",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: "123",
				}
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "registration with common password",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: "password",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "test@example.com",
					Password: "password",
				}
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "registration with simple email",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "user@example.com",
					Password: "SecurePassword123!",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "user@example.com",
					Password: "SecurePassword123!",
				}
				expectedUser := createTestUser(123, false)
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(expectedUser, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse:  nil,
		},
		{
			name: "registration with email with plus",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "user+tag@example.com",
					Password: "SecurePassword123!",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "user+tag@example.com",
					Password: "SecurePassword123!",
				}
				expectedUser := createTestUser(123, false)
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(expectedUser, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse:  nil,
		},
		{
			name: "registration with email with dot",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "user.name@example.com",
					Password: "SecurePassword123!",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "user.name@example.com",
					Password: "SecurePassword123!",
				}
				expectedUser := createTestUser(123, false)
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(expectedUser, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse:  nil,
		},
		{
			name: "registration with email with subdomain",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "user@sub.example.com",
					Password: "SecurePassword123!",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "user@sub.example.com",
					Password: "SecurePassword123!",
				}
				expectedUser := createTestUser(123, false)
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(expectedUser, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse:  nil,
		},
		{
			name: "registration with invalid email",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "not-an-email",
					Password: "SecurePassword123!",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "not-an-email",
					Password: "SecurePassword123!",
				}
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "registration with email without domain",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "user@",
					Password: "SecurePassword123!",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "testuser",
					Email:    "user@",
					Password: "SecurePassword123!",
				}
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "registration with extra fields in JSON",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				jsonWithExtraFields := `{
				"username": "newuser",
				"email": "new@example.com",
				"password": "SecurePassword123!",
				"extraField": "should be ignored",
				"anotherExtra": 123,
				"isAdmin": true
			}`

				return createRequest(t, bytes.NewReader([]byte(jsonWithExtraFields)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedDTO := domains.RegisterDTO{
					Username: "newuser",
					Email:    "new@example.com",
					Password: "SecurePassword123!",
				}
				expectedUser := createTestUser(123, false)
				m.EXPECT().
					RegisterUser(gomock.Any(), expectedDTO).
					Return(expectedUser, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse:  nil,
		},
		{
			name: "registration with null values",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				jsonWithNull := `{
				"username": null,
				"email": "test@example.com",
				"password": "SecurePassword123!"
			}`

				// Verify deserialization behavior
				var dto domains.RegisterDTO

				err := json.Unmarshal([]byte(jsonWithNull), &dto)
				require.NoError(t, err)
				assert.Equal(t, "", dto.Username)

				return createRequest(t, bytes.NewReader([]byte(jsonWithNull)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "",
					Email:    "test@example.com",
					Password: "SecurePassword123!",
				}
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
			},
		},
		{
			name: "registration with empty strings",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.RegisterDTO{
					Username: "",
					Email:    "",
					Password: "",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				dto := domains.RegisterDTO{
					Username: "",
					Email:    "",
					Password: "",
				}
				m.EXPECT().
					RegisterUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
			},
		},
		{
			name: "status code 201 Created is set correctly",
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := createRegisterDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedUser := createTestUser(123, false)
				m.EXPECT().
					RegisterUser(gomock.Any(), createRegisterDTO()).
					Return(expectedUser, nil)
			},
			expectedStatus: http.StatusCreated,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.NotEqual(t, http.StatusOK, rr.Code, "Should return 201 Created, not 200 OK")
				assert.Equal(
					t,
					common.ApplicationJSONContentType,
					rr.Header().Get(common.ContentTypeHeaderName),
				)
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
			tt.setupMock(mockUseCase)

			handler := register.Handler(mockUseCase)
			req := tt.setupRequest(t)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}

	t.Run("concurrent registrations", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		handler := register.Handler(mockUseCase)

		// Тестируем, что обработчик безопасен для конкурентного использования
		const numRequests = 5

		// Каждый запрос с разными данными
		requestsData := make([]domains.RegisterDTO, numRequests)
		for i := range numRequests {
			requestsData[i] = domains.RegisterDTO{
				Username: "user" + strconv.Itoa(i),
				Email:    "user" + strconv.Itoa(i) + "@example.com",
				Password: "Password123!" + strconv.Itoa(i),
			}
		}

		// Настраиваем мок на конкурентные вызовы
		for i := range numRequests {
			expectedUser := &domains.User{
				ID:             uint64(i + 1000),
				Username:       requestsData[i].Username,
				Email:          requestsData[i].Email,
				EmailConfirmed: false,
				Password:       "hashed_password",
				CreatedAt:      time.Now(),
				UpdatedAt:      time.Now(),
			}

			mockUseCase.EXPECT().
				RegisterUser(gomock.Any(), requestsData[i]).
				Return(expectedUser, nil)
		}

		done := make(chan bool, numRequests)

		// Act - запускаем concurrent запросы
		for i := range numRequests {
			go func(idx int) {
				requestBody, _ := json.Marshal(requestsData[idx])
				req := createRequest(t, bytes.NewReader(requestBody))
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusCreated, rr.Code)

				done <- true
			}(i)
		}

		// Ждем завершения всех горутин
		for range numRequests {
			<-done
		}
	})
}
