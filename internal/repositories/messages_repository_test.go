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
	"github.com/DKhorkov/libs/pointers"
	"github.com/pressly/goose"
	"github.com/stretchr/testify/suite"
)

func TestMessagesRepositoryTestSuite(t *testing.T) {
	suite.Run(t, new(MessagesRepositoryTestSuite))
}

type MessagesRepositoryTestSuite struct {
	suite.Suite

	cwd         string
	ctx         context.Context
	dbConnector postgresql.Connector
	pool        *sql.DB
	tx          postgresql.Transaction
	repository  *repositories.MessagesRepository
}

func (s *MessagesRepositoryTestSuite) SetupSuite() {
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

func (s *MessagesRepositoryTestSuite) SetupTest() {
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
	s.repository = repositories.NewMessagesRepository(tx)
}

func (s *MessagesRepositoryTestSuite) TearDownTest() {
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

func (s *MessagesRepositoryTestSuite) TearDownSuite() {
	if s.dbConnector != nil {
		s.NoError(s.dbConnector.Close())
	}
}

func (s *MessagesRepositoryTestSuite) TestSaveMessage_Success() {
	// Создаем тестовые данные
	s.createTestUsers()
	s.createTestChats()

	// Тест: Успешное сохранение сообщения
	message := domains.Message{
		ChatID: 1,
		Sender: domains.User{
			ID: 1,
		},
		Text: "New test message",
	}

	messageID, err := s.repository.SaveMessage(s.ctx, message)
	s.NoError(err)
	s.Greater(messageID, uint64(0))

	// Проверяем, что сообщение сохранено
	var (
		dbChatID   uint64
		dbSenderID uint64
		dbText     string
	)

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT chat_id, sender_id, text FROM messages WHERE id = $1`,
		messageID,
	).Scan(&dbChatID, &dbSenderID, &dbText)
	s.NoError(err)

	s.Equal(message.ChatID, dbChatID)
	s.Equal(message.Sender.ID, dbSenderID)
	s.Equal(message.Text, dbText)
}

func (s *MessagesRepositoryTestSuite) TestSaveMessage_EmptyText() {
	s.createTestUsers()
	s.createTestChats()

	// Тест: Сохранение сообщения с пустым текстом
	message := domains.Message{
		ChatID: 1,
		Sender: domains.User{
			ID: 1,
		},
		Text: "", // Пустой текст
	}

	messageID, err := s.repository.SaveMessage(s.ctx, message)
	s.NoError(err)
	s.Greater(messageID, uint64(0))

	// Проверяем, что текст пустой
	var dbText string

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT text FROM messages WHERE id = $1`,
		messageID,
	).Scan(&dbText)
	s.NoError(err)
	s.Equal("", dbText)
}

func (s *MessagesRepositoryTestSuite) TestSaveMessage_LongText() {
	s.createTestUsers()
	s.createTestChats()

	// Тест: Сохранение сообщения с длинным текстом
	longText := `Это очень длинное сообщение, которое содержит много текста. 
	Оно должно проверить, что длинные тексты корректно сохраняются в базе данных.
	Сообщение может содержать специальные символы: !@#$%^&*()_+-=[]{}|;:,.<>?
	А также эмодзи: 😀🎉🔥
	И даже несколько строк.
	
	Проверяем работу с разными символами и форматами.`

	message := domains.Message{
		ChatID: 1,
		Sender: domains.User{
			ID: 1,
		},
		Text: longText,
	}

	messageID, err := s.repository.SaveMessage(s.ctx, message)
	s.NoError(err)
	s.Greater(messageID, uint64(0))

	// Проверяем, что длинный текст сохранен
	var dbText string

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT text FROM messages WHERE id = $1`,
		messageID,
	).Scan(&dbText)
	s.NoError(err)
	s.Equal(longText, dbText)
}

func (s *MessagesRepositoryTestSuite) TestSaveMessage_InvalidChat() {
	// Тест: Попытка сохранить сообщение в несуществующий чат
	message := domains.Message{
		ChatID: 999, // Несуществующий чат
		Sender: domains.User{
			ID: 1,
		},
		Text: "Test message",
	}

	messageID, err := s.repository.SaveMessage(s.ctx, message)
	s.Error(err) // Должна быть ошибка foreign key constraint
	s.Equal(uint64(0), messageID)
}

func (s *MessagesRepositoryTestSuite) TestSaveMessage_InvalidSender() {
	// Тест: Попытка сохранить сообщение от несуществующего отправителя
	message := domains.Message{
		ChatID: 1,
		Sender: domains.User{
			ID: 999, // Несуществующий пользователь
		},
		Text: "Test message",
	}

	messageID, err := s.repository.SaveMessage(s.ctx, message)
	s.Error(err) // Должна быть ошибка foreign key constraint
	s.Equal(uint64(0), messageID)
}

func (s *MessagesRepositoryTestSuite) TestSaveMessage_MultipleMessages() {
	// Создаем тестовые данные
	s.createTestUsers()
	s.createTestChats()

	// Тест: Сохранение нескольких сообщений
	messages := []string{
		"First message",
		"Second message",
		"Third message",
	}

	for i, text := range messages {
		message := domains.Message{
			ChatID: 1,
			Sender: domains.User{
				ID: 1,
			},
			Text: text,
		}

		messageID, err := s.repository.SaveMessage(s.ctx, message)
		s.NoError(err)
		s.Greater(messageID, uint64(0))

		// Проверяем каждое сообщение
		var dbText string

		err = s.tx.QueryRowContext(
			s.ctx,
			`SELECT text FROM messages WHERE id = $1`,
			messageID,
		).Scan(&dbText)
		s.NoError(err)
		s.Equal(text, dbText, "Message %d should be saved correctly", i+1)
	}

	// Проверяем общее количество сообщений в чате 1
	var count int

	err := s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM messages WHERE chat_id = 1`,
	).Scan(&count)
	s.NoError(err)
	s.GreaterOrEqual(count, len(messages))
}

