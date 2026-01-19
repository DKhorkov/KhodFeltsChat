//go:build integration

package repositories_test

import (
	"context"
	"database/sql"
	"os"
	"path"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/repositories"
	"github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/loadenv"
	"github.com/DKhorkov/libs/logging"
	"github.com/pressly/goose"
	"github.com/stretchr/testify/suite"
)

func TestAuthRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(AuthRepositoryTestSuite))
}

type AuthRepositoryTestSuite struct {
	suite.Suite

	cwd         string
	ctx         context.Context
	dbConnector postgresql.Connector
	pool        *sql.DB
	tx          postgresql.Transaction
	repository  *repositories.AuthRepository
}

func (s *AuthRepositoryTestSuite) SetupSuite() {
	// Инициализируем переменные окружения
	loadenv.Init("../../.env")

	cwd, err := os.Getwd()
	s.NoError(err)

	cfg := config.New()
	logger := logging.New(
		cfg.Logging.Level,
		cfg.Logging.LogFilePath,
	)

	dbConnector, err := postgresql.New(
		postgresql.BuildDsn(cfg.Database),
		cfg.Database.Driver,
		logger,
		postgresql.WithMaxOpenConnections(cfg.Database.Pool.MaxOpenConnections),
		postgresql.WithMaxIdleConnections(cfg.Database.Pool.MaxIdleConnections),
		postgresql.WithMaxConnectionLifetime(cfg.Database.Pool.MaxConnectionLifetime),
		postgresql.WithMaxConnectionIdleTime(cfg.Database.Pool.MaxConnectionIdleTime),
	)
	s.NoError(err, "failed to connect to database")

	s.NoError(goose.SetDialect(cfg.Database.Driver))

	s.cwd = cwd
	s.dbConnector = dbConnector
	s.ctx = context.Background()

	sqlPool, ok := s.dbConnector.Pool().(*sql.DB)
	s.True(ok)

	s.pool = sqlPool
}

func (s *AuthRepositoryTestSuite) SetupTest() {
	// Применяем миграции
	s.NoError(
		goose.Up(
			s.pool,
			path.Dir(
				path.Dir(s.cwd),
			)+"/migrations",
		),
	)

	tx, err := s.dbConnector.Transaction(s.ctx)
	s.NoError(err)
	s.tx = tx

	// Создаем репозиторий с транзакцией
	s.repository = repositories.NewAuthRepository(tx)
}

func (s *AuthRepositoryTestSuite) TearDownTest() {
	// Откатываем транзакцию
	if s.tx != nil {
		s.NoError(s.tx.Rollback())
	}

	// Откатываем миграции
	s.NoError(
		goose.DownTo(
			s.pool,
			path.Dir(
				path.Dir(s.cwd),
			)+"/migrations",
			0,
		),
	)
}

func (s *AuthRepositoryTestSuite) TearDownSuite() {
	if s.dbConnector != nil {
		s.NoError(s.dbConnector.Close())
	}
}

