//go:build integration

package reactions_test

import (
	"context"
	"database/sql"
	"os"
	"path"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/repositories/reactions"
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
	repository  *reactions.Repository
	logger      logging.Logger
}

func (s *RepositoryTestSuite) SetupSuite() {
	loadenv.Init("../../../.env")

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

	s.repository = reactions.New(tx, s.logger)
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

// --- Тесты справочника ---

func (s *RepositoryTestSuite) TestListReactions_ReturnsSeedInSortOrder() {
	got, err := s.repository.ListReactions(s.ctx)
	s.NoError(err)
	s.Len(got, 8, "seed вставляет 8 emoji")

	// Первый — 👍 (sort_order = 10)
	s.Equal("👍", got[0].Emoji)
	// Второй — ❤️ (sort_order = 20)
	s.Equal("❤️", got[1].Emoji)
	// 🔥 идёт перед 💯
	s.Equal("🔥", got[2].Emoji)
	s.Equal("💯", got[3].Emoji)
	// Последний — 😡 (sort_order = 80)
	s.Equal("😡", got[len(got)-1].Emoji)

	// Все ID положительные
	for _, r := range got {
		s.Greater(r.ID, uint64(0))
	}
}

func (s *RepositoryTestSuite) TestGetReactionByID_Found() {
	seed, err := s.repository.ListReactions(s.ctx)
	s.NoError(err)
	s.NotEmpty(seed)

	got, err := s.repository.GetReactionByID(s.ctx, seed[0].ID)
	s.NoError(err)
	s.NotNil(got)
	s.Equal(seed[0].ID, got.ID)
	s.Equal(seed[0].Emoji, got.Emoji)
}

func (s *RepositoryTestSuite) TestGetReactionByID_NotFound() {
	got, err := s.repository.GetReactionByID(s.ctx, 9999)
	s.Nil(got)
	s.ErrorIs(err, customerrors.ErrReactionNotFound)
}

// --- Тесты M2M ---

func (s *RepositoryTestSuite) TestAddMessageReaction_Success() {
	s.createTestData()
	reactionID := s.firstReactionID()

	dto := domains.MessageReactionDTO{
		MessageID:  1,
		UserID:     1,
		ReactionID: reactionID,
	}
	s.NoError(s.repository.AddMessageReaction(s.ctx, dto))

	// Строка появилась
	var count int
	err := s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND reaction_id = $3`,
		dto.MessageID, dto.UserID, dto.ReactionID,
	).Scan(&count)
	s.NoError(err)
	s.Equal(1, count)
}

func (s *RepositoryTestSuite) TestAddMessageReaction_Duplicate_ReturnsErrReactionAlreadyExists() {
	s.createTestData()
	reactionID := s.firstReactionID()

	dto := domains.MessageReactionDTO{
		MessageID:  1,
		UserID:     1,
		ReactionID: reactionID,
	}
	s.NoError(s.repository.AddMessageReaction(s.ctx, dto))

	err := s.repository.AddMessageReaction(s.ctx, dto)
	s.ErrorIs(err, customerrors.ErrReactionAlreadyExists)
}

func (s *RepositoryTestSuite) TestAddMessageReaction_MultipleReactionsFromSameUser() {
	// Один юзер может поставить несколько разных реакций на одно сообщение
	s.createTestData()
	seed, err := s.repository.ListReactions(s.ctx)
	s.NoError(err)
	s.GreaterOrEqual(len(seed), 2)

	s.NoError(s.repository.AddMessageReaction(s.ctx, domains.MessageReactionDTO{
		MessageID: 1, UserID: 1, ReactionID: seed[0].ID,
	}))
	s.NoError(s.repository.AddMessageReaction(s.ctx, domains.MessageReactionDTO{
		MessageID: 1, UserID: 1, ReactionID: seed[1].ID,
	}))

	var count int
	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM message_reactions WHERE message_id = 1 AND user_id = 1`,
	).Scan(&count)
	s.NoError(err)
	s.Equal(2, count)
}