func (s *MessagesRepositoryTestSuite) TestGetChatMessages_WithoutPagination() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Получение всех сообщений чата без пагинации
	chatID := uint64(1)

	messages, err := s.repository.GetChatMessages(s.ctx, chatID, nil)
	s.NoError(err)
	s.NotNil(messages)

	// В чате 1 должно быть как минимум 5 сообщений + 3 для пагинации = 8
	s.GreaterOrEqual(len(messages), 8)

	// Проверяем сортировку по ID в порядке убывания
	for i := range len(messages) - 1 {
		s.Greater(messages[i].ID, messages[i+1].ID, "Messages should be sorted by ID DESC")
	}

	// Проверяем структуру первого сообщения
	firstMessage := messages[0]
	s.NotZero(firstMessage.ID)
	s.Equal(chatID, firstMessage.ChatID)
	s.NotEmpty(firstMessage.Text)
	s.NotZero(firstMessage.CreatedAt)
	s.NotZero(firstMessage.UpdatedAt)

	// Проверяем отправителя
	s.NotZero(firstMessage.Sender.ID)
	s.NotEmpty(firstMessage.Sender.Username)
	s.NotEmpty(firstMessage.Sender.Email)
	s.NotEmpty(firstMessage.Sender.Password)
	s.NotZero(firstMessage.Sender.CreatedAt)
	s.NotZero(firstMessage.Sender.UpdatedAt)

	// Проверяем, что все сообщения принадлежат указанному чату
	for _, msg := range messages {
		s.Equal(chatID, msg.ChatID, "All messages should belong to chat %d", chatID)
	}
}

