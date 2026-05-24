//go:build integration

package messages_test

import (
	"context"
	"database/sql"
	"os"
	"path"
	"testing"
	"time"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/domains"
	"github.com/DKhorkov/kfc/internal/repositories/messages"
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
	repository  *messages.Repository
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
	s.repository = messages.New(tx, s.logger)
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

func (s *RepositoryTestSuite) TestSaveMessage_Success() {
	// Создаем тестовые данные
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

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

func (s *RepositoryTestSuite) TestSaveMessage_EmptyText() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

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

func (s *RepositoryTestSuite) TestSaveMessage_LongText() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

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

func (s *RepositoryTestSuite) TestSaveMessage_InvalidChat() {
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

func (s *RepositoryTestSuite) TestSaveMessage_InvalidSender() {
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

func (s *RepositoryTestSuite) TestSaveMessage_MultipleMessages() {
	// Создаем тестовые данные
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

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

func (s *RepositoryTestSuite) TestGetChatMessages_WithoutPagination() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Получение всех сообщений чата без пагинации
	userID := uint64(1)
	chatID := uint64(1)

	messages, err := s.repository.GetChatMessages(s.ctx, userID, chatID, nil)
	s.NoError(err)
	s.NotNil(messages)

	s.GreaterOrEqual(len(messages), 3)

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
	s.False(firstMessage.IsRead)

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

func (s *RepositoryTestSuite) TestGetChatMessages_WithLimit() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Получение сообщений с лимитом
	userID := uint64(1)
	chatID := uint64(1)
	pagination := &domains.Pagination{
		Limit: pointers.New[uint64](3),
	}

	messages, err := s.repository.GetChatMessages(s.ctx, userID, chatID, pagination)
	s.NoError(err)
	s.NotNil(messages)
	s.Equal(3, len(messages))

	// Проверяем сортировку
	s.Equal(uint64(13), messages[0].ID) // Последнее сообщение
	s.Equal(uint64(4), messages[1].ID)
	s.Equal(uint64(1), messages[2].ID)

	// Проверяем, что все сообщения принадлежат указанному чату
	for _, msg := range messages {
		s.Equal(chatID, msg.ChatID)
		s.False(msg.IsRead)
	}
}

func (s *RepositoryTestSuite) TestGetChatMessages_WithOffset() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Получение сообщений с оффсетом
	userID := uint64(1)
	chatID := uint64(1)
	pagination := &domains.Pagination{
		Limit:  pointers.New[uint64](2),
		Offset: pointers.New[uint64](2),
	}

	messages, err := s.repository.GetChatMessages(s.ctx, userID, chatID, pagination)
	s.NoError(err)
	s.NotNil(messages)
	s.Equal(1, len(messages))

	// Должны пропустить первые 2 сообщения
	s.Equal(uint64(1), messages[0].ID)

	// Проверяем, что все сообщения принадлежат указанному чату
	for _, msg := range messages {
		s.Equal(chatID, msg.ChatID)
		s.False(msg.IsRead)
	}
}

func (s *RepositoryTestSuite) TestGetChatMessages_EmptyChat() {
	// Тест: Получение сообщений из пустого чата
	// Создаем новый пустой чат
	_, err := s.tx.ExecContext(
		s.ctx,
		`INSERT INTO chats (id, title, type) VALUES (100, 'Empty Chat', 'group')`,
	)
	s.NoError(err)

	userID := uint64(1)
	chatID := uint64(100)

	messages, err := s.repository.GetChatMessages(s.ctx, userID, chatID, nil)
	s.NoError(err)
	s.Nil(messages) // Должен вернуться nil
}

func (s *RepositoryTestSuite) TestGetChatMessages_NonExistentChat() {
	userID := uint64(1)
	chatID := uint64(999)

	// Тест: Получение сообщений из несуществующего чата
	messages, err := s.repository.GetChatMessages(s.ctx, userID, chatID, nil)
	s.NoError(err)
	s.Nil(messages) // Должен вернуться nil
}

func (s *RepositoryTestSuite) TestGetChatMessages_DifferentChats() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	userID := uint64(1)

	// Тест: Проверяем, что возвращаются только сообщения указанного чата
	chat1Messages, err := s.repository.GetChatMessages(s.ctx, userID, 1, nil)
	s.NoError(err)
	s.NotNil(chat1Messages)

	chat2Messages, err := s.repository.GetChatMessages(s.ctx, userID, 2, nil)
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

func (s *RepositoryTestSuite) TestGetChatMessages_ZeroLimit() {
	// Тест: Получение сообщений с лимитом 0
	userID := uint64(1)
	chatID := uint64(1)
	pagination := &domains.Pagination{
		Limit: pointers.New[uint64](0),
	}

	messages, err := s.repository.GetChatMessages(s.ctx, userID, chatID, pagination)
	s.NoError(err)
	s.Nil(messages) // Должен вернуться nil при лимите 0
}

func (s *RepositoryTestSuite) TestGetMessageByID_Success() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Получение сообщения по ID
	userID := uint64(1)
	messageID := uint64(1)

	message, err := s.repository.GetMessageByID(s.ctx, userID, messageID)
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
	s.False(message.IsRead)
}

func (s *RepositoryTestSuite) TestGetMessageByID_DifferentSender() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Получение сообщения от другого отправителя
	userID := uint64(1)
	messageID := uint64(2) // Сообщение от Jane

	message, err := s.repository.GetMessageByID(s.ctx, userID, messageID)
	s.Error(err)
	s.Nil(message)
}

func (s *RepositoryTestSuite) TestGetMessageByID_NotFound() {
	userID := uint64(1)

	// Тест: Попытка получить несуществующее сообщение
	message, err := s.repository.GetMessageByID(s.ctx, userID, 999)
	s.Error(err)
	s.Nil(message)
	s.Contains(err.Error(), "no rows")
}

func (s *RepositoryTestSuite) TestGetMessageByID_ZeroID() {
	userID := uint64(1)

	// Тест: Попытка получить сообщение с ID = 0
	message, err := s.repository.GetMessageByID(s.ctx, userID, 0)
	s.Error(err)
	s.Nil(message)
}

func (s *RepositoryTestSuite) TestGetMessageByID_JoinCorrectness() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Проверка правильности JOIN
	userID := uint64(1)
	messageID := uint64(6) // Сообщение в приватном чате

	message, err := s.repository.GetMessageByID(s.ctx, userID, messageID)
	s.NoError(err)
	s.NotNil(message)

	// Проверяем, что данные отправителя корректны
	s.Equal(uint64(1), message.Sender.ID)
	s.Equal("john_doe", message.Sender.Username)
	s.Equal("Hi Alice!", message.Text)
	s.False(message.IsRead)

	// Проверяем, что это сообщение из приватного чата
	s.Equal(uint64(2), message.ChatID)
}

