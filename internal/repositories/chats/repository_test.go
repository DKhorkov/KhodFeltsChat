//go:build integration

package chats_test

import (
	"context"
	"database/sql"
	"os"
	"path"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/repositories/chats"
	"github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/loadenv"
	"github.com/DKhorkov/libs/logging"
	"github.com/DKhorkov/libs/pointers"
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
	repository  *chats.Repository
	logger      logging.Logger
}

func (s *RepositoryTestSuite) SetupSuite() {
	// Инициализируем переменные окружения
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
	// Применяем миграции
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

	// Создаем репозиторий с транзакцией
	s.repository = chats.New(tx, s.logger)
}

func (s *RepositoryTestSuite) TearDownTest() {
	// Откатываем транзакцию
	if s.tx != nil {
		s.NoError(s.tx.Rollback())
	}

	// Откатываем миграции
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

func (s *RepositoryTestSuite) TestGetChatMembers_Success() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	// Тест: Получение участников чата
	chatID := uint64(1) // General Chat

	members, err := s.repository.GetChatMembers(s.ctx, chatID)
	s.NoError(err)
	s.NotNil(members)
	s.Equal(5, len(members)) // В General Chat 5 участников

	// Проверяем структуру участников
	for _, member := range members {
		s.NotZero(member.ID)
		s.NotEmpty(member.Username)
		s.NotEmpty(member.Email)
		s.NotEmpty(member.Password)
		s.NotZero(member.CreatedAt)
		s.NotZero(member.UpdatedAt)
	}

	// Проверяем конкретных участников
	memberIDs := make(map[uint64]bool)
	for _, member := range members {
		memberIDs[member.ID] = true
	}

	// Должны быть пользователи 1, 2, 3, 4, 5
	s.True(memberIDs[1], "User 1 should be a member")
	s.True(memberIDs[2], "User 2 should be a member")
	s.True(memberIDs[3], "User 3 should be a member")
	s.True(memberIDs[4], "User 4 should be a member")
	s.True(memberIDs[5], "User 5 should be a member")
	s.False(memberIDs[6], "User 6 should not be a member")
}

func (s *RepositoryTestSuite) TestGetChatMembers_PrivateChat() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	// Тест: Получение участников приватного чата
	chatID := uint64(2) // Private Chat 1-4

	members, err := s.repository.GetChatMembers(s.ctx, chatID)
	s.NoError(err)
	s.NotNil(members)
	s.Equal(2, len(members)) // В приватном чате 2 участника

	memberIDs := make(map[uint64]bool)
	for _, member := range members {
		memberIDs[member.ID] = true
	}

	// Должны быть только пользователи 1 и 4
	s.True(memberIDs[1], "User 1 should be a member")
	s.True(memberIDs[4], "User 4 should be a member")
	s.False(memberIDs[2], "User 2 should not be a member")
	s.False(memberIDs[3], "User 3 should not be a member")
}

func (s *RepositoryTestSuite) TestGetChatMembers_EmptyChat() {
	// Тест: Получение участников пустого чата
	chatID := uint64(6) // Empty Chat

	members, err := s.repository.GetChatMembers(s.ctx, chatID)
	s.NoError(err)
	s.Nil(members) // Должен вернуться nil
}

func (s *RepositoryTestSuite) TestGetChatMembers_NonExistentChat() {
	// Тест: Получение участников несуществующего чата
	members, err := s.repository.GetChatMembers(s.ctx, 999)
	s.NoError(err)
	s.Nil(members) // Должен вернуться nil
}

