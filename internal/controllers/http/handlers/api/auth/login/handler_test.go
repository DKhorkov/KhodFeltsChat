package login_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/controllers/http/handlers/api/auth/login"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/cookies"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestHandler(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)

	// Вспомогательная функция для создания конфига кук
	createCookiesConfig := func() config.CookiesConfig {
		return config.CookiesConfig{
			AccessToken: cookies.Config{
				Path:     "/",
				Domain:   "example.com",
				MaxAge:   900, // 15 минут
				Secure:   true,
				HTTPOnly: true,
				SameSite: http.SameSiteStrictMode,
			},
			RefreshToken: cookies.Config{
				Path:     "/",
				Domain:   "example.com",
				MaxAge:   604800, // 7 дней
				Secure:   true,
				HTTPOnly: true,
				SameSite: http.SameSiteStrictMode,
			},
		}
	}

	// Вспомогательная функция для создания запроса
	createRequest := func(t *testing.T, requestBody io.Reader) *http.Request {
		t.Helper()

		return httptest.NewRequest(http.MethodPost, "/login", requestBody)
	}

	// Вспомогательная функция для создания LoginDTO
	createLoginDTO := func() domains.LoginDTO {
		return domains.LoginDTO{
			Login:    "test@example.com",
			Password: "SecurePassword123!",
		}
	}

	// Вспомогательная функция для создания тестовых токенов
	createTestTokens := func() domains.TokensDTO {
		return domains.TokensDTO{
			AccessToken:  "access_token_123",
			RefreshToken: "refresh_token_456",
		}
	}

	// Вспомогательная функция для проверки, что куки не установлены
	checkNoCookies := func(t *testing.T, rr *httptest.ResponseRecorder) {
		t.Helper()

		c := rr.Result().Cookies()
		assert.Empty(t, c)
	}

	// Вспомогательная функция для проверки Content-Type при ошибках
	checkErrorContentType := func(t *testing.T, rr *httptest.ResponseRecorder) {
		t.Helper()

		assert.Equal(t, "text/plain; charset=utf-8", rr.Header().Get("Content-Type"))
	}

	// Вспомогательная функция для проверки кук при успешном логине
	checkSuccessfulCookies := func(
		expectedTokens domains.TokensDTO,
		cookiesCfg config.CookiesConfig,
	) func(t *testing.T, rr *httptest.ResponseRecorder) {
		return func(t *testing.T, rr *httptest.ResponseRecorder) {
			t.Helper()

			assert.Empty(t, rr.Body.String())

			c := rr.Result().Cookies()
			assert.Len(t, c, 2)

			var (
				accessTokenCookie  *http.Cookie
				refreshTokenCookie *http.Cookie
			)

			for _, cookie := range c {
				switch cookie.Name {
				case login.AccessTokenCookieName:
					accessTokenCookie = cookie
				case login.RefreshTokenCookieName:
					refreshTokenCookie = cookie
				}
			}

			require.NotNil(t, accessTokenCookie)
			require.NotNil(t, refreshTokenCookie)

			// Проверяем значения токенов
			assert.Equal(t, expectedTokens.AccessToken, accessTokenCookie.Value)
			assert.Equal(t, expectedTokens.RefreshToken, refreshTokenCookie.Value)

			// Проверяем настройки access token cookie
			assert.Equal(t, cookiesCfg.AccessToken.Path, accessTokenCookie.Path)
			assert.Equal(t, cookiesCfg.AccessToken.Domain, accessTokenCookie.Domain)
			assert.Equal(t, cookiesCfg.AccessToken.MaxAge, accessTokenCookie.MaxAge)
			assert.Equal(t, cookiesCfg.AccessToken.Secure, accessTokenCookie.Secure)
			assert.Equal(t, cookiesCfg.AccessToken.HTTPOnly, accessTokenCookie.HttpOnly)
			assert.Equal(t, int(cookiesCfg.AccessToken.SameSite), int(accessTokenCookie.SameSite))

			// Проверяем настройки refresh token cookie
			assert.Equal(t, cookiesCfg.RefreshToken.Path, refreshTokenCookie.Path)
			assert.Equal(t, cookiesCfg.RefreshToken.Domain, refreshTokenCookie.Domain)
			assert.Equal(t, cookiesCfg.RefreshToken.MaxAge, refreshTokenCookie.MaxAge)
			assert.Equal(t, cookiesCfg.RefreshToken.Secure, refreshTokenCookie.Secure)
			assert.Equal(t, cookiesCfg.RefreshToken.HTTPOnly, refreshTokenCookie.HttpOnly)
			assert.Equal(t, int(cookiesCfg.RefreshToken.SameSite), int(refreshTokenCookie.SameSite))
		}
	}

	defaultCookiesConfig := createCookiesConfig()

	tests := []struct {
		name           string
		cookiesConfig  config.CookiesConfig
		setupRequest   func(t *testing.T) *http.Request
		setupMock      func(m *mockusecases.MockAuthUseCases)
		expectedStatus int
		checkResponse  func(t *testing.T, rr *httptest.ResponseRecorder)
	}{
		{
			name:          "successful login with email",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := createLoginDTO()
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					LoginUser(gomock.Any(), createLoginDTO()).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  checkSuccessfulCookies(createTestTokens(), defaultCookiesConfig),
		},
		{
			name:          "bad request - empty request body",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, bytes.NewReader([]byte{}))
			},
			setupMock:      func(_ *mockusecases.MockAuthUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				checkErrorContentType(t, rr)
				checkNoCookies(t, rr)
			},
		},
		{
			name:          "bad request - invalid JSON",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				invalidJSON := `{"login": "test@example.com", "password": invalid}`

				return createRequest(t, bytes.NewReader([]byte(invalidJSON)))
			},
			setupMock:      func(_ *mockusecases.MockAuthUseCases) {},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), "invalid character")
				checkErrorContentType(t, rr)
				checkNoCookies(t, rr)
			},
		},
		{
			name:          "bad request - missing login",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, bytes.NewReader([]byte(`{"password": "password123"}`)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{Password: "password123"}).
					Return(&domains.TokensDTO{}, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				checkErrorContentType(t, rr)
				checkNoCookies(t, rr)
			},
		},
		{
			name:          "bad request - missing password",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, bytes.NewReader([]byte(`{"login": "test@example.com"}`)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{Login: "test@example.com"}).
					Return(&domains.TokensDTO{}, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				checkErrorContentType(t, rr)
				checkNoCookies(t, rr)
			},
		},
		{
			name:          "bad request - empty object",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				return createRequest(t, bytes.NewReader([]byte(`{}`)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{}).
					Return(&domains.TokensDTO{}, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				checkErrorContentType(t, rr)
				checkNoCookies(t, rr)
			},
		},
		{
			name:          "bad request - validation failed",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.LoginDTO{
					Login:    "invalid-email",
					Password: "123",
				}

				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{
						Login:    "invalid-email",
						Password: "123",
					}).
					Return(&domains.TokensDTO{}, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
				checkErrorContentType(t, rr)
				checkNoCookies(t, rr)
			},
		},
		{
			name:          "not found - user not found",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				requestBody, err := json.Marshal(createLoginDTO())
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LoginUser(gomock.Any(), createLoginDTO()).
					Return(&domains.TokensDTO{}, customerrors.ErrUserNotFound)
			},
			expectedStatus: http.StatusNotFound,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrUserNotFound.Error())
				checkErrorContentType(t, rr)
				checkNoCookies(t, rr)
			},
		},
		{
			name:          "forbidden - email not confirmed",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				requestBody, err := json.Marshal(createLoginDTO())
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LoginUser(gomock.Any(), createLoginDTO()).
					Return(&domains.TokensDTO{}, customerrors.ErrEmailNotConfirmed)
			},
			expectedStatus: http.StatusForbidden,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrEmailNotConfirmed.Error())
				checkErrorContentType(t, rr)
				checkNoCookies(t, rr)
			},
		},
		{
			name:          "unauthorized - wrong password",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				requestBody, err := json.Marshal(createLoginDTO())
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LoginUser(gomock.Any(), createLoginDTO()).
					Return(&domains.TokensDTO{}, customerrors.ErrWrongPassword)
			},
			expectedStatus: http.StatusUnauthorized,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrWrongPassword.Error())
				checkErrorContentType(t, rr)
				checkNoCookies(t, rr)
			},
		},
		{
			name:          "internal server error - use case error",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				requestBody, err := json.Marshal(createLoginDTO())
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LoginUser(gomock.Any(), createLoginDTO()).
					Return(&domains.TokensDTO{}, errors.New("database connection failed"))
			},
			expectedStatus: http.StatusInternalServerError,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), "database connection failed")
				checkErrorContentType(t, rr)
				checkNoCookies(t, rr)
			},
		},
		{
			name: "login with secure cookie configuration",
			cookiesConfig: config.CookiesConfig{
				AccessToken: cookies.Config{
					Path:     "/api",
					Domain:   "secure.example.com",
					MaxAge:   300,
					Secure:   true,
					HTTPOnly: true,
					SameSite: http.SameSiteStrictMode,
				},
				RefreshToken: cookies.Config{
					Path:     "/api",
					Domain:   "secure.example.com",
					MaxAge:   2592000, // 30 дней
					Secure:   true,
					HTTPOnly: true,
					SameSite: http.SameSiteStrictMode,
				},
			},
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				requestBody, err := json.Marshal(createLoginDTO())
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					LoginUser(gomock.Any(), createLoginDTO()).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: checkSuccessfulCookies(createTestTokens(), config.CookiesConfig{
				AccessToken: cookies.Config{
					Path:     "/api",
					Domain:   "secure.example.com",
					MaxAge:   300,
					Secure:   true,
					HTTPOnly: true,
					SameSite: http.SameSiteStrictMode,
				},
				RefreshToken: cookies.Config{
					Path:     "/api",
					Domain:   "secure.example.com",
					MaxAge:   2592000,
					Secure:   true,
					HTTPOnly: true,
					SameSite: http.SameSiteStrictMode,
				},
			}),
		},
		{
			name: "login with development cookie configuration (non-secure)",
			cookiesConfig: config.CookiesConfig{
				AccessToken: cookies.Config{
					Path:     "/",
					Domain:   "localhost",
					MaxAge:   900,
					Secure:   false,
					HTTPOnly: true,
					SameSite: http.SameSiteLaxMode,
				},
				RefreshToken: cookies.Config{
					Path:     "/",
					Domain:   "localhost",
					MaxAge:   604800,
					Secure:   false,
					HTTPOnly: true,
					SameSite: http.SameSiteLaxMode,
				},
			},
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				requestBody, err := json.Marshal(createLoginDTO())
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					LoginUser(gomock.Any(), createLoginDTO()).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: checkSuccessfulCookies(createTestTokens(), config.CookiesConfig{
				AccessToken: cookies.Config{
					Path:     "/",
					Domain:   "localhost",
					MaxAge:   900,
					Secure:   false,
					HTTPOnly: true,
					SameSite: http.SameSiteLaxMode,
				},
				RefreshToken: cookies.Config{
					Path:     "/",
					Domain:   "localhost",
					MaxAge:   604800,
					Secure:   false,
					HTTPOnly: true,
					SameSite: http.SameSiteLaxMode,
				},
			}),
		},
		{
			name: "login with session cookie configuration (no MaxAge)",
			cookiesConfig: config.CookiesConfig{
				AccessToken: cookies.Config{
					Path:     "/",
					Domain:   "example.com",
					MaxAge:   0, // Session cookie
					Secure:   true,
					HTTPOnly: true,
					SameSite: http.SameSiteNoneMode,
				},
				RefreshToken: cookies.Config{
					Path:     "/",
					Domain:   "example.com",
					MaxAge:   0, // Session cookie
					Secure:   true,
					HTTPOnly: true,
					SameSite: http.SameSiteNoneMode,
				},
			},
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				requestBody, err := json.Marshal(createLoginDTO())
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					LoginUser(gomock.Any(), createLoginDTO()).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: checkSuccessfulCookies(createTestTokens(), config.CookiesConfig{
				AccessToken: cookies.Config{
					Path:     "/",
					Domain:   "example.com",
					MaxAge:   0,
					Secure:   true,
					HTTPOnly: true,
					SameSite: http.SameSiteNoneMode,
				},
				RefreshToken: cookies.Config{
					Path:     "/",
					Domain:   "example.com",
					MaxAge:   0,
					Secure:   true,
					HTTPOnly: true,
					SameSite: http.SameSiteNoneMode,
				},
			}),
		},
		{
			name:          "login with simple email format",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.LoginDTO{Login: "user@example.com", Password: "SecurePassword123!"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{Login: "user@example.com", Password: "SecurePassword123!"}).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name:          "login with email containing plus",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.LoginDTO{
					Login:    "user+tag@example.com",
					Password: "SecurePassword123!",
				}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{Login: "user+tag@example.com", Password: "SecurePassword123!"}).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name:          "login with username format",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.LoginDTO{Login: "testuser", Password: "SecurePassword123!"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{Login: "testuser", Password: "SecurePassword123!"}).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name:          "login with extra fields in JSON",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				jsonWithExtraFields := `{
				"login": "test@example.com",
				"password": "SecurePassword123!",
				"extraField": "should be ignored",
				"anotherExtra": 123,
				"rememberMe": true
			}`

				return createRequest(t, bytes.NewReader([]byte(jsonWithExtraFields)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{
						Login:    "test@example.com",
						Password: "SecurePassword123!",
					}).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name:          "login with null values",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				jsonWithNull := `{
				"login": null,
				"password": "SecurePassword123!"
			}`

				return createRequest(t, bytes.NewReader([]byte(jsonWithNull)))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{
						Login:    "",
						Password: "SecurePassword123!",
					}).
					Return(&domains.TokensDTO{}, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
			},
		},
		{
			name:          "login with empty strings",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.LoginDTO{Login: "", Password: ""}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{Login: "", Password: ""}).
					Return(&domains.TokensDTO{}, customerrors.ErrValidationFailed)
			},
			expectedStatus: http.StatusBadRequest,
			checkResponse: func(t *testing.T, rr *httptest.ResponseRecorder) {
				t.Helper()

				assert.Contains(t, rr.Body.String(), customerrors.ErrValidationFailed.Error())
			},
		},
		{
			name:          "empty tokens from use case",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				requestBody, err := json.Marshal(createLoginDTO())
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := domains.TokensDTO{AccessToken: "", RefreshToken: ""}
				m.EXPECT().
					LoginUser(gomock.Any(), createLoginDTO()).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse: checkSuccessfulCookies(
				domains.TokensDTO{AccessToken: "", RefreshToken: ""},
				defaultCookiesConfig,
			),
		},
		{
			name:          "login with lowercase email (case-sensitive)",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.LoginDTO{Login: "user@example.com", Password: "SecurePassword123!"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{Login: "user@example.com", Password: "SecurePassword123!"}).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name:          "login with uppercase email (case-sensitive)",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.LoginDTO{Login: "USER@EXAMPLE.COM", Password: "SecurePassword123!"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{Login: "USER@EXAMPLE.COM", Password: "SecurePassword123!"}).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name:          "login with mixed case email (case-sensitive)",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.LoginDTO{Login: "UsEr@ExAmPlE.CoM", Password: "SecurePassword123!"}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{Login: "UsEr@ExAmPlE.CoM", Password: "SecurePassword123!"}).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
		{
			name:          "login with email containing special characters",
			cookiesConfig: defaultCookiesConfig,
			setupRequest: func(t *testing.T) *http.Request {
				t.Helper()

				dto := domains.LoginDTO{
					Login:    "user.name+tag@example.com",
					Password: "SecurePassword123!",
				}
				requestBody, err := json.Marshal(dto)
				require.NoError(t, err)

				return createRequest(t, bytes.NewReader(requestBody))
			},
			setupMock: func(m *mockusecases.MockAuthUseCases) {
				expectedTokens := createTestTokens()
				m.EXPECT().
					LoginUser(gomock.Any(), domains.LoginDTO{Login: "user.name+tag@example.com", Password: "SecurePassword123!"}).
					Return(&expectedTokens, nil)
			},
			expectedStatus: http.StatusNoContent,
			checkResponse:  nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
			tt.setupMock(mockUseCase)
			handler := login.Handler(mockUseCase, tt.cookiesConfig)
			req := tt.setupRequest(t)
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, tt.expectedStatus, rr.Code)

			if tt.checkResponse != nil {
				tt.checkResponse(t, rr)
			}
		})
	}

	// Тесты, которые не вписываются в табличный формат

	t.Run("concurrent login requests", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		const numRequests = 5

		// Каждый запрос с разными данными
		requestsData := make([]domains.LoginDTO, numRequests)
		for i := range numRequests {
			requestsData[i] = domains.LoginDTO{
				Login:    "user" + strconv.Itoa(i) + "@example.com",
				Password: "Password123!" + strconv.Itoa(i),
			}
		}

		// Настраиваем мок на конкурентные вызовы
		for i := range numRequests {
			expectedTokens := domains.TokensDTO{
				AccessToken:  "access_" + strconv.Itoa(i),
				RefreshToken: "refresh_" + strconv.Itoa(i),
			}

			mockUseCase.EXPECT().
				LoginUser(gomock.Any(), requestsData[i]).
				Return(&expectedTokens, nil)
		}

		done := make(chan bool, numRequests)

		// Act - запускаем concurrent запросы
		for i := range numRequests {
			go func(idx int) {
				requestBody, _ := json.Marshal(requestsData[idx])
				req := createRequest(t, bytes.NewReader(requestBody))
				rr := httptest.NewRecorder()
				handler.ServeHTTP(rr, req)

				assert.Equal(t, http.StatusNoContent, rr.Code)

				// Проверяем, что установлены правильные куки
				c := rr.Result().Cookies()
				assert.Len(t, c, 2)

				var (
					accessTokenCookie  *http.Cookie
					refreshTokenCookie *http.Cookie
				)

				for _, cookie := range c {
					switch cookie.Name {
					case login.AccessTokenCookieName:
						accessTokenCookie = cookie
					case login.RefreshTokenCookieName:
						refreshTokenCookie = cookie
					}
				}

				require.NotNil(t, accessTokenCookie)
				require.NotNil(t, refreshTokenCookie)
				assert.Equal(t, "access_"+strconv.Itoa(idx), accessTokenCookie.Value)
				assert.Equal(t, "refresh_"+strconv.Itoa(idx), refreshTokenCookie.Value)

				done <- true
			}(i)
		}

		// Ждем завершения всех горутин
		for range numRequests {
			<-done
		}
	})

	t.Run("rate limiting scenarios", func(t *testing.T) {
		t.Parallel()

		// Arrange
		mockUseCase := mockusecases.NewMockAuthUseCases(ctrl)
		cookiesConfig := createCookiesConfig()
		handler := login.Handler(mockUseCase, cookiesConfig)

		// Многократные попытки входа с неверным паролем
		dto := domains.LoginDTO{
			Login:    "test@example.com",
			Password: "WrongPassword123!",
		}

		requestBody, err := json.Marshal(dto)
		require.NoError(t, err)

		// Несколько попыток с неверным паролем
		const attempts = 3
		for range attempts {
			mockUseCase.EXPECT().
				LoginUser(gomock.Any(), dto).
				Return(&domains.TokensDTO{}, customerrors.ErrWrongPassword)
		}

		// Act & Assert - несколько попыток входа
		for range attempts {
			req := createRequest(t, bytes.NewReader(requestBody))
			rr := httptest.NewRecorder()

			handler.ServeHTTP(rr, req)

			assert.Equal(t, http.StatusUnauthorized, rr.Code)
			assert.Contains(t, rr.Body.String(), customerrors.ErrWrongPassword.Error())

			// Куки не должны устанавливаться
			c := rr.Result().Cookies()
			assert.Empty(t, c)
		}
	})
}