func (s *AuthRepositoryTestSuite) TestRegisterUser_Success() {
	// Тест: Успешная регистрация пользователя
	userData := domains.RegisterDTO{
		Username: "new_user22",
		Email:    "new22@example.com",
		Password: "$2a$10$newhashedpassword222",
	}

	userID, err := s.repository.RegisterUser(s.ctx, userData)
	s.NoError(err)
	s.Greater(userID, uint64(0))

	// Проверяем, что пользователь создан
	var (
		dbUsername       string
		dbEmail          string
		dbEmailConfirmed bool
		dbPassword       string
	)

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT username, email, email_confirmed, password FROM users WHERE id = $1`,
		userID,
	).Scan(&dbUsername, &dbEmail, &dbEmailConfirmed, &dbPassword)
	s.NoError(err)

	s.Equal(userData.Username, dbUsername)
	s.Equal(userData.Email, dbEmail)
	s.False(dbEmailConfirmed) // По умолчанию должен быть false
	s.Equal(userData.Password, dbPassword)
}

func (s *AuthRepositoryTestSuite) TestRegisterUser_DuplicateUsername() {
	s.createTestUsers()

	// Тест: Регистрация с существующим username
	userData := domains.RegisterDTO{
		Username: "john_doe", // Уже существует
		Email:    "unique@example.com",
		Password: "$2a$10$newhashedpassword",
	}

	userID, err := s.repository.RegisterUser(s.ctx, userData)
	s.Error(err)
	s.Equal(uint64(0), userID)
	s.Contains(err.Error(), "unique constraint") // Должна быть ошибка уникальности
}

func (s *AuthRepositoryTestSuite) TestRegisterUser_DuplicateEmail() {
	s.createTestUsers()

	// Тест: Регистрация с существующим email
	userData := domains.RegisterDTO{
		Username: "unique_username",
		Email:    "john@example.com", // Уже существует
		Password: "$2a$10$newhashedpassword",
	}

	userID, err := s.repository.RegisterUser(s.ctx, userData)
	s.Error(err)
	s.Equal(uint64(0), userID)
	s.Contains(err.Error(), "unique constraint") // Должна быть ошибка уникальности
}

func (s *AuthRepositoryTestSuite) TestRegisterUser_EmptyFields() {
	s.createTestUsers()

	// Тест 1: Пустое имя пользователя
	userData := domains.RegisterDTO{
		Username: "",
		Email:    "test@example.com",
		Password: "$2a$10$hashedpassword",
	}

	userID, err := s.repository.RegisterUser(s.ctx, userData)
	s.Error(err)
	s.Equal(uint64(0), userID)

	// Тест 2: Пустой email
	userData = domains.RegisterDTO{
		Username: "testuser",
		Email:    "",
		Password: "$2a$10$hashedpassword",
	}

	userID, err = s.repository.RegisterUser(s.ctx, userData)
	s.Error(err)
	s.Equal(uint64(0), userID)

	// Тест 3: Пустой пароль
	userData = domains.RegisterDTO{
		Username: "testuser",
		Email:    "test@example.com",
		Password: "",
	}

	userID, err = s.repository.RegisterUser(s.ctx, userData)
	s.Error(err)
	s.Equal(uint64(0), userID)
}

func (s *AuthRepositoryTestSuite) TestCreateRefreshToken_Success() {
	s.createTestUsers()

	// Тест: Создание refresh token
	userID := uint64(1)
	refreshToken := "new_refresh_token_value"
	ttl := 24 * time.Hour

	tokenID, err := s.repository.CreateRefreshToken(s.ctx, userID, refreshToken, ttl)
	s.NoError(err)
	s.Greater(tokenID, uint64(0))

	// Проверяем, что токен создан
	var (
		dbUserID uint64
		dbValue  string
		dbTTL    time.Time
	)

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT user_id, value, ttl FROM refresh_tokens WHERE id = $1`,
		tokenID,
	).Scan(&dbUserID, &dbValue, &dbTTL)
	s.NoError(err)

	s.Equal(userID, dbUserID)
	s.Equal(refreshToken, dbValue)
	s.WithinDuration(time.Now().UTC().Add(ttl), dbTTL, time.Second)
}

func (s *AuthRepositoryTestSuite) TestCreateRefreshToken_DuplicateValue() {
	s.createTestUsers()

	// Тест: Создание токена с дублирующимся значением
	s.createTestRefreshTokens()

	userID := uint64(2)
	refreshToken := "valid_refresh_token_1" // Уже существует
	ttl := 24 * time.Hour

	tokenID, err := s.repository.CreateRefreshToken(s.ctx, userID, refreshToken, ttl)
	s.Error(err)
	s.Equal(uint64(0), tokenID)
	s.Contains(err.Error(), "unique constraint") // Должна быть ошибка уникальности
}

func (s *AuthRepositoryTestSuite) TestCreateRefreshToken_InvalidUser() {
	// Тест: Создание токена для несуществующего пользователя
	userID := uint64(999) // Не существует
	refreshToken := "some_token"
	ttl := 24 * time.Hour

	tokenID, err := s.repository.CreateRefreshToken(s.ctx, userID, refreshToken, ttl)
	s.Error(err) // Должна быть ошибка foreign key constraint
	s.Equal(uint64(0), tokenID)
}