func (s *RepositoryTestSuite) TestGetUserChats_WithoutPagination() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	// Тест: Получение всех чатов пользователя без пагинации
	userID := uint64(1) // John Doe

	chats, err := s.repository.GetUserChats(s.ctx, userID, nil)
	s.NoError(err)
	s.NotNil(chats)

	// У John должно быть 3 чата: General Chat (1), Private Chat 1-4 (2), Project Chat (3)
	s.Equal(3, len(chats))

	// Проверяем сортировку: чаты должны быть отсортированы по времени последнего сообщения
	// или updated_at, если сообщений нет (в порядке убывания)
	expectedOrder := []uint64{3, 2, 1} // General, Project, Private (по времени последних сообщений)
	for i, chat := range chats {
		s.Equal(expectedOrder[i], chat.ID,
			"Chat at position %d should have ID %d, got %d", i, expectedOrder[i], chat.ID)
	}
}

func (s *RepositoryTestSuite) TestGetUserChats_WithLimit() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	// Тест: Получение чатов пользователя с лимитом
	userID := uint64(1)
	pagination := &domains.Pagination{
		Limit: pointers.New[uint64](2),
	}

	chats, err := s.repository.GetUserChats(s.ctx, userID, pagination)
	s.NoError(err)
	s.NotNil(chats)
	s.Equal(2, len(chats))

	// Должны вернуться первые 2 чата: General Chat и Project Chat
	s.Equal(uint64(3), chats[0].ID)
	s.Equal(uint64(2), chats[1].ID)
}

func (s *RepositoryTestSuite) TestGetUserChats_WithOffset() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	// Тест: Получение чатов пользователя с оффсетом
	userID := uint64(1)
	pagination := &domains.Pagination{
		Limit:  pointers.New[uint64](1),
		Offset: pointers.New[uint64](1),
	}

	chats, err := s.repository.GetUserChats(s.ctx, userID, pagination)
	s.NoError(err)
	s.NotNil(chats)
	s.Equal(1, len(chats))

	s.Equal(uint64(2), chats[0].ID)
}

func (s *RepositoryTestSuite) TestGetUserChats_UnreadCount_Zero_WhenAllRead() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	userID := uint64(1)
	chatID := uint64(1)

	s.createMessage(chatID, 2)
	s.createMessageStatus(1, userID, true, false)

	userChats, err := s.repository.GetUserChats(s.ctx, userID, nil)
	s.NoError(err)

	chat := s.findChatByID(userChats, chatID)
	s.NotNil(chat)
	s.Equal(uint64(0), chat.UnreadCount, "UnreadCount must be 0 when all messages are read")
}

func (s *RepositoryTestSuite) TestGetUserChats_UnreadCount_CountsUnreadOnly() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	userID := uint64(1)
	chatID := uint64(1)

	s.createMessage(chatID, 2)
	s.createMessage(chatID, 2)
	s.createMessage(chatID, 2)

	s.createMessageStatus(1, userID, false, false) // unread
	s.createMessageStatus(2, userID, false, false) // unread
	s.createMessageStatus(3, userID, true, false)  // read

	userChats, err := s.repository.GetUserChats(s.ctx, userID, nil)
	s.NoError(err)

	chat := s.findChatByID(userChats, chatID)
	s.NotNil(chat)
	s.Equal(uint64(2), chat.UnreadCount, "UnreadCount must count only unread non-deleted messages")
}

func (s *RepositoryTestSuite) TestGetUserChats_UnreadCount_ExcludesDeletedStatuses() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	userID := uint64(1)
	chatID := uint64(1)

	s.createMessage(chatID, 2)
	s.createMessage(chatID, 2)

	s.createMessageStatus(1, userID, false, false) // unread, not deleted
	s.createMessageStatus(2, userID, false, true)  // unread, but deleted for user

	userChats, err := s.repository.GetUserChats(s.ctx, userID, nil)
	s.NoError(err)

	chat := s.findChatByID(userChats, chatID)
	s.NotNil(chat)
	s.Equal(uint64(1), chat.UnreadCount, "UnreadCount must exclude deleted statuses")
}