func (s *RepositoryTestSuite) TestRemoveMessageReaction_ExistingRow_ReturnsNil() {
	s.createTestData()
	reactionID := s.firstReactionID()

	dto := domains.MessageReactionDTO{
		MessageID: 1, UserID: 1, ReactionID: reactionID,
	}
	s.NoError(s.repository.AddMessageReaction(s.ctx, dto))

	s.NoError(s.repository.RemoveMessageReaction(s.ctx, dto))

	var count int
	err := s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM message_reactions WHERE message_id = $1 AND user_id = $2 AND reaction_id = $3`,
		dto.MessageID, dto.UserID, dto.ReactionID,
	).Scan(&count)
	s.NoError(err)
	s.Equal(0, count)
}

func (s *RepositoryTestSuite) TestRemoveMessageReaction_MissingRow_ReturnsErrReactionNotSet() {
	s.createTestData()
	reactionID := s.firstReactionID()

	err := s.repository.RemoveMessageReaction(s.ctx, domains.MessageReactionDTO{
		MessageID: 1, UserID: 1, ReactionID: reactionID,
	})
	s.ErrorIs(err, customerrors.ErrReactionNotSet)
}

func (s *RepositoryTestSuite) TestListReactionsForMessages_EmptyInput() {
	got, err := s.repository.ListReactionsForMessages(s.ctx, nil)
	s.NoError(err)
	s.NotNil(got)
	s.Empty(got)

	got, err = s.repository.ListReactionsForMessages(s.ctx, []uint64{})
	s.NoError(err)
	s.NotNil(got)
	s.Empty(got)
}

func (s *RepositoryTestSuite) TestListReactionsForMessages_AggregatesByMessageAndReaction() {
	s.createTestData()
	seed, err := s.repository.ListReactions(s.ctx)
	s.NoError(err)
	s.GreaterOrEqual(len(seed), 2)

	// Сообщение 1: reaction[0] от user 1 и user 2, reaction[1] от user 1
	// Сообщение 2: reaction[0] от user 3
	// Сообщение 3: реакций нет — в ответе не должно быть ключа
	s.NoError(s.repository.AddMessageReaction(s.ctx, domains.MessageReactionDTO{
		MessageID: 1, UserID: 1, ReactionID: seed[0].ID,
	}))
	s.NoError(s.repository.AddMessageReaction(s.ctx, domains.MessageReactionDTO{
		MessageID: 1, UserID: 2, ReactionID: seed[0].ID,
	}))
	s.NoError(s.repository.AddMessageReaction(s.ctx, domains.MessageReactionDTO{
		MessageID: 1, UserID: 1, ReactionID: seed[1].ID,
	}))
	s.NoError(s.repository.AddMessageReaction(s.ctx, domains.MessageReactionDTO{
		MessageID: 2, UserID: 3, ReactionID: seed[0].ID,
	}))

	got, err := s.repository.ListReactionsForMessages(s.ctx, []uint64{1, 2, 3})
	s.NoError(err)

	// Сообщение 1: две группы (по одной на каждую реакцию), порядок — по sort_order
	msg1 := got[1]
	s.Len(msg1, 2, "две разные реакции на сообщении 1")
	s.Equal(seed[0].Emoji, msg1[0].Reaction.Emoji)
	s.ElementsMatch([]uint64{1, 2}, msg1[0].UserIDs)
	s.Equal(seed[1].Emoji, msg1[1].Reaction.Emoji)
	s.Equal([]uint64{1}, msg1[1].UserIDs)

	// Сообщение 2: одна реакция от одного юзера
	msg2 := got[2]
	s.Len(msg2, 1)
	s.Equal(seed[0].Emoji, msg2[0].Reaction.Emoji)
	s.Equal([]uint64{3}, msg2[0].UserIDs)

	// Сообщение 3 — нет в мапе
	_, ok := got[3]
	s.False(ok, "сообщение без реакций в мапу не попадает")
}

func (s *RepositoryTestSuite) TestMessageReactions_CascadeOnUserDelete() {
	s.createTestData()
	reactionID := s.firstReactionID()

	s.NoError(s.repository.AddMessageReaction(s.ctx, domains.MessageReactionDTO{
		MessageID: 1, UserID: 2, ReactionID: reactionID,
	}))

	_, err := s.tx.ExecContext(s.ctx, `DELETE FROM users WHERE id = 2`)
	s.NoError(err)

	var count int
	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM message_reactions WHERE user_id = 2`,
	).Scan(&count)
	s.NoError(err)
	s.Equal(0, count, "CASCADE удалил реакции удалённого юзера")
}