func (s *AuthRepositoryTestSuite) TestCreateRefreshToken_ZeroTTL() {
	s.createTestUsers()

	// Тест: Создание токена с нулевым TTL
	userID := uint64(1)
	refreshToken := "token_with_zero_ttl"
	ttl := 0 * time.Hour

	tokenID, err := s.repository.CreateRefreshToken(s.ctx, userID, refreshToken, ttl)
	s.NoError(err)
	s.Greater(tokenID, uint64(0))

	// Проверяем TTL
	var dbTTL time.Time

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT ttl FROM refresh_tokens WHERE id = $1`,
		tokenID,
	).Scan(&dbTTL)
	s.NoError(err)

	s.WithinDuration(time.Now().UTC(), dbTTL, time.Second)
}

func (s *AuthRepositoryTestSuite) TestGetRefreshTokenByUserID_Success() {
	s.createTestUsers()

	// Тест: Получение валидного refresh token
	s.createTestRefreshTokens()

	token, err := s.repository.GetRefreshTokenByUserID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(token)
	s.Equal(uint64(1), token.ID)
	s.Equal(uint64(1), token.UserID)
	s.Equal("valid_refresh_token_1", token.Value)
	s.True(token.TTL.After(time.Now().UTC())) // Должен быть валидным
	s.NotZero(token.CreatedAt)
	s.NotZero(token.UpdatedAt)

	// Проверяем, что возвращается именно валидный токен (не истекший)
	s.True(token.TTL.After(time.Now().UTC()))
}

func (s *AuthRepositoryTestSuite) TestGetRefreshTokenByUserID_OnlyValidToken() {
	s.createTestUsers()

	// Тест: Должен возвращаться только неистекший токен
	s.createTestRefreshTokens()

	// Для пользователя 1 есть и валидный и истекший токен
	// Должен вернуться только валидный
	token, err := s.repository.GetRefreshTokenByUserID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(token)
	s.Equal("valid_refresh_token_1", token.Value)    // Должен быть валидный токен
	s.NotEqual("expired_refresh_token", token.Value) // Не должен быть истекший
}

func (s *AuthRepositoryTestSuite) TestGetRefreshTokenByUserID_NotFound() {
	// Тест 1: Пользователь без refresh token
	token, err := s.repository.GetRefreshTokenByUserID(s.ctx, 3) // У пользователя 3 нет токенов
	s.Error(err)
	s.Nil(token)
	s.Contains(err.Error(), "no rows")

	// Тест 2: Несуществующий пользователь
	token, err = s.repository.GetRefreshTokenByUserID(s.ctx, 999)
	s.Error(err)
	s.Nil(token)
	s.Contains(err.Error(), "no rows")
}

func (s *AuthRepositoryTestSuite) TestGetRefreshTokenByUserID_AllTokensExpired() {
	// Создаем пользователя только с истекшим токеном
	_, err := s.tx.ExecContext(
		s.ctx,
		`INSERT INTO users (id, username, email, email_confirmed, password) 
		 VALUES (4, 'expired_user', 'expired@example.com', true, 'hashed')`,
	)
	s.NoError(err)

	_, err = s.tx.ExecContext(
		s.ctx,
		`INSERT INTO refresh_tokens (user_id, value, ttl) 
		 VALUES (4, 'expired_only', NOW() - INTERVAL '1 day')`,
	)
	s.NoError(err)

	token, err := s.repository.GetRefreshTokenByUserID(s.ctx, 4)
	s.Error(err)
	s.Nil(token)
	s.Contains(err.Error(), "no rows")
}

func (s *AuthRepositoryTestSuite) TestExpireRefreshToken_Success() {
	s.createTestUsers()

	// Тест: Успешное истечение refresh token
	s.createTestRefreshTokens()

	refreshToken := "valid_refresh_token_1"
	err := s.repository.ExpireRefreshToken(s.ctx, refreshToken)
	s.NoError(err)

	// Проверяем, что TTL стал в прошлом
	var ttl time.Time

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT ttl FROM refresh_tokens WHERE value = $1`,
		refreshToken,
	).Scan(&ttl)
	s.NoError(err)

	// TTL должен быть в прошлом (минус 24 часа от текущего времени)
	expectedExpiry := time.Now().UTC().Add(-24 * time.Hour)
	s.WithinDuration(expectedExpiry, ttl, 5*time.Second)
	s.True(ttl.Before(time.Now().UTC()))
}

