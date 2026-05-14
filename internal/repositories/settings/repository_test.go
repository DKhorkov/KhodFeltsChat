//go:build integration

package settings_test

import (
	"context"
	"database/sql"
	"os"
	"path"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/repositories/settings"
	"github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/loadenv"
	"github.com/DKhorkov/libs/logging"
	"github.com/pressly/goose"
	"github.com/stretchr/testify/suite"
)

func TestRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(RepositoryTestSuite))
}

type RepositoryTestSuite struct {
	suite.Suite

	cwd         string
	ctx         context.Context
	dbConnector postgresql.Connector
	pool        *sql.DB
	tx          postgresql.Transaction
	repository  *settings.Repository
}

func (s *RepositoryTestSuite) SetupSuite() {
	loadenv.Init("../../../.env")

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

func (s *RepositoryTestSuite) SetupTest() {
	s.NoError(
		goose.Up(
			s.pool,
			path.Dir(
				path.Dir(
					path.Dir(s.cwd),
				),
			)+"/migrations",
		),
	)

	tx, err := s.dbConnector.Transaction(s.ctx)
	s.NoError(err)
	s.tx = tx

	s.repository = settings.New(tx)

	s.createTestData()
}

func (s *RepositoryTestSuite) TearDownTest() {
	if s.tx != nil {
		s.NoError(s.tx.Rollback())
	}

	s.NoError(
		goose.DownTo(
			s.pool,
			path.Dir(
				path.Dir(
					path.Dir(s.cwd),
				),
			)+"/migrations",
			0,
		),
	)
}

func (s *RepositoryTestSuite) TearDownSuite() {
	if s.dbConnector != nil {
		s.NoError(s.dbConnector.Close())
	}
}

func (s *RepositoryTestSuite) TestCreateSettings_Success() {
	// Создаем настройки для нового пользователя (без существующих настроек)
	newSettings := domains.Settings{
		UserID: 3,
		Theme:  domains.ThemeDark,
	}

	err := s.repository.CreateSettings(s.ctx, newSettings)
	s.NoError(err)

	// Проверяем, что настройки были созданы
	result, err := s.repository.GetSettingsByUserID(s.ctx, 3)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(uint64(3), result.UserID)
	s.Equal(domains.ThemeDark, result.Theme)
	s.NotZero(result.ID)
	s.NotZero(result.CreatedAt)
	s.NotZero(result.UpdatedAt)
}

func (s *RepositoryTestSuite) TestCreateSettings_DefaultTheme() {
	// Создаем настройки с темой по умолчанию (Light)
	newSettings := domains.Settings{
		UserID: 3,
		Theme:  domains.ThemeLight,
	}

	err := s.repository.CreateSettings(s.ctx, newSettings)
	s.NoError(err)

	result, err := s.repository.GetSettingsByUserID(s.ctx, 3)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(domains.ThemeLight, result.Theme)
}

func (s *RepositoryTestSuite) TestCreateSettings_DuplicateUserID() {
	// Попытка создать настройки для пользователя, у которого уже есть настройки
	duplicateSettings := domains.Settings{
		UserID: 1,
		Theme:  domains.ThemeDark,
	}

	err := s.repository.CreateSettings(s.ctx, duplicateSettings)
	s.Error(err) // UNIQUE constraint на user_id
}

func (s *RepositoryTestSuite) TestGetSettingsByUserID_Success() {
	// Тест 1: Получение настроек пользователя с светлой темой
	result, err := s.repository.GetSettingsByUserID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(uint64(1), result.UserID)
	s.Equal(domains.ThemeLight, result.Theme)
	s.NotZero(result.ID)
	s.NotZero(result.CreatedAt)
	s.NotZero(result.UpdatedAt)

	// Тест 2: Получение настроек пользователя с тёмной темой
	result, err = s.repository.GetSettingsByUserID(s.ctx, 2)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(uint64(2), result.UserID)
	s.Equal(domains.ThemeDark, result.Theme)
	s.NotZero(result.ID)
	s.NotZero(result.CreatedAt)
	s.NotZero(result.UpdatedAt)
}

func (s *RepositoryTestSuite) TestGetSettingsByUserID_NotFound() {
	// Попытка получить настройки несуществующего пользователя
	result, err := s.repository.GetSettingsByUserID(s.ctx, 999)
	s.Error(err)
	s.Nil(result)
	s.Contains(err.Error(), "no rows")
}

func (s *RepositoryTestSuite) TestGetSettingsByUserID_ZeroID() {
	// Попытка получить настройки с ID = 0
	result, err := s.repository.GetSettingsByUserID(s.ctx, 0)
	s.Error(err)
	s.Nil(result)
}

func (s *RepositoryTestSuite) TestUpdateSettings_Success() {
	// Получаем оригинальные настройки
	original, err := s.repository.GetSettingsByUserID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(original)
	s.Equal(domains.ThemeLight, original.Theme)

	// Обновляем тему
	updatedSettings := domains.Settings{
		UserID: 1,
		Theme:  domains.ThemeDark,
	}

	err = s.repository.UpdateSettings(s.ctx, updatedSettings)
	s.NoError(err)

	// Проверяем, что тема обновлена
	result, err := s.repository.GetSettingsByUserID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(domains.ThemeDark, result.Theme)
	s.Equal(original.ID, result.ID)
	s.Equal(original.UserID, result.UserID)
	s.Equal(original.CreatedAt, result.CreatedAt)

	// Проверяем, что updated_at обновлен
	s.True(result.UpdatedAt.After(original.UpdatedAt) ||
		result.UpdatedAt.Equal(original.UpdatedAt))
}

func (s *RepositoryTestSuite) TestUpdateSettings_SameTheme() {
	// Получаем оригинальные настройки
	original, err := s.repository.GetSettingsByUserID(s.ctx, 2)
	s.NoError(err)
	s.NotNil(original)

	// Обновляем с той же темой
	sameSettings := domains.Settings{
		UserID: 2,
		Theme:  original.Theme,
	}

	err = s.repository.UpdateSettings(s.ctx, sameSettings)
	s.NoError(err)

	// Проверяем, что данные остались прежними
	result, err := s.repository.GetSettingsByUserID(s.ctx, 2)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(original.Theme, result.Theme)
	s.Equal(original.ID, result.ID)
	s.Equal(original.UserID, result.UserID)
	s.Equal(original.CreatedAt, result.CreatedAt)
}

func (s *RepositoryTestSuite) TestUpdateSettings_NotFound() {
	// Попытка обновить настройки несуществующего пользователя
	updatedSettings := domains.Settings{
		UserID: 999,
		Theme:  domains.ThemeDark,
	}

	err := s.repository.UpdateSettings(s.ctx, updatedSettings)
	s.NoError(err) // UPDATE без совпадений не возвращает ошибку

	// Проверяем, что настройки не были созданы
	result, err := s.repository.GetSettingsByUserID(s.ctx, 999)
	s.Error(err)
	s.Nil(result)
}

func (s *RepositoryTestSuite) TestUpdateSettings_SequentialUpdates() {
	// Проверяем, что последнее обновление сохраняется
	err := s.repository.UpdateSettings(s.ctx, domains.Settings{
		UserID: 1,
		Theme:  domains.ThemeDark,
	})
	s.NoError(err)

	err = s.repository.UpdateSettings(s.ctx, domains.Settings{
		UserID: 1,
		Theme:  domains.ThemeLight,
	})
	s.NoError(err)

	result, err := s.repository.GetSettingsByUserID(s.ctx, 1)
	s.NoError(err)
	s.NotNil(result)
	s.Equal(domains.ThemeLight, result.Theme)
}

func (s *RepositoryTestSuite) createTestData() {
	// Создаем тестовых пользователей (settings зависит от users через FK)
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

	now := time.Now().UTC()
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
			now,
		)
		s.NoError(err)
	}

	// Создаем настройки для первых двух пользователей
	// (третий без настроек — для тестов CreateSettings)
	testSettings := []struct {
		userID uint64
		theme  domains.ThemeType
	}{
		{1, domains.ThemeLight},
		{2, domains.ThemeDark},
	}

	for _, ts := range testSettings {
		_, err := s.tx.ExecContext(
			s.ctx,
			`INSERT INTO settings (user_id, theme, created_at, updated_at)
			 VALUES ($1, $2, $3, $3)`,
			ts.userID,
			ts.theme,
			now,
		)
		s.NoError(err)
	}
}