func (s *RepositoryTestSuite) TestGetUserChats_UserWithNoChats() {
	// Тест: Получение чатов пользователя без чатов
	// Создаем пользователя без чатов
	_, err := s.tx.ExecContext(
		s.ctx,
		`INSERT INTO users (id, username, email, password) 
		 VALUES (100, 'nochats', 'nochats@example.com', 'hashed')`,
	)
	s.NoError(err)

	chats, err := s.repository.GetUserChats(s.ctx, 100, nil)
	s.NoError(err)
	s.Nil(chats) // Должен вернуться nil
}

func (s *RepositoryTestSuite) TestGetUserChats_SortingOrder() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	// Тест: Проверка правильности сортировки чатов

	// User 4 (Alice) участвует в чатах: General Chat (1), Private Chat 1-4 (2), Team Chat (5)
	// Сообщения:
	// - General Chat: последнее сообщение 30 минут назад (от user 5)
	// - Private Chat 1-4: нет сообщений, используем updated_at
	// - Team Chat: последнее сообщение 2 часа назад (от user 6)

	chats, err := s.repository.GetUserChats(s.ctx, 4, nil)
	s.NoError(err)
	s.NotNil(chats)
	s.Equal(3, len(chats))

	s.Equal(uint64(5), chats[0].ID)
	s.Equal(uint64(2), chats[1].ID)
	s.Equal(uint64(1), chats[2].ID)
}

func (s *RepositoryTestSuite) TestCreateChat_Success() {
	s.createTestUsers()

	// Тест: Успешное создание чата
	chat := domains.Chat{
		Title:       pointers.New("New Group Chat"),
		Description: pointers.New("Chat for testing"),
		Type:        "group",
		Members: []domains.User{
			{ID: 1},
			{ID: 2},
			{ID: 3},
		},
	}

	chatID, err := s.repository.CreateChat(s.ctx, chat)
	s.NoError(err)
	s.Greater(chatID, uint64(0))

	// Проверяем, что чат создан
	var (
		dbTitle       string
		dbDescription string
		dbType        string
	)

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT title, description, type FROM chats WHERE id = $1`,
		chatID,
	).Scan(&dbTitle, &dbDescription, &dbType)
	s.NoError(err)

	s.Equal(*chat.Title, dbTitle)
	s.Equal(*chat.Description, dbDescription)
	s.Equal(chat.Type, domains.ChatType(dbType))

	// Проверяем участников чата
	rows, err := s.tx.QueryContext(
		s.ctx,
		`SELECT user_id FROM chats_members WHERE chat_id = $1 ORDER BY user_id`,
		chatID,
	)
	s.NoError(err)

	defer rows.Close()

	var memberIDs []uint64

	for rows.Next() {
		var userID uint64

		err = rows.Scan(&userID)
		s.NoError(err)

		memberIDs = append(memberIDs, userID)
	}

	s.NoError(rows.Err())

	s.Equal(3, len(memberIDs))
	s.Equal(uint64(1), memberIDs[0])
	s.Equal(uint64(2), memberIDs[1])
	s.Equal(uint64(3), memberIDs[2])
}

func (s *RepositoryTestSuite) TestCreateChat_EmptyTitle() {
	s.createTestUsers()

	// Тест: Создание чата с пустым заголовком
	chat := domains.Chat{
		Title:       pointers.New(""),
		Description: pointers.New("Chat with empty title"),
		Type:        "group",
		Members: []domains.User{
			{ID: 1},
		},
	}

	chatID, err := s.repository.CreateChat(s.ctx, chat)
	s.NoError(err)
	s.Greater(chatID, uint64(0))

	// Проверяем, что заголовок пустой
	var dbTitle string

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT title FROM chats WHERE id = $1`,
		chatID,
	).Scan(&dbTitle)
	s.NoError(err)
	s.Equal("", dbTitle)
}

