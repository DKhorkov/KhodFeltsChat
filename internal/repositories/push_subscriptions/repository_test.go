//go:build integration

package push_subscriptions_test

import (
	"context"
	"database/sql"
	"os"
	"path"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	pushsubscriptions "github.com/DKhorkov/kfc/internal/repositories/push_subscriptions"
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
	repository  *pushsubscriptions.Repository
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

	s.repository = pushsubscriptions.New(tx)

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

func (s *RepositoryTestSuite) TestCreatePushSubscription_Success() {
	subscription := domains.PushSubscription{
		UserID:   2,
		Endpoint: "https://fcm.googleapis.com/fcm/send/new",
		EncryptionKey:   "new-p256dh",
		Auth:     "new-auth",
	}

	id, err := s.repository.CreatePushSubscription(s.ctx, subscription)
	s.NoError(err)
	s.NotZero(id)

	// Проверяем, что подписка была создана
	subscriptions, err := s.repository.GetPushSubscriptionsByUserID(s.ctx, 2)
	s.NoError(err)
	s.NotEmpty(subscriptions)

	found := false
	for _, sub := range subscriptions {
		if sub.Endpoint == "https://fcm.googleapis.com/fcm/send/new" {
			found = true
			s.Equal(uint64(2), sub.UserID)
			s.Equal("new-p256dh", sub.EncryptionKey)
			s.Equal("new-auth", sub.Auth)
			s.NotZero(sub.CreatedAt)
		}
	}

	s.True(found)
}

func (s *RepositoryTestSuite) TestGetPushSubscriptionsByUserID_Success() {
	subscriptions, err := s.repository.GetPushSubscriptionsByUserID(s.ctx, 1)
	s.NoError(err)
	s.Len(subscriptions, 2)

	for _, sub := range subscriptions {
		s.Equal(uint64(1), sub.UserID)
		s.NotZero(sub.ID)
		s.NotEmpty(sub.Endpoint)
		s.NotEmpty(sub.EncryptionKey)
		s.NotEmpty(sub.Auth)
		s.NotZero(sub.CreatedAt)
	}
}

func (s *RepositoryTestSuite) TestGetPushSubscriptionsByUserID_NoSubscriptions() {
	subscriptions, err := s.repository.GetPushSubscriptionsByUserID(s.ctx, 3)
	s.NoError(err)
	s.Empty(subscriptions)
}

func (s *RepositoryTestSuite) TestDeletePushSubscription_Success() {
	// Получаем подписки пользователя 1
	subscriptions, err := s.repository.GetPushSubscriptionsByUserID(s.ctx, 1)
	s.NoError(err)
	s.NotEmpty(subscriptions)

	// Удаляем первую подписку
	err = s.repository.DeletePushSubscription(s.ctx, subscriptions[0].ID)
	s.NoError(err)

	// Проверяем, что осталась только одна подписка
	remaining, err := s.repository.GetPushSubscriptionsByUserID(s.ctx, 1)
	s.NoError(err)
	s.Len(remaining, 1)
}

func (s *RepositoryTestSuite) TestDeletePushSubscription_NotFound() {
	// Удаление несуществующей подписки не возвращает ошибку
	err := s.repository.DeletePushSubscription(s.ctx, 999)
	s.NoError(err)
}

func (s *RepositoryTestSuite) createTestData() {
	// Создаем тестовых пользователей (push_subscriptions зависит от users через FK)
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

	// Создаем тестовые подписки для пользователя 1
	testSubscriptions := []struct {
		userID        uint64
		endpoint      string
		encryptionKey string
		auth          string
	}{
		{1, "https://fcm.googleapis.com/fcm/send/test1", "p256dh-1", "auth-1"},
		{1, "https://fcm.googleapis.com/fcm/send/test2", "p256dh-2", "auth-2"},
	}

	for _, ts := range testSubscriptions {
		_, err := s.tx.ExecContext(
			s.ctx,
			`INSERT INTO push_subscriptions (user_id, endpoint, encryption_key, auth, created_at)
			 VALUES ($1, $2, $3, $4, $5)`,
			ts.userID,
			ts.endpoint,
			ts.encryptionKey,
			ts.auth,
			now,
		)
		s.NoError(err)
	}
}