func (s *MessagesRepositoryTestSuite) TestGetChatMessages_WithLimit() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Получение сообщений с лимитом
	chatID := uint64(1)
	pagination := &domains.Pagination{
		Limit: pointers.New[uint64](3),
	}

	messages, err := s.repository.GetChatMessages(s.ctx, chatID, pagination)
	s.NoError(err)
	s.NotNil(messages)
	s.Equal(3, len(messages))

	// Проверяем сортировку
	s.Equal(uint64(15), messages[0].ID) // Последнее сообщение
	s.Equal(uint64(14), messages[1].ID)
	s.Equal(uint64(13), messages[2].ID)

	// Проверяем, что все сообщения принадлежат указанному чату
	for _, msg := range messages {
		s.Equal(chatID, msg.ChatID)
	}
}

func (s *MessagesRepositoryTestSuite) TestGetChatMessages_WithOffset() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Получение сообщений с оффсетом
	chatID := uint64(1)
	pagination := &domains.Pagination{
		Limit:  pointers.New[uint64](2),
		Offset: pointers.New[uint64](2),
	}

	messages, err := s.repository.GetChatMessages(s.ctx, chatID, pagination)
	s.NoError(err)
	s.NotNil(messages)
	s.Equal(2, len(messages))

	// Должны пропустить первые 2 сообщения
	s.Equal(uint64(13), messages[0].ID)
	s.Equal(uint64(5), messages[1].ID) // После сообщений 15, 14, 13

	// Проверяем, что все сообщения принадлежат указанному чату
	for _, msg := range messages {
		s.Equal(chatID, msg.ChatID)
	}
}

func (s *MessagesRepositoryTestSuite) TestGetChatMessages_EmptyChat() {
	// Тест: Получение сообщений из пустого чата
	// Создаем новый пустой чат
	_, err := s.tx.ExecContext(
		s.ctx,
		`INSERT INTO chats (id, title, type) VALUES (100, 'Empty Chat', 'group')`,
	)
	s.NoError(err)

	messages, err := s.repository.GetChatMessages(s.ctx, 100, nil)
	s.NoError(err)
	s.Nil(messages) // Должен вернуться nil
}

func (s *MessagesRepositoryTestSuite) TestGetChatMessages_NonExistentChat() {
	// Тест: Получение сообщений из несуществующего чата
	messages, err := s.repository.GetChatMessages(s.ctx, 999, nil)
	s.NoError(err)
	s.Nil(messages) // Должен вернуться nil
}

func (s *MessagesRepositoryTestSuite) TestGetChatMessages_DifferentChats() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Проверяем, что возвращаются только сообщения указанного чата
	chat1Messages, err := s.repository.GetChatMessages(s.ctx, 1, nil)
	s.NoError(err)
	s.NotNil(chat1Messages)

	chat2Messages, err := s.repository.GetChatMessages(s.ctx, 2, nil)
	s.NoError(err)
	s.NotNil(chat2Messages)

	// Сообщения из разных чатов должны быть разными
	chat1IDs := make(map[uint64]bool)
	for _, msg := range chat1Messages {
		chat1IDs[msg.ID] = true

		s.Equal(uint64(1), msg.ChatID, "Message %d should belong to chat 1", msg.ID)
	}

	for _, msg := range chat2Messages {
		s.Equal(uint64(2), msg.ChatID, "Message %d should belong to chat 2", msg.ID)
		s.False(chat1IDs[msg.ID], "Message %d should not appear in chat 1", msg.ID)
	}
}

func (s *MessagesRepositoryTestSuite) TestGetChatMessages_ZeroLimit() {
	// Тест: Получение сообщений с лимитом 0
	pagination := &domains.Pagination{
		Limit: pointers.New[uint64](0),
	}

	messages, err := s.repository.GetChatMessages(s.ctx, 1, pagination)
	s.NoError(err)
	s.Nil(messages) // Должен вернуться nil при лимите 0
}