func (s *RepositoryTestSuite) TestCreateChat_NoMembers() {
	s.createTestUsers()

	// Тест: Создание чата без участников
	chat := domains.Chat{
		Title:       pointers.New("Empty Chat"),
		Description: pointers.New("Chat with no members"),
		Type:        "group",
		Members:     []domains.User{}, // Пустой список участников
	}

	chatID, err := s.repository.CreateChat(s.ctx, chat)
	s.Error(err)
	s.Empty(chatID)
}

func (s *RepositoryTestSuite) TestCreateChat_InvalidMember() {
	// Тест: Попытка создать чат с несуществующим участником
	chat := domains.Chat{
		Title:       pointers.New("Invalid Chat"),
		Description: pointers.New("Chat with invalid member"),
		Type:        "group",
		Members: []domains.User{
			{ID: 1},
			{ID: 999}, // Несуществующий пользователь
		},
	}

	chatID, err := s.repository.CreateChat(s.ctx, chat)
	s.Error(err) // Должна быть ошибка foreign key constraint
	s.Equal(uint64(0), chatID)
}

func (s *RepositoryTestSuite) TestCreateChat_PrivateChat() {
	s.createTestUsers()

	// Тест: Создание приватного чата
	chat := domains.Chat{
		Title:       pointers.New(""), // Для приватных чатов заголовок может быть пустым
		Description: pointers.New(""),
		Type:        "private",
		Members: []domains.User{
			{ID: 1},
			{ID: 5},
		},
	}

	chatID, err := s.repository.CreateChat(s.ctx, chat)
	s.NoError(err)
	s.Greater(chatID, uint64(0))

	// Проверяем тип чата
	var dbType string

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT type FROM chats WHERE id = $1`,
		chatID,
	).Scan(&dbType)
	s.NoError(err)
	s.Equal("private", dbType)

	// Проверяем участников
	var count int

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM chats_members WHERE chat_id = $1`,
		chatID,
	).Scan(&count)
	s.NoError(err)
	s.Equal(2, count)
}

func (s *RepositoryTestSuite) TestGetChatByID_Success() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	// Тест: Получение чата по ID
	chatID := uint64(1) // General Chat
	userID := uint64(1)

	chat, err := s.repository.GetChatByID(s.ctx, chatID, userID)
	s.NoError(err)
	s.NotNil(chat)

	s.Equal(chatID, chat.ID)
	s.Equal("General Chat", *chat.Title)
	s.Equal("General discussion for everyone", *chat.Description)
	s.Equal(domains.ChatTypeGroup, chat.Type)
	s.NotZero(chat.CreatedAt)
	s.NotZero(chat.UpdatedAt)

	// Проверяем, что поля Members и Messages не заполнены, UnreadCount = 0 (нет сообщений)
	s.Nil(chat.Members)
	s.Nil(chat.Messages)
	s.Zero(chat.UnreadCount)
}

func (s *RepositoryTestSuite) TestGetChatByID_PrivateChat() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	// Тест: Получение приватного чата
	chatID := uint64(2) // Private Chat 1-4
	userID := uint64(1)

	chat, err := s.repository.GetChatByID(s.ctx, chatID, userID)
	s.NoError(err)
	s.NotNil(chat)

	s.Equal(chatID, chat.ID)
	s.Equal("Private Chat 1-4", *chat.Title)
	s.Equal("Private between John and Alice", *chat.Description)
	s.Equal(domains.ChatTypePrivate, chat.Type)
}

func (s *RepositoryTestSuite) TestGetChatByID_NotFound() {
	// Тест: Попытка получить несуществующий чат
	chat, err := s.repository.GetChatByID(s.ctx, 999, 0)
	s.Error(err)
	s.Nil(chat)
	s.Contains(err.Error(), "no rows")
}

func (s *RepositoryTestSuite) TestGetChatByID_ZeroID() {
	// Тест: Попытка получить чат с ID = 0
	chat, err := s.repository.GetChatByID(s.ctx, 0, 0)
	s.Error(err)
	s.Nil(chat)
}

