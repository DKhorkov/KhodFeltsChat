//go:build integration

package repositories_test

import (
	"context"
	"database/sql"
	"os"
	"path"
	"strings"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/repositories"
	"github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/loadenv"
	"github.com/DKhorkov/libs/logging"
	"github.com/DKhorkov/libs/pointers"
	"github.com/pressly/goose"
	"github.com/stretchr/testify/suite"
)

func TestUsersRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(UsersRepositoryTestSuite))
}

type UsersRepositoryTestSuite struct {
	suite.Suite

	cwd         string
	ctx         context.Context
	dbConnector postgresql.Connector
	pool        *sql.DB
	tx          postgresql.Transaction
	repository  *repositories.UsersRepository
	logger      logging.Logger
}

func (s *UsersRepositoryTestSuite) SetupSuite() {
	// Инициализируем переменные окружения
	loadenv.Init("../../.env")

	cwd, err := os.Getwd()
	s.NoError(err)

	cfg := config.New()
	s.logger = logging.New(
		cfg.Logging.Level,
		cfg.Logging.LogFilePath,
	)

	dbConnector, err := postgresql.New(
		postgresql.BuildDsn(cfg.Database),
		cfg.Database.Driver,
		s.logger,
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

func (s *UsersRepositoryTestSuite) SetupTest() {
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
	s.repository = repositories.NewUsersRepository(tx, s.logger)

	// Создаем тестовых пользователей
	s.createTestUsers()
}

func (s *UsersRepositoryTestSuite) TearDownTest() {
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

func (s *UsersRepositoryTestSuite) TearDownSuite() {
	if s.dbConnector != nil {
		s.NoError(s.dbConnector.Close())
	}
}

func (s *UsersRepositoryTestSuite) TestGetUserByID_Success() {
	// Тест 1: Получение существующего пользователя с подтвержденным email
	user, err := s.repository.GetUserByID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(user)
	s.Equal(uint64(1), user.ID)
	s.Equal("john_doe", user.Username)
	s.Equal("john@example.com", user.Email)
	s.True(user.EmailConfirmed)
	s.Equal("$2a$10$hashedpassword1", user.Password)
	s.NotZero(user.CreatedAt)
	s.NotZero(user.UpdatedAt)

	// Тест 2: Получение пользователя с неподтвержденным email
	user, err = s.repository.GetUserByID(s.ctx, 2)
	s.NoError(err)
	s.NotNil(user)
	s.Equal(uint64(2), user.ID)
	s.Equal("jane_smith", user.Username)
	s.Equal("jane@example.com", user.Email)
	s.False(user.EmailConfirmed)
	s.Equal("$2a$10$hashedpassword2", user.Password)

	// Тест 3: Получение пользователя с длинным именем
	user, err = s.repository.GetUserByID(s.ctx, 8)
	s.NoError(err)
	s.NotNil(user)
	s.Equal(uint64(8), user.ID)
	s.Equal("user_with_long_name", user.Username)
	s.True(user.EmailConfirmed)
}

func (s *UsersRepositoryTestSuite) TestGetUserByID_NotFound() {
	// Тест: Попытка получить несуществующего пользователя
	user, err := s.repository.GetUserByID(s.ctx, 999)
	s.Error(err)
	s.Nil(user)
	s.Contains(err.Error(), "no rows")
}

func (s *UsersRepositoryTestSuite) TestGetUserByID_ZeroID() {
	// Тест: Попытка получить пользователя с ID = 0
	user, err := s.repository.GetUserByID(s.ctx, 0)
	s.Error(err)
	s.Nil(user)
}

func (s *UsersRepositoryTestSuite) TestGetUserByUsername_Success() {
	// Тест 1: Получение пользователя по имени пользователя с подтвержденным email
	user, err := s.repository.GetUserByUsername(s.ctx, "john_doe")
	s.NoError(err)
	s.NotNil(user)
	s.Equal(uint64(1), user.ID)
	s.Equal("john_doe", user.Username)
	s.Equal("john@example.com", user.Email)
	s.True(user.EmailConfirmed)
	s.Equal("$2a$10$hashedpassword1", user.Password)

	// Тест 2: Получение пользователя с неподтвержденным email
	user, err = s.repository.GetUserByUsername(s.ctx, "jane_smith")
	s.NoError(err)
	s.NotNil(user)
	s.Equal(uint64(2), user.ID)
	s.Equal("jane_smith", user.Username)
	s.Equal("jane@example.com", user.Email)
	s.False(user.EmailConfirmed)

	// Тест 3: Получение пользователя с подчеркиваниями
	user, err = s.repository.GetUserByUsername(s.ctx, "user_with_long_name")
	s.NoError(err)
	s.NotNil(user)
	s.Equal(uint64(8), user.ID)
}

func (s *UsersRepositoryTestSuite) TestGetUserByUsername_NotFound() {
	// Тест 1: Несуществующее имя пользователя
	user, err := s.repository.GetUserByUsername(s.ctx, "nonexistent")
	s.Error(err)
	s.Nil(user)
	s.Contains(err.Error(), "no rows")

	// Тест 2: Пустое имя пользователя
	user, err = s.repository.GetUserByUsername(s.ctx, "")
	s.Error(err)
	s.Nil(user)
}

func (s *UsersRepositoryTestSuite) TestGetUserByUsername_CaseSensitive() {
	// Тест: Проверка чувствительности к регистру
	user, err := s.repository.GetUserByUsername(s.ctx, "JOHN_DOE") // в верхнем регистре
	s.Error(err)
	s.Nil(user)
	s.Contains(err.Error(), "no rows")
}

func (s *UsersRepositoryTestSuite) TestGetUserByEmail_Success() {
	// Тест 1: Получение пользователя по email с подтвержденным email
	user, err := s.repository.GetUserByEmail(s.ctx, "john@example.com")
	s.NoError(err)
	s.NotNil(user)
	s.Equal(uint64(1), user.ID)
	s.Equal("john_doe", user.Username)
	s.Equal("john@example.com", user.Email)
	s.True(user.EmailConfirmed)
	s.Equal("$2a$10$hashedpassword1", user.Password)

	// Тест 2: Получение пользователя с неподтвержденным email
	user, err = s.repository.GetUserByEmail(s.ctx, "jane@example.com")
	s.NoError(err)
	s.NotNil(user)
	s.Equal(uint64(2), user.ID)
	s.Equal("jane_smith", user.Username)
	s.Equal("jane@example.com", user.Email)
	s.False(user.EmailConfirmed)

	// Тест 3: Получение пользователя по длинному email
	user, err = s.repository.GetUserByEmail(s.ctx, "long@example.com")
	s.NoError(err)
	s.NotNil(user)
	s.Equal(uint64(8), user.ID)
}

func (s *UsersRepositoryTestSuite) TestGetUserByEmail_NotFound() {
	// Тест 1: Несуществующий email
	user, err := s.repository.GetUserByEmail(s.ctx, "nonexistent@example.com")
	s.Error(err)
	s.Nil(user)
	s.Contains(err.Error(), "no rows")

	// Тест 2: Пустой email
	user, err = s.repository.GetUserByEmail(s.ctx, "")
	s.Error(err)
	s.Nil(user)
}

func (s *UsersRepositoryTestSuite) TestGetUserByEmail_CaseSensitive() {
	// Тест: Email должен быть чувствителен к регистру
	user, err := s.repository.GetUserByEmail(s.ctx, "JOHN@EXAMPLE.COM") // в верхнем регистре
	s.Error(err)
	s.Nil(user)
	s.Contains(err.Error(), "no rows")
}

func (s *UsersRepositoryTestSuite) TestGetUsers_WithoutFilters() {
	// Тест: Получение всех пользователей без фильтров
	users, err := s.repository.GetUsers(s.ctx, nil, nil)
	s.NoError(err)
	s.NotNil(users)
	s.GreaterOrEqual(len(users), 8) // Должно быть как минимум 8 тестовых пользователей

	// Проверяем сортировку по ID в порядке убывания
	for i := range len(users) - 1 {
		s.Greater(users[i].ID, users[i+1].ID, "Users should be sorted by ID DESC")
	}

	// Проверяем данные первого пользователя (должен быть с наибольшим ID)
	s.Equal(uint64(8), users[0].ID)
	s.Equal("user_with_long_name", users[0].Username)
	s.True(users[0].EmailConfirmed)
	s.NotEmpty(users[0].Password)

	// Проверяем, что все поля заполнены
	for _, user := range users {
		s.NotZero(user.ID)
		s.NotEmpty(user.Username)
		s.NotEmpty(user.Email)
		s.NotEmpty(user.Password)
		s.NotZero(user.CreatedAt)
		s.NotZero(user.UpdatedAt)
	}
}

func (s *UsersRepositoryTestSuite) TestGetUsers_WithUsernameFilter() {
	// Тест 1: Поиск пользователей по части имени (регистронезависимый)
	filters := &domains.UsersFilters{
		Username: pointers.New("test"),
	}
	users, err := s.repository.GetUsers(s.ctx, filters, nil)
	s.NoError(err)
	s.NotNil(users)

	// Должны найтись пользователи с "test" в имени
	s.GreaterOrEqual(len(users), 2)

	for _, user := range users {
		s.Contains(strings.ToLower(user.Username), "test")
	}

	// Тест 2: Поиск пользователей по полному имени
	filters.Username = pointers.New("john_doe")
	users, err = s.repository.GetUsers(s.ctx, filters, nil)
	s.NoError(err)
	s.NotNil(users)
	s.Equal(1, len(users))
	s.Equal("john_doe", users[0].Username)
	s.True(users[0].EmailConfirmed)

	// Тест 3: Поиск пользователей с пустым фильтром
	filters.Username = pointers.New("")
	users, err = s.repository.GetUsers(s.ctx, filters, nil)
	s.NoError(err)
	s.NotNil(users)
	s.GreaterOrEqual(len(users), 8) // Должны вернуться все пользователи

	// Тест 4: Поиск с регистронезависимым фильтром
	filters.Username = pointers.New("TEST") // в верхнем регистре
	users, err = s.repository.GetUsers(s.ctx, filters, nil)
	s.NoError(err)
	s.NotNil(users)
	s.GreaterOrEqual(len(users), 2)

	for _, user := range users {
		s.Contains(strings.ToLower(user.Username), "test")
	}
}

func (s *UsersRepositoryTestSuite) TestGetUsers_WithPagination() {
	// Тест 1: Пагинация с лимитом
	pagination := &domains.Pagination{
		Limit: pointers.New[uint64](3),
	}
	users, err := s.repository.GetUsers(s.ctx, nil, pagination)
	s.NoError(err)
	s.NotNil(users)
	s.Equal(3, len(users))

	// Проверяем сортировку
	s.Equal(uint64(8), users[0].ID)
	s.Equal(uint64(7), users[1].ID)
	s.Equal(uint64(6), users[2].ID)

	// Проверяем все поля
	for _, user := range users {
		s.NotZero(user.ID)
		s.NotEmpty(user.Username)
		s.NotEmpty(user.Email)
		s.NotEmpty(user.Password)
	}

	// Тест 2: Пагинация с оффсетом
	pagination = &domains.Pagination{
		Limit:  pointers.New[uint64](2),
		Offset: pointers.New[uint64](2),
	}
	users, err = s.repository.GetUsers(s.ctx, nil, pagination)
	s.NoError(err)
	s.NotNil(users)
	s.Equal(2, len(users))

	// Должны пропустить первых двух пользователей
	s.Equal(uint64(6), users[0].ID)
	s.Equal(uint64(5), users[1].ID)

	// Тест 3: Пагинация с фильтром и лимитом
	filters := &domains.UsersFilters{
		Username: pointers.New("test"),
	}
	pagination = &domains.Pagination{
		Limit: pointers.New[uint64](1),
	}
	users, err = s.repository.GetUsers(s.ctx, filters, pagination)
	s.NoError(err)
	s.NotNil(users)
	s.Equal(1, len(users))
	s.Contains(strings.ToLower(users[0].Username), "test")

	// Тест 4: Пагинация с лимитом 0 (должны вернуться пустой список)
	pagination = &domains.Pagination{
		Limit: pointers.New[uint64](0),
	}
	users, err = s.repository.GetUsers(s.ctx, nil, pagination)
	s.NoError(err)
	s.Nil(users)
}

func (s *UsersRepositoryTestSuite) TestGetUsers_WithFilterAndPagination() {
	// Тест: Комбинированный тест с фильтром и пагинацией
	filters := &domains.UsersFilters{
		Username: pointers.New("test"),
	}
	pagination := &domains.Pagination{
		Limit:  pointers.New[uint64](1),
		Offset: pointers.New[uint64](1),
	}

	users, err := s.repository.GetUsers(s.ctx, filters, pagination)
	s.NoError(err)
	s.NotNil(users)
	s.Equal(1, len(users))
	s.Contains(strings.ToLower(users[0].Username), "test")
}

func (s *UsersRepositoryTestSuite) TestGetUsers_EmptyResult() {
	// Тест: Поиск несуществующего пользователя
	filters := &domains.UsersFilters{
		Username: pointers.New("nonexistentusername"),
	}
	users, err := s.repository.GetUsers(s.ctx, filters, nil)
	s.NoError(err)
	s.Nil(users)
}

func (s *UsersRepositoryTestSuite) TestUpdateUser_Success() {
	// Тест 1: Обновление имени пользователя (email_confirmed и password не должны изменяться)
	userData := domains.UpdateUserDTO{
		ID:       1,
		Username: "updated_john",
	}

	// Получаем оригинальные данные пользователя
	originalUser, err := s.repository.GetUserByID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(originalUser)

	// Обновляем пользователя
	err = s.repository.UpdateUser(s.ctx, userData)
	s.NoError(err)

	// Проверяем, что пользователь обновлен
	updatedUser, err := s.repository.GetUserByID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(updatedUser)
	s.Equal("updated_john", updatedUser.Username)

	// Проверяем, что email_confirmed и password не изменились
	s.Equal(originalUser.Email, updatedUser.Email)
	s.Equal(originalUser.EmailConfirmed, updatedUser.EmailConfirmed)
	s.Equal(originalUser.Password, updatedUser.Password)

	// Тест 2: Обновление другого пользователя
	userData = domains.UpdateUserDTO{
		ID:       2,
		Username: "updated_jane",
	}

	err = s.repository.UpdateUser(s.ctx, userData)
	s.NoError(err)

	updatedUser, err = s.repository.GetUserByID(s.ctx, 2)
	s.NoError(err)
	s.NotNil(updatedUser)
	s.Equal("updated_jane", updatedUser.Username)
	s.False(updatedUser.EmailConfirmed) // Должно остаться false
}

func (s *UsersRepositoryTestSuite) TestUpdateUser_EmptyUsername() {
	// Тест: Обновление с пустым именем пользователя
	userData := domains.UpdateUserDTO{
		ID:       1,
		Username: "",
	}

	err := s.repository.UpdateUser(s.ctx, userData)
	s.NoError(err)

	// Проверяем, что имя пользователя стало пустым
	user, err := s.repository.GetUserByID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(user)
	s.Equal("", user.Username)

	// Проверяем, что другие поля не изменились
	s.Equal("john@example.com", user.Email)
	s.True(user.EmailConfirmed)
	s.Equal("$2a$10$hashedpassword1", user.Password)
}

func (s *UsersRepositoryTestSuite) TestUpdateUser_LongUsername() {
	// Тест: Обновление с длинным именем пользователя
	longUsername := "very_long_username_that_exceeds_normal_length_but_should_work"
	userData := domains.UpdateUserDTO{
		ID:       1,
		Username: longUsername,
	}

	err := s.repository.UpdateUser(s.ctx, userData)
	s.NoError(err)

	user, err := s.repository.GetUserByID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(user)
	s.Equal(longUsername, user.Username)
}

func (s *UsersRepositoryTestSuite) TestUpdateUser_SameUsername() {
	// Тест: Обновление с тем же именем пользователя
	originalUser, err := s.repository.GetUserByID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(originalUser)

	userData := domains.UpdateUserDTO{
		ID:       1,
		Username: originalUser.Username, // То же самое имя
	}

	err = s.repository.UpdateUser(s.ctx, userData)
	s.NoError(err)

	// Проверяем, что updated_at обновлен, даже если имя не изменилось
	updatedUser, err := s.repository.GetUserByID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(updatedUser)
	s.Equal(originalUser.Username, updatedUser.Username)

	// Проверяем, что другие поля не изменились
	s.Equal(originalUser.Email, updatedUser.Email)
	s.Equal(originalUser.EmailConfirmed, updatedUser.EmailConfirmed)
	s.Equal(originalUser.Password, updatedUser.Password)
	s.Equal(originalUser.CreatedAt, updatedUser.CreatedAt)

	// Проверяем, что updated_at обновлен
	s.True(updatedUser.UpdatedAt.After(originalUser.UpdatedAt) ||
		updatedUser.UpdatedAt.Equal(originalUser.UpdatedAt))
}

func (s *UsersRepositoryTestSuite) TestUpdateUser_NotFound() {
	// Тест: Попытка обновить несуществующего пользователя
	userData := domains.UpdateUserDTO{
		ID:       999,
		Username: "new_username",
	}

	err := s.repository.UpdateUser(s.ctx, userData)
	s.NoError(err) // UPDATE без WHERE не должен возвращать ошибку, если строка не найдена

	// Проверяем, что пользователь не был создан
	user, err := s.repository.GetUserByID(s.ctx, 999)
	s.Error(err)
	s.Nil(user)
}

func (s *UsersRepositoryTestSuite) TestUpdateUser_SpecialCharacters() {
	// Тест: Обновление с именем, содержащим специальные символы
	userData := domains.UpdateUserDTO{
		ID:       1,
		Username: "user-name.with.dots_and_underscores",
	}

	err := s.repository.UpdateUser(s.ctx, userData)
	s.NoError(err)

	user, err := s.repository.GetUserByID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(user)
	s.Equal("user-name.with.dots_and_underscores", user.Username)
}

func (s *UsersRepositoryTestSuite) TestUpdateUser_ConcurrentUpdates() {
	// Тест: Проверка, что последнее обновление сохраняется
	userData1 := domains.UpdateUserDTO{
		ID:       1,
		Username: "first_update",
	}

	userData2 := domains.UpdateUserDTO{
		ID:       1,
		Username: "second_update",
	}

	// Выполняем два обновления последовательно
	err := s.repository.UpdateUser(s.ctx, userData1)
	s.NoError(err)

	err = s.repository.UpdateUser(s.ctx, userData2)
	s.NoError(err)

	// Проверяем, что сохранилось последнее обновление
	user, err := s.repository.GetUserByID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(user)
	s.Equal("second_update", user.Username)
}

func (s *UsersRepositoryTestSuite) TestUpdateUser_EmailConfirmedNotChanged() {
	// Тест: Проверка, что email_confirmed не меняется при обновлении username
	// Сначала получаем пользователя с неподтвержденным email
	user, err := s.repository.GetUserByID(s.ctx, 2)
	s.NoError(err)
	s.NotNil(user)
	s.False(user.EmailConfirmed)

	// Обновляем username
	userData := domains.UpdateUserDTO{
		ID:       2,
		Username: "updated_username",
	}

	err = s.repository.UpdateUser(s.ctx, userData)
	s.NoError(err)

	// Проверяем, что email_confirmed остался false
	updatedUser, err := s.repository.GetUserByID(s.ctx, 2)
	s.NoError(err)
	s.NotNil(updatedUser)
	s.Equal("updated_username", updatedUser.Username)
	s.False(updatedUser.EmailConfirmed)
}

func (s *UsersRepositoryTestSuite) TestUpdateUser_PasswordNotChanged() {
	// Тест: Проверка, что password не меняется при обновлении username
	// Получаем оригинальный пароль
	originalUser, err := s.repository.GetUserByID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(originalUser)
	originalPassword := originalUser.Password

	// Обновляем username
	userData := domains.UpdateUserDTO{
		ID:       1,
		Username: "updated_username",
	}

	err = s.repository.UpdateUser(s.ctx, userData)
	s.NoError(err)

	// Проверяем, что password не изменился
	updatedUser, err := s.repository.GetUserByID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(updatedUser)
	s.Equal(originalPassword, updatedUser.Password)
}

func (s *UsersRepositoryTestSuite) TestGetUsers_EmailConfirmedFilter() {
	// Тест: Проверяем, что можно получить пользователей с подтвержденным email
	users, err := s.repository.GetUsers(s.ctx, nil, nil)
	s.NoError(err)
	s.NotNil(users)

	// Считаем пользователей с подтвержденным и неподтвержденным email
	confirmedCount := 0
	notConfirmedCount := 0

	for _, user := range users {
		if user.EmailConfirmed {
			confirmedCount++
		} else {
			notConfirmedCount++
		}
	}

	// В тестовых данных должно быть как минимум 4 подтвержденных и 3 неподтвержденных
	s.GreaterOrEqual(confirmedCount, 4)
	s.GreaterOrEqual(notConfirmedCount, 3)
}

func (s *UsersRepositoryTestSuite) createTestUsers() {
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
		{4, "alice_johnson", "alice@example.com", false, "$2a$10$hashedpassword4"},
		{5, "test_user", "test@example.com", true, "$2a$10$hashedpassword5"},
		{6, "search_test", "search@example.com", true, "$2a$10$hashedpassword6"},
		{7, "another_test", "another@example.com", false, "$2a$10$hashedpassword7"},
		{8, "user_with_long_name", "long@example.com", true, "$2a$10$hashedpassword8"},
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