func (s *RepositoryTestSuite) TestGetChatMessages_WithLargeOffset() {
	// Тест: Пагинация с большим оффсетом
	userID := uint64(1)
	chatID := uint64(1)
	pagination := &domains.Pagination{
		Limit:  pointers.New[uint64](5),
		Offset: pointers.New[uint64](100), // Очень большой оффсет
	}

	messages, err := s.repository.GetChatMessages(s.ctx, userID, chatID, pagination)
	s.NoError(err)
	s.Nil(messages) // Должен вернуться nil при слишком большом оффсете
}

func (s *RepositoryTestSuite) TestMessageOrdering() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
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

	userID := uint64(1)

	// Получаем сообщения
	chatMessages, err := s.repository.GetChatMessages(s.ctx, userID, 3, nil)
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

func (s *RepositoryTestSuite) TestChangeMessagesIsReadStatus_Success() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	// Тест: Успешное изменение статуса прочтения
	userID := uint64(1)
	messages := []domains.Message{{ID: 1}}
	newIsRead := true

	// Проверяем начальное состояние (должно быть false)
	var initialIsRead bool

	err := s.tx.QueryRowContext(
		s.ctx,
		`SELECT is_read FROM messages_statuses WHERE user_id = $1 AND message_id = $2`,
		userID, messages[0].ID,
	).Scan(&initialIsRead)
	s.NoError(err)
	s.False(initialIsRead, "Initial is_read should be false")

	// Изменяем статус
	err = s.repository.ChangeMessagesIsReadStatus(s.ctx, userID, messages, newIsRead)
	s.NoError(err)

	// Проверяем конечное состояние
	var finalIsRead bool

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT is_read FROM messages_statuses WHERE user_id = $1 AND message_id = $2`,
		userID, messages[0].ID,
	).Scan(&finalIsRead)
	s.NoError(err)
	s.True(finalIsRead, "Final is_read should be true")
}

func (s *RepositoryTestSuite) TestChangeMessagesIsReadStatus_NonExistentRecord() {
	// Тест: Попытка изменить статус для несуществующей записи
	err := s.repository.ChangeMessagesIsReadStatus(s.ctx, 999, []domains.Message{{ID: 222}}, true)
	s.NoError(err) // UPDATE без найденных строк не возвращает ошибку
}

func (s *RepositoryTestSuite) TestReadAllChatMessages_Success() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	userID := uint64(1)
	chatID := uint64(1)

	// Добавляем непрочитанные статусы для user 1 на сообщения от других пользователей в чате 1
	for _, messageID := range []uint64{2, 3, 5, 14, 15} {
		_, err := s.tx.ExecContext(
			s.ctx,
			`INSERT INTO messages_statuses (message_id, user_id, is_read) VALUES ($1, $2, $3)`,
			messageID, userID, false,
		)
		s.NoError(err)
	}

	// Проверяем что есть непрочитанные
	var unreadCount int

	err := s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM messages_statuses ms
		 JOIN messages m ON ms.message_id = m.id
		 WHERE ms.user_id = $1 AND m.chat_id = $2 AND ms.is_read = false`,
		userID, chatID,
	).Scan(&unreadCount)
	s.NoError(err)
	s.Greater(unreadCount, 0)

	// Помечаем все прочитанными
	err = s.repository.ReadAllChatMessages(s.ctx, userID, chatID)
	s.NoError(err)

	// Проверяем что непрочитанных больше нет
	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM messages_statuses ms
		 JOIN messages m ON ms.message_id = m.id
		 WHERE ms.user_id = $1 AND m.chat_id = $2 AND ms.is_read = false`,
		userID, chatID,
	).Scan(&unreadCount)
	s.NoError(err)
	s.Equal(0, unreadCount)
}

func (s *RepositoryTestSuite) TestReadAllChatMessages_DoesNotAffectOtherChats() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	userID := uint64(1)

	// Добавляем непрочитанные статусы для user 1 в чате 1 и чате 2
	for _, messageID := range []uint64{2, 3} { // чат 1
		_, err := s.tx.ExecContext(
			s.ctx,
			`INSERT INTO messages_statuses (message_id, user_id, is_read) VALUES ($1, $2, $3)`,
			messageID, userID, false,
		)
		s.NoError(err)
	}

	_, err := s.tx.ExecContext(
		s.ctx,
		`INSERT INTO messages_statuses (message_id, user_id, is_read) VALUES ($1, $2, $3)`,
		7, userID, false, // сообщение в чате 2
	)
	s.NoError(err)

	// Помечаем прочитанными только чат 1
	err = s.repository.ReadAllChatMessages(s.ctx, userID, 1)
	s.NoError(err)

	// Проверяем что в чате 2 осталось непрочитанное
	var unreadCount int

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM messages_statuses ms
		 JOIN messages m ON ms.message_id = m.id
		 WHERE ms.user_id = $1 AND m.chat_id = $2 AND ms.is_read = false`,
		userID, uint64(2),
	).Scan(&unreadCount)
	s.NoError(err)
	s.Equal(3, unreadCount)
}