func (s *RepositoryTestSuite) TestGetChatByID_UnreadCount_CountsUnreadOnly() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	userID := uint64(1)
	chatID := uint64(1)

	s.createMessage(chatID, 2)
	s.createMessage(chatID, 2)
	s.createMessage(chatID, 2)

	s.createMessageStatus(1, userID, false, false) // unread
	s.createMessageStatus(2, userID, false, false) // unread
	s.createMessageStatus(3, userID, true, false)  // read

	chat, err := s.repository.GetChatByID(s.ctx, chatID, userID)
	s.NoError(err)
	s.NotNil(chat)
	s.Equal(uint64(2), chat.UnreadCount, "UnreadCount must count only unread non-deleted messages")
}

func (s *RepositoryTestSuite) TestGetChatByID_UnreadCount_ExcludesDeletedStatuses() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	userID := uint64(1)
	chatID := uint64(1)

	s.createMessage(chatID, 2)
	s.createMessage(chatID, 2)

	s.createMessageStatus(1, userID, false, false) // unread, not deleted
	s.createMessageStatus(2, userID, false, true)  // unread, but deleted for user

	chat, err := s.repository.GetChatByID(s.ctx, chatID, userID)
	s.NoError(err)
	s.NotNil(chat)
	s.Equal(uint64(1), chat.UnreadCount, "UnreadCount must exclude deleted statuses")
}

func (s *RepositoryTestSuite) TestGetChatByID_UnreadCount_ZeroForUserIDZero() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	chatID := uint64(1)

	s.createMessage(chatID, 2)
	s.createMessageStatus(1, 1, false, false) // unread для юзера 1

	// userID=0 используется для existence-чеков; подзапрос вернёт 0 — нет статусов для user_id=0.
	chat, err := s.repository.GetChatByID(s.ctx, chatID, 0)
	s.NoError(err)
	s.NotNil(chat)
	s.Zero(chat.UnreadCount, "UnreadCount must be 0 when called with userID=0")
}

func (s *RepositoryTestSuite) TestPrivateChatExists_True() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	// Тест: Проверка существования приватного чата (должен существовать)
	members := []domains.User{
		{ID: 1},
		{ID: 4},
	}

	exists, err := s.repository.PrivateChatExists(s.ctx, members)
	s.NoError(err)
	s.True(exists, "Private chat between users 1 and 4 should exist")
}

func (s *RepositoryTestSuite) TestPrivateChatExists_False() {
	// Тест: Проверка существования приватного чата (не должен существовать)
	members := []domains.User{
		{ID: 1},
		{ID: 5},
	}

	exists, err := s.repository.PrivateChatExists(s.ctx, members)
	s.NoError(err)
	s.False(exists, "Private chat between users 1 and 5 should not exist")
}

func (s *RepositoryTestSuite) TestPrivateChatExists_ThreeUsers() {
	// Тест: Проверка существования приватного чата с тремя пользователями
	members := []domains.User{
		{ID: 1},
		{ID: 2},
		{ID: 3},
	}

	exists, err := s.repository.PrivateChatExists(s.ctx, members)
	s.NoError(err)
	s.False(exists, "Private chat with 3 users should not exist (only 2-user private chats)")
}

func (s *RepositoryTestSuite) TestPrivateChatExists_SingleUser() {
	// Тест: Проверка существования приватного чата с одним пользователем
	members := []domains.User{
		{ID: 1},
	}

	exists, err := s.repository.PrivateChatExists(s.ctx, members)
	s.NoError(err)
	s.False(exists, "Private chat with 1 user should not exist")
}

func (s *RepositoryTestSuite) TestPrivateChatExists_EmptyList() {
	// Тест: Проверка существования приватного чата с пустым списком пользователей
	members := []domains.User{}

	exists, err := s.repository.PrivateChatExists(s.ctx, members)
	s.NoError(err)
	s.False(exists, "Private chat with empty member list should not exist")
}