func (s *AuthRepositoryTestSuite) TestExpireRefreshToken_NotFound() {
	// Тест: Попытка истечь несуществующий токен
	err := s.repository.ExpireRefreshToken(s.ctx, "nonexistent_token")
	s.NoError(err) // UPDATE без найденных строк не возвращает ошибку

	// Проверяем, что ничего не изменилось
	var count int

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM refresh_tokens WHERE ttl < NOW() - INTERVAL '23 hours'`,
	).Scan(&count)
	s.NoError(err)
	s.Equal(0, count) // Ничего не должно быть истекшим
}

func (s *AuthRepositoryTestSuite) TestExpireRefreshToken_AlreadyExpired() {
	// Тест: Истечение уже истекшего токена
	s.createTestUsers()
	s.createTestRefreshTokens()

	refreshToken := "expired_refresh_token"

	// Получаем текущий TTL
	var currentTTL time.Time

	err := s.tx.QueryRowContext(
		s.ctx,
		`SELECT ttl FROM refresh_tokens WHERE value = $1`,
		refreshToken,
	).Scan(&currentTTL)
	s.NoError(err)
	s.True(currentTTL.Before(time.Now().UTC())) // Убеждаемся, что уже истек

	// Выполняем истечение
	err = s.repository.ExpireRefreshToken(s.ctx, refreshToken)
	s.NoError(err)

	// Проверяем новый TTL
	var newTTL time.Time

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT ttl FROM refresh_tokens WHERE value = $1`,
		refreshToken,
	).Scan(&newTTL)
	s.NoError(err)

	// Новый TTL должен быть еще раньше
	expectedExpiry := time.Now().UTC().Add(-24 * time.Hour)
	s.WithinDuration(expectedExpiry, newTTL, 5*time.Second)
	s.True(newTTL.Before(currentTTL)) // Новый TTL должен быть раньше старого
}

func (s *AuthRepositoryTestSuite) TestVerifyEmail_Success() {
	s.createTestUsers()

	// Тест: Успешное подтверждение email
	userID := uint64(2) // Пользователь с неподтвержденным email

	// Проверяем начальное состояние
	var initialConfirmed bool

	err := s.tx.QueryRowContext(
		s.ctx,
		`SELECT email_confirmed FROM users WHERE id = $1`,
		userID,
	).Scan(&initialConfirmed)
	s.NoError(err)
	s.False(initialConfirmed)

	// Выполняем подтверждение
	err = s.repository.VerifyEmail(s.ctx, userID)
	s.NoError(err)

	// Проверяем конечное состояние
	var finalConfirmed bool

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT email_confirmed FROM users WHERE id = $1`,
		userID,
	).Scan(&finalConfirmed)
	s.NoError(err)
	s.True(finalConfirmed)
}

func (s *AuthRepositoryTestSuite) TestVerifyEmail_AlreadyConfirmed() {
	s.createTestUsers()

	// Тест: Подтверждение уже подтвержденного email
	userID := uint64(1) // Пользователь с уже подтвержденным email

	// Проверяем начальное состояние
	var initialConfirmed bool

	err := s.tx.QueryRowContext(
		s.ctx,
		`SELECT email_confirmed FROM users WHERE id = $1`,
		userID,
	).Scan(&initialConfirmed)
	s.NoError(err)
	s.True(initialConfirmed)

	// Выполняем подтверждение (должно остаться true)
	err = s.repository.VerifyEmail(s.ctx, userID)
	s.NoError(err)

	// Проверяем конечное состояние
	var finalConfirmed bool

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT email_confirmed FROM users WHERE id = $1`,
		userID,
	).Scan(&finalConfirmed)
	s.NoError(err)
	s.True(finalConfirmed)
}