func (s *RepositoryTestSuite) TestMessageReactions_CascadeOnMessageDelete() {
	s.createTestData()
	reactionID := s.firstReactionID()

	s.NoError(s.repository.AddMessageReaction(s.ctx, domains.MessageReactionDTO{
		MessageID: 1, UserID: 1, ReactionID: reactionID,
	}))

	// Удаляем связанные статусы (FK от messages), потом само сообщение
	_, err := s.tx.ExecContext(s.ctx, `DELETE FROM messages_statuses WHERE message_id = 1`)
	s.NoError(err)
	_, err = s.tx.ExecContext(s.ctx, `DELETE FROM messages WHERE id = 1`)
	s.NoError(err)

	var count int
	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM message_reactions WHERE message_id = 1`,
	).Scan(&count)
	s.NoError(err)
	s.Equal(0, count, "CASCADE удалил реакции удалённого сообщения")
}

// --- Helpers ---

func (s *RepositoryTestSuite) firstReactionID() uint64 {
	seed, err := s.repository.ListReactions(s.ctx)
	s.NoError(err)
	s.NotEmpty(seed)
	return seed[0].ID
}

func (s *RepositoryTestSuite) createTestData() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()
}

func (s *RepositoryTestSuite) createTestUsers() {
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
			u.id, u.username, u.email, u.emailConfirmed, u.password, time.Now().UTC(),
		)
		s.NoError(err)
	}
}

func (s *RepositoryTestSuite) createTestChats() {
	_, err := s.tx.ExecContext(
		s.ctx,
		`INSERT INTO chats (id, title, description, type, created_at, updated_at)
		 VALUES (1, 'General', 'General chat', 'group', $1, $1)`,
		time.Now().UTC(),
	)
	s.NoError(err)
}

func (s *RepositoryTestSuite) createTestChatMembers() {
	members := []uint64{1, 2, 3}
	for _, uid := range members {
		_, err := s.tx.ExecContext(
			s.ctx,
			`INSERT INTO chats_members (chat_id, user_id, created_at, updated_at)
			 VALUES (1, $1, $2, $2)`,
			uid, time.Now().UTC(),
		)
		s.NoError(err)
	}
}

func (s *RepositoryTestSuite) createTestMessages() {
	messages := []struct {
		id       uint64
		senderID uint64
		text     string
	}{
		{1, 1, "Hello!"},
		{2, 2, "Hi there!"},
		{3, 3, "Good day!"},
	}

	for _, m := range messages {
		_, err := s.tx.ExecContext(
			s.ctx,
			`INSERT INTO messages (id, chat_id, sender_id, text, created_at, updated_at)
			 VALUES ($1, 1, $2, $3, $4, $4)`,
			m.id, m.senderID, m.text, time.Now().UTC(),
		)
		s.NoError(err)

		_, err = s.tx.ExecContext(
			s.ctx,
			`INSERT INTO messages_statuses (message_id, user_id, is_read)
			 VALUES ($1, $2, false)`,
			m.id, m.senderID,
		)
		s.NoError(err)
	}
}