func (s *RepositoryTestSuite) TestPrivateChatExists_DifferentOrder() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	// Тест: Проверка существования приватного чата (порядок пользователей не важен)
	members := []domains.User{
		{ID: 4},
		{ID: 1},
	}

	exists, err := s.repository.PrivateChatExists(s.ctx, members)
	s.NoError(err)
	s.True(exists, "Private chat should exist regardless of user order")
}

func (s *RepositoryTestSuite) createTestUsers() {
	// Создаем пользователей
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
		{5, "charlie_brown", "charlie@example.com", true, "$2a$10$hashedpassword5"},
		{6, "diana_ross", "diana@example.com", false, "$2a$10$hashedpassword6"},
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

func (s *RepositoryTestSuite) createTestChats() {
	// Создаем чаты
	chats := []struct {
		id          uint64
		title       string
		description string
		chatType    string
	}{
		{1, "General Chat", "General discussion for everyone", "group"},
		{2, "Private Chat 1-4", "Private between John and Alice", "private"},
		{3, "Project Chat", "Project discussions", "group"},
		{4, "Private Chat 2-3", "Private between Jane and Bob", "private"},
		{5, "Team Chat", "Team collaboration", "group"},
		{6, "Empty Chat", "Chat with no members", "group"},
	}

	for _, c := range chats {
		_, err := s.tx.ExecContext(
			s.ctx,
			`INSERT INTO chats (id, title, description, type, created_at, updated_at) 
			 VALUES ($1, $2, $3, $4, $5, $5)`,
			c.id,
			c.title,
			c.description,
			c.chatType,
			time.Now().UTC(),
		)
		s.NoError(err)
	}
}

func (s *RepositoryTestSuite) createTestChatMembers() {
	// Создаем участников чатов
	chatMembers := []struct {
		chatID uint64
		userID uint64
	}{
		// General Chat (id=1) - много участников
		{1, 1},
		{1, 2},
		{1, 3},
		{1, 4},
		{1, 5},

		// Private Chat 1-4 (id=2) - приватный чат
		{2, 1},
		{2, 4},

		// Project Chat (id=3)
		{3, 1},
		{3, 2},
		{3, 3},

		// Private Chat 2-3 (id=4)
		{4, 2},
		{4, 3},

		// Team Chat (id=5)
		{5, 4},
		{5, 5},
		{5, 6},
	}

	for _, cm := range chatMembers {
		_, err := s.tx.ExecContext(
			s.ctx,
			`INSERT INTO chats_members (chat_id, user_id, created_at, updated_at)
			 VALUES ($1, $2, $3, $3)`,
			cm.chatID,
			cm.userID,
			time.Now().UTC(),
		)
		s.NoError(err)
	}
}

func (s *RepositoryTestSuite) createMessage(chatID, senderID uint64) {
	_, err := s.tx.ExecContext(
		s.ctx,
		`INSERT INTO messages (chat_id, sender_id, text, created_at, updated_at)
		 VALUES ($1, $2, 'test message', $3, $3)`,
		chatID,
		senderID,
		time.Now().UTC(),
	)
	s.NoError(err)
}

func (s *RepositoryTestSuite) createMessageStatus(messageID, userID uint64, isRead, isDeleted bool) {
	_, err := s.tx.ExecContext(
		s.ctx,
		`INSERT INTO messages_statuses (message_id, user_id, is_read, is_deleted)
		 VALUES ($1, $2, $3, $4)`,
		messageID,
		userID,
		isRead,
		isDeleted,
	)
	s.NoError(err)
}

func (s *RepositoryTestSuite) findChatByID(userChats []domains.Chat, chatID uint64) *domains.Chat {
	for i := range userChats {
		if userChats[i].ID == chatID {
			return &userChats[i]
		}
	}

	return nil
}
