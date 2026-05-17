package update_test

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

	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/users/update"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/common"
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

	// Вспомогательная функция для создания запроса
	createRequest := func(t *testing.T, userID uint64, requestBody io.Reader) *http.Request {
		t.Helper()

		req := httptest.NewRequest(http.MethodPatch, "/me", requestBody)
		ctx := contextlib.WithValue(req.Context(), authmiddleware.UserIDContextKey, userID)

		return req.WithContext(ctx)
	}

	// Вспомогательная функция для создания тестового пользователя
	createTestUser := func(id uint64, emailConfirmed bool) *domains.User {
		now := time.Now()

		return &domains.User{
			ID:             id,
			Username:       "updateduser" + strconv.FormatUint(id, 10),
			Email:          "updated" + strconv.FormatUint(id, 10) + "@example.com",
			EmailConfirmed: emailConfirmed,
			Password:       "hashed_password_here",
			CreatedAt:      now.Add(-24 * time.Hour),
			UpdatedAt:      now,
		}
	}

	// Вспомогательная функция для создания DTO
	createUpdateDTO := func() domains.UpdateUserDTO {
		return domains.UpdateUserDTO{
			Username: pointers.New("newusername"),
		}
	}

	tests := []struct {
		name           string
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockUsersUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name: "successful update current user",
			setupRequest: func(t *testing.T) *http.Request {
				dto := createUpdateDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				dto := createUpdateDTO()
				dto.ID = 123
				m.EXPECT().
					UpdateUser(gomock.Any(), dto).
					Return(createTestUser(123, true), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(
					t,
					common.ApplicationJSONContentType,
					rr.Header().Get(common.ContentTypeHeaderName),
				)

				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, float64(123), response["id"])
				assert.Equal(t, "updateduser123", response["username"])
				assert.Equal(t, "updated123@example.com", response["email"])
				assert.True(t, response["emailConfirmed"].(bool))
				assert.NotNil(t, response["createdAt"])
				assert.NotNil(t, response["updatedAt"])

				_, passwordExists := response["password"]
				assert.False(t, passwordExists, "password should not be included in response")
			},
		},
		{
			name: "successful partial update",
			setupRequest: func(t *testing.T) *http.Request {
				dto := domains.UpdateUserDTO{
					Username: pointers.New("onlyusernameupdated"),
				}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				expectedDTO := domains.UpdateUserDTO{
					ID:       123,
					Username: pointers.New("onlyusernameupdated"),
				}
				m.EXPECT().
					UpdateUser(gomock.Any(), expectedDTO).
					Return(createTestUser(123, true), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse:  nil,
		},
		{
			name: "unauthorized - no userID in context",
			setupRequest: func(t *testing.T) *http.Request {
				dto := createUpdateDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return httptest.NewRequest(http.MethodPatch, "/me", bytes.NewReader(requestBody))
			},
			setupMock:      func(_ *mockusecases.MockUsersUseCases) {},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - empty request body",
			setupRequest: func(t *testing.T) *http.Request {
				return createRequest(t, 123, bytes.NewReader([]byte{}))
			},
			setupMock:      func(_ *mockusecases.MockUsersUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - invalid JSON",
			setupRequest: func(t *testing.T) *http.Request {
				invalidJSON := `{"username": "test", "email": invalid}`

				return createRequest(t, 123, bytes.NewReader([]byte(invalidJSON)))
			},
			setupMock:      func(_ *mockusecases.MockUsersUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "bad request - validation failed",
			setupRequest: func(t *testing.T) *http.Request {
				dto := domains.UpdateUserDTO{
					Username: pointers.New("ab"),
				}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				expectedDTO := domains.UpdateUserDTO{
					ID:       123,
					Username: pointers.New("ab"),
				}
				m.EXPECT().
					UpdateUser(gomock.Any(), expectedDTO).
					Return(nil, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "not found - user not found",
			setupRequest: func(t *testing.T) *http.Request {
				dto := createUpdateDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, 999, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				dto := createUpdateDTO()
				dto.ID = 999
				m.EXPECT().
					UpdateUser(gomock.Any(), dto).
					Return(nil, customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "internal server error - use case error",
			setupRequest: func(t *testing.T) *http.Request {
				dto := createUpdateDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				dto := createUpdateDTO()
				dto.ID = 123
				m.EXPECT().
					UpdateUser(gomock.Any(), dto).
					Return(nil, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
			},
		},
		{
			name: "internal server error - JSON encoding error",
			setupRequest: func(t *testing.T) *http.Request {
				dto := createUpdateDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				dto := createUpdateDTO()
				dto.ID = 123
				m.EXPECT().
					UpdateUser(gomock.Any(), dto).
					Return(createTestUser(123, true), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse:  nil,
		},
		{
			name: "update with empty DTO",
			setupRequest: func(t *testing.T) *http.Request {
				dto := domains.UpdateUserDTO{}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				expectedDTO := domains.UpdateUserDTO{ID: 123}
				m.EXPECT().
					UpdateUser(gomock.Any(), expectedDTO).
					Return(&domains.User{
						ID:             123,
						Username:       "originaluser",
						Email:          "original@example.com",
						EmailConfirmed: true,
						Password:       "hashed_password",
						CreatedAt:      time.Now().Add(-24 * time.Hour),
						UpdatedAt:      time.Now().Add(-24 * time.Hour),
					}, nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, "originaluser", response["username"])
				assert.Equal(t, "original@example.com", response["email"])
			},
		},
		{
			name: "update with extra fields in JSON",
			setupRequest: func(t *testing.T) *http.Request {
				jsonWithExtraFields := `{
				"username": "newuser",
				"email": "new@example.com",
				"extraField": "should be ignored",
				"anotherExtra": 123
			}`

				return createRequest(t, 123, bytes.NewReader([]byte(jsonWithExtraFields)))
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				expectedDTO := domains.UpdateUserDTO{
					ID:       123,
					Username: pointers.New("newuser"),
				}
				m.EXPECT().
					UpdateUser(gomock.Any(), expectedDTO).
					Return(createTestUser(123, true), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse:  nil,
		},
		{
			name: "request body too large",
			setupRequest: func(t *testing.T) *http.Request {
				hugeBody := make([]byte, 10*1024*1024)
				for i := range hugeBody {
					hugeBody[i] = 'a'
				}

				hugeJSON := []byte(`{"username":"` + string(hugeBody) + `"}`)
				req := createRequest(t, 123, bytes.NewReader(hugeJSON))
				req.Body = http.MaxBytesReader(nil, req.Body, 1024)

				return req
			},
			setupMock:      func(_ *mockusecases.MockUsersUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse:  nil,
		},
		{
			name: "DTO ID is overwritten with userID from context",
			setupRequest: func(t *testing.T) *http.Request {
				dto := domains.UpdateUserDTO{
					ID:       999,
					Username: pointers.New("newusername"),
				}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, 123, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockUsersUseCases) {
				expectedDTO := domains.UpdateUserDTO{
					ID:       123,
					Username: pointers.New("newusername"),
				}
				m.EXPECT().
					UpdateUser(gomock.Any(), expectedDTO).
					Return(createTestUser(123, true), nil)
			},
			expectedStatus: http.StatusOK,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				var response map[string]any

				err := json.Unmarshal(rr.Body.Bytes(), &response)
				require.NoError(t, err)

				assert.Equal(t, float64(123), response["id"])
				assert.NotEqual(t, float64(999), response["id"])
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
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

	t.Run("concurrent updates", func(t *testing.T) {
		t.Parallel()

		mockUseCase := mockusecases.NewMockUsersUseCases(ctrl)
		handler := update.Handler(mockUseCase)

		const numRequests = 5

		userID := uint64(123)

		requestsData := make([]domains.UpdateUserDTO, numRequests)
		for i := range numRequests {
			requestsData[i] = domains.UpdateUserDTO{
				Username: pointers.New("user" + strconv.Itoa(i)),
			}
		}

		for i := range numRequests {
			expectedDTO := requestsData[i]
			expectedDTO.ID = userID

			expectedUser := &domains.User{
				ID:             userID,
				Username:       *expectedDTO.Username,
				EmailConfirmed: true,
				Password:       "hashed_password",
				CreatedAt:      time.Now().Add(-24 * time.Hour),
				UpdatedAt:      time.Now(),
			}

			mockUseCase.EXPECT().
				UpdateUser(gomock.Any(), expectedDTO).
				Return(expectedUser, nil)
		}

		done := make(chan bool, numRequests)

		for i := range numRequests {
			go func(idx int) {
				requestBody, _ := json.Marshal(requestsData[idx])
				req := createRequest(t, userID, bytes.NewReader(requestBody))
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusOK, rr.Code)

				done <- true
			}(i)
		}

		for range numRequests {
			<-done
		}
	})
}