func (s *MessagesRepositoryTestSuite) TestGetMessageByID_Success() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Получение сообщения по ID
	messageID := uint64(1)

	message, err := s.repository.GetMessageByID(s.ctx, messageID)
	s.NoError(err)
	s.NotNil(message)

	s.Equal(messageID, message.ID)
	s.Equal(uint64(1), message.ChatID)
	s.Equal(uint64(1), message.Sender.ID)
	s.Equal("john_doe", message.Sender.Username)
	s.Equal("john@example.com", message.Sender.Email)
	s.True(message.Sender.EmailConfirmed)
	s.Equal("$2a$10$hashedpassword1", message.Sender.Password)
	s.Equal("Hello everyone!", message.Text)
	s.NotZero(message.CreatedAt)
	s.NotZero(message.UpdatedAt)
	s.NotZero(message.Sender.CreatedAt)
	s.NotZero(message.Sender.UpdatedAt)
}

func (s *MessagesRepositoryTestSuite) TestGetMessageByID_DifferentSender() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Получение сообщения от другого отправителя
	messageID := uint64(2) // Сообщение от Jane

	message, err := s.repository.GetMessageByID(s.ctx, messageID)
	s.NoError(err)
	s.NotNil(message)

	s.Equal(messageID, message.ID)
	s.Equal(uint64(1), message.ChatID)
	s.Equal(uint64(2), message.Sender.ID)
	s.Equal("jane_smith", message.Sender.Username)
	s.Equal("jane@example.com", message.Sender.Email)
	s.False(message.Sender.EmailConfirmed) // У Jane неподтвержденный email
	s.Equal("$2a$10$hashedpassword2", message.Sender.Password)
	s.Equal("Hi John!", message.Text)
}

func (s *MessagesRepositoryTestSuite) TestGetMessageByID_NotFound() {
	// Тест: Попытка получить несуществующее сообщение
	message, err := s.repository.GetMessageByID(s.ctx, 999)
	s.Error(err)
	s.Nil(message)
	s.Contains(err.Error(), "no rows")
}

func (s *MessagesRepositoryTestSuite) TestGetMessageByID_ZeroID() {
	// Тест: Попытка получить сообщение с ID = 0
	message, err := s.repository.GetMessageByID(s.ctx, 0)
	s.Error(err)
	s.Nil(message)
}

func (s *MessagesRepositoryTestSuite) TestGetMessageByID_JoinCorrectness() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Проверка правильности JOIN
	messageID := uint64(6) // Сообщение в приватном чате

	message, err := s.repository.GetMessageByID(s.ctx, messageID)
	s.NoError(err)
	s.NotNil(message)

	// Проверяем, что данные отправителя корректны
	s.Equal(uint64(1), message.Sender.ID)
	s.Equal("john_doe", message.Sender.Username)
	s.Equal("Hi Alice!", message.Text)

	// Проверяем, что это сообщение из приватного чата
	s.Equal(uint64(2), message.ChatID)
}

func (s *MessagesRepositoryTestSuite) TestGetChatMessages_WithLargeOffset() {
	// Тест: Пагинация с большим оффсетом
	chatID := uint64(1)
	pagination := &domains.Pagination{
		Limit:  pointers.New[uint64](5),
		Offset: pointers.New[uint64](100), // Очень большой оффсет
	}

	messages, err := s.repository.GetChatMessages(s.ctx, chatID, pagination)
	s.NoError(err)
	s.Nil(messages) // Должен вернуться nil при слишком большом оффсете
}