func (s *RepositoryTestSuite) TestReadAllChatMessages_NonExistentChat() {
	err := s.repository.ReadAllChatMessages(s.ctx, 999, 999)
	s.NoError(err) // UPDATE без найденных строк не возвращает ошибку
}

func (s *RepositoryTestSuite) TestDeleteMessageForUser_Success() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	userID := uint64(1)
	messageID := uint64(1)

	// Проверяем начальное состояние: is_deleted = false
	var initialIsDeleted bool

	err := s.tx.QueryRowContext(
		s.ctx,
		`SELECT is_deleted FROM messages_statuses WHERE user_id = $1 AND message_id = $2`,
		userID, messageID,
	).Scan(&initialIsDeleted)
	s.NoError(err)
	s.False(initialIsDeleted, "Initial is_deleted should be false")

	// Удаляем сообщение для пользователя
	err = s.repository.DeleteMessageForUser(s.ctx, userID, messageID)
	s.NoError(err)

	// Проверяем конечное состояние: is_deleted = true
	var finalIsDeleted bool

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT is_deleted FROM messages_statuses WHERE user_id = $1 AND message_id = $2`,
		userID, messageID,
	).Scan(&finalIsDeleted)
	s.NoError(err)
	s.True(finalIsDeleted, "Final is_deleted should be true")
}

func (s *RepositoryTestSuite) TestDeleteMessageForUser_DoesNotAffectOtherUsers() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	userID := uint64(1)
	otherUserID := uint64(2)
	messageID := uint64(2) // Сообщение от user 2, статус есть у user 2

	// Добавляем статус для user 1 на это сообщение
	_, err := s.tx.ExecContext(
		s.ctx,
		`INSERT INTO messages_statuses (message_id, user_id, is_read, is_deleted) VALUES ($1, $2, false, false)`,
		messageID, userID,
	)
	s.NoError(err)

	// Удаляем сообщение только для user 1
	err = s.repository.DeleteMessageForUser(s.ctx, userID, messageID)
	s.NoError(err)

	// Проверяем, что для user 1 is_deleted = true
	var user1Deleted bool

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT is_deleted FROM messages_statuses WHERE user_id = $1 AND message_id = $2`,
		userID, messageID,
	).Scan(&user1Deleted)
	s.NoError(err)
	s.True(user1Deleted)

	// Проверяем, что для другого пользователя is_deleted = false
	var otherUserDeleted bool

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT is_deleted FROM messages_statuses WHERE user_id = $1 AND message_id = $2`,
		otherUserID, messageID,
	).Scan(&otherUserDeleted)
	s.NoError(err)
	s.False(otherUserDeleted, "Other user's is_deleted should remain false")
}

func (s *RepositoryTestSuite) TestDeleteMessageForUser_NonExistentRecord() {
	err := s.repository.DeleteMessageForUser(s.ctx, 999, 999)
	s.NoError(err) // UPDATE без найденных строк не возвращает ошибку
}

func (s *RepositoryTestSuite) TestDeleteMessageForAll_Success() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	messageID := uint64(1)

	// Добавляем статусы для нескольких пользователей
	for _, userID := range []uint64{2, 3} {
		_, err := s.tx.ExecContext(
			s.ctx,
			`INSERT INTO messages_statuses (message_id, user_id, is_read, is_deleted) VALUES ($1, $2, false, false)`,
			messageID, userID,
		)
		s.NoError(err)
	}

	// Удаляем сообщение для всех
	err := s.repository.DeleteMessageForAll(s.ctx, messageID)
	s.NoError(err)

	// Проверяем, что is_deleted = true для всех пользователей
	var count int

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM messages_statuses WHERE message_id = $1 AND is_deleted = false`,
		messageID,
	).Scan(&count)
	s.NoError(err)
	s.Equal(0, count, "All statuses should have is_deleted = true")

	var totalCount int

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT COUNT(*) FROM messages_statuses WHERE message_id = $1 AND is_deleted = true`,
		messageID,
	).Scan(&totalCount)
	s.NoError(err)
	s.GreaterOrEqual(totalCount, 3, "At least 3 statuses should be marked as deleted")
}

func (s *RepositoryTestSuite) TestDeleteMessageForAll_NonExistentRecord() {
	err := s.repository.DeleteMessageForAll(s.ctx, 999)
	s.NoError(err) // UPDATE без найденных строк не возвращает ошибку
}

func (s *RepositoryTestSuite) TestSaveMessage_WithReplyToMessage() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	// Сначала создаём оригинальное сообщение
	originalMessage := domains.Message{
		ChatID: 1,
		Sender: domains.User{ID: 1},
		Text:   "Original message",
	}

	originalID, err := s.repository.SaveMessage(s.ctx, originalMessage)
	s.NoError(err)
	s.Greater(originalID, uint64(0))

	// Создаём ответ на сообщение
	replyMessage := domains.Message{
		ChatID: 1,
		Sender: domains.User{ID: 2},
		Text:   "Reply to original",
		ReplyToMessage: &domains.Message{
			ID: originalID,
		},
	}

	replyID, err := s.repository.SaveMessage(s.ctx, replyMessage)
	s.NoError(err)
	s.Greater(replyID, uint64(0))

	// Проверяем, что reply_to_message_id сохранён
	var replyToMessageID *uint64

	err = s.tx.QueryRowContext(
		s.ctx,
		`SELECT reply_to_message_id FROM messages WHERE id = $1`,
		replyID,
	).Scan(&replyToMessageID)
	s.NoError(err)
	s.NotNil(replyToMessageID)
	s.Equal(originalID, *replyToMessageID)
}

func (s *RepositoryTestSuite) TestGetChatMessages_WithReply() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	// Создаём оригинальное сообщение
	originalMessage := domains.Message{
		ChatID: 1,
		Sender: domains.User{ID: 1},
		Text:   "Original message for reply test",
	}

	originalID, err := s.repository.SaveMessage(s.ctx, originalMessage)
	s.NoError(err)

	// Создаём ответ
	replyMessage := domains.Message{
		ChatID: 1,
		Sender: domains.User{ID: 2},
		Text:   "This is a reply",
		ReplyToMessage: &domains.Message{
			ID: originalID,
		},
	}

	replyID, err := s.repository.SaveMessage(s.ctx, replyMessage)
	s.NoError(err)

	// Получаем сообщения чата от имени user 2
	userID := uint64(2)
	chatMessages, err := s.repository.GetChatMessages(s.ctx, userID, 1, nil)
	s.NoError(err)
	s.NotNil(chatMessages)

	// Находим ответное сообщение
	var foundReply *domains.Message

	for i, msg := range chatMessages {
		if msg.ID == replyID {
			foundReply = &chatMessages[i]

			break
		}
	}

	s.NotNil(foundReply, "Reply message should be found in chat messages")
	s.NotNil(foundReply.ReplyToMessage, "Reply should have ReplyToMessage set")
	s.Equal(originalID, foundReply.ReplyToMessage.ID)
	s.Equal("Original message for reply test", foundReply.ReplyToMessage.Text)
	s.Equal(uint64(1), foundReply.ReplyToMessage.Sender.ID)
	s.Equal("john_doe", foundReply.ReplyToMessage.Sender.Username)
}

func (s *RepositoryTestSuite) TestGetMessageByID_WithReply() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()

	// Создаём оригинальное сообщение
	originalMessage := domains.Message{
		ChatID: 1,
		Sender: domains.User{ID: 1},
		Text:   "Original for GetMessageByID test",
	}

	originalID, err := s.repository.SaveMessage(s.ctx, originalMessage)
	s.NoError(err)

	// Создаём ответ
	replyMessage := domains.Message{
		ChatID: 1,
		Sender: domains.User{ID: 2},
		Text:   "Reply for GetMessageByID test",
		ReplyToMessage: &domains.Message{
			ID: originalID,
		},
	}

	replyID, err := s.repository.SaveMessage(s.ctx, replyMessage)
	s.NoError(err)

	// Получаем ответное сообщение по ID
	userID := uint64(2)
	message, err := s.repository.GetMessageByID(s.ctx, userID, replyID)
	s.NoError(err)
	s.NotNil(message)

	s.Equal(replyID, message.ID)
	s.NotNil(message.ReplyToMessage, "Message should have ReplyToMessage set")
	s.Equal(originalID, message.ReplyToMessage.ID)
	s.Equal("Original for GetMessageByID test", message.ReplyToMessage.Text)
	s.Equal(uint64(1), message.ReplyToMessage.Sender.ID)
	s.Equal("john_doe", message.ReplyToMessage.Sender.Username)
}

func (s *RepositoryTestSuite) TestGetChatMessages_FiltersDeletedMessages() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	userID := uint64(1)
	chatID := uint64(1)

	// Получаем сообщения до удаления
	messagesBefore, err := s.repository.GetChatMessages(s.ctx, userID, chatID, nil)
	s.NoError(err)
	countBefore := len(messagesBefore)
	s.Greater(countBefore, 0)

	// Помечаем одно сообщение как удалённое для user 1
	messageIDToDelete := messagesBefore[0].ID

	err = s.repository.DeleteMessageForUser(s.ctx, userID, messageIDToDelete)
	s.NoError(err)

	// Получаем сообщения после удаления
	messagesAfter, err := s.repository.GetChatMessages(s.ctx, userID, chatID, nil)
	s.NoError(err)
	s.Equal(countBefore-1, len(messagesAfter), "Deleted message should be filtered out")

	// Проверяем, что удалённого сообщения нет в результатах
	for _, msg := range messagesAfter {
		s.NotEqual(messageIDToDelete, msg.ID, "Deleted message should not appear in results")
	}
}

func (s *RepositoryTestSuite) TestGetMessageByID_DeletedMessage() {
	s.createTestUsers()
	s.createTestChats()
	s.createTestChatMembers()
	s.createTestMessages()

	userID := uint64(1)
	messageID := uint64(1)

	// Проверяем, что сообщение доступно
	message, err := s.repository.GetMessageByID(s.ctx, userID, messageID)
	s.NoError(err)
	s.NotNil(message)

	// Удаляем сообщение для user 1
	err = s.repository.DeleteMessageForUser(s.ctx, userID, messageID)
	s.NoError(err)

	// Проверяем, что удалённое сообщение больше не доступно
	message, err = s.repository.GetMessageByID(s.ctx, userID, messageID)
	s.Error(err)
	s.Nil(message)
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

func (s *RepositoryTestSuite) createTestChatMembers() {
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
			`INSERT INTO chats_members (chat_id, user_id, created_at, updated_at) 
			 VALUES ($1, $2, $3, $3)`,
			cm.chatID,
			cm.userID,
			time.Now().UTC(),
		)
		s.NoError(err)
	}
}

func (s *RepositoryTestSuite) createTestMessages() {
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

		_, err = s.tx.ExecContext(
			s.ctx,
			`INSERT INTO messages_statuses (message_id, user_id, is_read) 
			 VALUES ($1, $2, $3)`,
			m.id,
			m.senderID,
			false,
		)
		s.NoError(err)
	}
}