func (s *AuthRepositoryTestSuite) TestVerifyEmail_UserNotFound() {
	// Тест: Попытка подтвердить email несуществующего пользователя
	err := s.repository.VerifyEmail(s.ctx, 999)
	s.NoError(err) // UPDATE без найденных строк не возвращает ошибку

	// Проверяем, что ничего не изменилось
	var count int

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM users WHERE id = 999`,
	).Scan(&count)
	s.NoError(err)
	s.Equal(0, count)
}

func (s *AuthRepositoryTestSuite) TestChangePassword_Success() {
	s.createTestUsers()

	// Тест: Успешная смена пароля
	userID := uint64(1)
	newPassword := "$2a$10$newsecurehashedpassword"

	// Получаем старый пароль
	var oldPassword string

	err := s.tx.QueryRowContext(
		s.ctx,
		`SELECT password FROM users WHERE id = $1`,
		userID,
	).Scan(&oldPassword)
	s.NoError(err)

	// Меняем пароль
	err = s.repository.ChangePassword(s.ctx, userID, newPassword)
	s.NoError(err)

	// Проверяем новый пароль
	var dbPassword string

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT password FROM users WHERE id = $1`,
		userID,
	).Scan(&dbPassword)
	s.NoError(err)

	s.Equal(newPassword, dbPassword)
	s.NotEqual(oldPassword, dbPassword)

	// Проверяем, что другие поля не изменились
	var (
		dbUsername       string
		dbEmail          string
		dbEmailConfirmed bool
	)

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT username, email, email_confirmed FROM users WHERE id = $1`,
		userID,
	).Scan(&dbUsername, &dbEmail, &dbEmailConfirmed)
	s.NoError(err)

	s.Equal("john_doe", dbUsername)
	s.Equal("john@example.com", dbEmail)
	s.True(dbEmailConfirmed)
}

func (s *AuthRepositoryTestSuite) TestChangePassword_SamePassword() {
	s.createTestUsers()

	// Тест: Смена пароля на тот же самый
	userID := uint64(1)
	samePassword := "$2a$10$hashedpassword1" // Тот же пароль

	err := s.repository.ChangePassword(s.ctx, userID, samePassword)
	s.NoError(err)

	// Проверяем, что пароль остался тем же
	var dbPassword string

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT password FROM users WHERE id = $1`,
		userID,
	).Scan(&dbPassword)
	s.NoError(err)

	s.Equal(samePassword, dbPassword)
}

func (s *AuthRepositoryTestSuite) TestChangePassword_EmptyPassword() {
	s.createTestUsers()

	// Тест: Смена пароля на пустой
	userID := uint64(1)
	emptyPassword := ""

	err := s.repository.ChangePassword(s.ctx, userID, emptyPassword)
	s.NoError(err)

	// Проверяем, что пароль стал пустым
	var dbPassword string

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT password FROM users WHERE id = $1`,
		userID,
	).Scan(&dbPassword)
	s.NoError(err)

	s.Equal(emptyPassword, dbPassword)
}

func (s *AuthRepositoryTestSuite) TestChangePassword_UserNotFound() {
	// Тест: Попытка сменить пароль несуществующему пользователю
	err := s.repository.ChangePassword(s.ctx, 999, "newpassword")
	s.NoError(err) // UPDATE без найденных строк не возвращает ошибку

	// Проверяем, что пользователь не создан
	var count int

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM users WHERE id = 999`,
	).Scan(&count)
	s.NoError(err)
	s.Equal(0, count)
}

func (s *AuthRepositoryTestSuite) createTestUsers() {
	users := []struct {
		id             uint64
		username       string
		email          string
		emailConfirmed bool
		password       string
	}{
		{1, "john_doe", "john@example.com", true, "$2a$10$hashedpassword1"},
		{2, "jane_smith", "jane@example.com", false, "$2a$10$hashedpassword2"},
		{3, "bob_wilson", "bob@example.com", true, "$2a$10$hashedpassword3"},
	}

	for _, u := range users {
		_, err := s.tx.ExecContext(
			s.ctx,
			`INSERT INTO users (id, username, email, email_confirmed, password, created_at, updated_at) 
			 VALUES ($1, $2, $3, $4, $5, $6, $6)`,
			u.id,
			u.username,
			u.email,
			u.emailConfirmed,
			u.password,
			time.Now().UTC(),
		)
		s.NoError(err)
	}
}

func (s *AuthRepositoryTestSuite) createTestRefreshTokens() {
	refreshTokens := []struct {
		id     uint64
		userID uint64
		value  string
		ttl    time.Time
	}{
		{1, 1, "valid_refresh_token_1", time.Now().UTC().Add(24 * time.Hour)},
		{2, 2, "valid_refresh_token_2", time.Now().UTC().Add(12 * time.Hour)},
		{3, 1, "expired_refresh_token", time.Now().UTC().Add(-1 * time.Hour)},
	}

	for _, rt := range refreshTokens {
		_, err := s.tx.ExecContext(
			s.ctx,
			`INSERT INTO refresh_tokens (id, user_id, value, ttl, created_at, updated_at) 
			 VALUES ($1, $2, $3, $4, $5, $5)`,
			rt.id,
			rt.userID,
			rt.value,
			rt.ttl,
			time.Now().UTC(),
		)
		s.NoError(err)
	}
}