func (s *MessagesRepositoryTestSuite) TestMessageOrdering() {
	s.createTestUsers()
	s.createTestChats()
	// Тест: Проверка порядка сообщений при разных сценариях

	// Сохраняем несколько сообщений с разными временами
	messagesToSave := []struct {
		text  string
		delay time.Duration
	}{
		{"Message 1", 0},
		{"Message 2", time.Millisecond * 10},
		{"Message 3", time.Millisecond * 20},
	}

	var messageIDs []uint64

	for _, msg := range messagesToSave {
		// Имитируем небольшую задержку
		time.Sleep(msg.delay)

		message := domains.Message{
			ChatID: 3, // Используем чат 3
			Sender: domains.User{
				ID: 1,
			},
			Text: msg.text,
		}

		messageID, err := s.repository.SaveMessage(s.ctx, message)
		s.NoError(err)

		messageIDs = append(messageIDs, messageID)
	}

	// Получаем сообщения
	chatMessages, err := s.repository.GetChatMessages(s.ctx, 3, nil)
	s.NoError(err)
	s.NotNil(chatMessages)

	// Проверяем, что сообщения отсортированы по ID в порядке убывания
	// (последние созданные сообщения должны быть первыми)
	for i := range len(chatMessages) - 1 {
		s.Greater(chatMessages[i].ID, chatMessages[i+1].ID,
			"Messages should be sorted by ID DESC, position %d", i)
	}

	// Проверяем, что наши новые сообщения в правильном порядке
	foundCount := 0

	for _, msg := range chatMessages {
		for _, savedID := range messageIDs {
			if msg.ID == savedID {
				foundCount++

				break
			}
		}
	}

	s.Equal(len(messageIDs), foundCount, "All saved messages should be returned")
}

func (s *MessagesRepositoryTestSuite) createTestUsers() {
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

func (s *MessagesRepositoryTestSuite) createTestChats() {
	// Создаем чаты
	chats := []struct {
		id          uint64
		title       string
		description string
		chatType    string
	}{
		{1, "General Chat", "General discussion", "group"},
		{2, "Private Chat", "Private conversation", "private"},
		{3, "Project Chat", "Project discussions", "group"},
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

func (s *MessagesRepositoryTestSuite) createTestChatMembers() {
	// Создаем участников чатов
	chatMembers := []struct {
		chatID uint64
		userID uint64
		isRead bool
	}{
		{1, 1, true},
		{1, 2, true},
		{1, 3, false},
		{2, 1, true},
		{2, 4, true},
		{3, 1, true},
		{3, 2, true},
		{3, 3, true},
		{3, 4, false},
	}

	for _, cm := range chatMembers {
		_, err := s.tx.ExecContext(
			s.ctx,
			`INSERT INTO chats_members (chat_id, user_id, is_read, created_at, updated_at) 
			 VALUES ($1, $2, $3, $4, $4)`,
			cm.chatID,
			cm.userID,
			cm.isRead,
			time.Now().UTC(),
		)
		s.NoError(err)
	}
}

func (s *MessagesRepositoryTestSuite) createTestMessages() {
	// Создаем сообщения
	messages := []struct {
		id       uint64
		chatID   uint64
		senderID uint64
		text     string
	}{
		{1, 1, 1, "Hello everyone!"},
		{2, 1, 2, "Hi John!"},
		{3, 1, 3, "Good morning!"},
		{4, 1, 1, "How are you all doing?"},
		{5, 1, 2, "I'm doing great!"},
		{6, 2, 1, "Hi Alice!"},
		{7, 2, 4, "Hello John!"},
		{8, 2, 1, "How's your project going?"},
		{9, 3, 1, "Let's discuss the project"},
		{10, 3, 2, "I've completed my tasks"},
		{11, 3, 3, "I need more time"},
		{12, 3, 4, "I'll send my update tomorrow"},
		{13, 1, 1, "Test message for pagination 1"},
		{14, 1, 2, "Test message for pagination 2"},
		{15, 1, 3, "Test message for pagination 3"},
	}

	for _, m := range messages {
		_, err := s.tx.ExecContext(
			s.ctx,
			`INSERT INTO messages (id, chat_id, sender_id, text, created_at, updated_at) 
			 VALUES ($1, $2, $3, $4, $5, $5)`,
			m.id,
			m.chatID,
			m.senderID,
			m.text,
			time.Now().UTC(),
		)
		s.NoError(err)
	}
}
