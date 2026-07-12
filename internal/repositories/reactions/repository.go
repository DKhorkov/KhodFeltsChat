package reactions

import (
	"context"
	"fmt"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/DKhorkov/libs/logging"
	sq "github.com/Masterminds/squirrel"
)

const (
	reactionsTableName        = "reactions"
	messageReactionsTableName = "messages_reactions"

	idColumnName         = "id"
	emojiColumnName      = "emoji"
	sortOrderColumnName  = "sort_order"
	messageIDColumnName  = "message_id"
	userIDColumnName     = "user_id"
	reactionIDColumnName = "reaction_id"

	asc = "ASC"
)

type Repository struct {
	tx     pg.Transaction
	logger logging.Logger
}

func New(tx pg.Transaction, logger logging.Logger) *Repository {
	return &Repository{tx: tx, logger: logger}
}

func (r *Repository) ListReactions(ctx context.Context) ([]domains.Reaction, error) {
	stmt, params, err := sq.
		Select(idColumnName, emojiColumnName, sortOrderColumnName).
		From(reactionsTableName).
		OrderBy(fmt.Sprintf("%s %s", sortOrderColumnName, asc)).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.tx.QueryContext(ctx, stmt, params...)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err = rows.Close(); err != nil {
			logging.LogErrorContext(ctx, r.logger, "Failed to close SQL rows", err)
		}
	}()

	var reactions []domains.Reaction

	for rows.Next() {
		reaction := domains.Reaction{}
		columns := pg.GetEntityColumns(&reaction)

		if err = rows.Scan(columns...); err != nil {
			return nil, err
		}

		reactions = append(reactions, reaction)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return reactions, nil
}

func (r *Repository) GetReactionByID(
	ctx context.Context,
	id uint64,
) (*domains.Reaction, error) {
	stmt, params, err := sq.
		Select(idColumnName, emojiColumnName, sortOrderColumnName).
		From(reactionsTableName).
		Where(sq.Eq{idColumnName: id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	reaction := &domains.Reaction{}

	columns := pg.GetEntityColumns(reaction)
	if err = r.tx.QueryRowContext(ctx, stmt, params...).Scan(columns...); err != nil {
		return nil, err
	}

	return reaction, nil
}

func (r *Repository) AddMessageReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) error {
	stmt, params, err := sq.
		Insert(messageReactionsTableName).
		Columns(messageIDColumnName, userIDColumnName, reactionIDColumnName).
		Values(dto.MessageID, dto.UserID, dto.ReactionID).
		Suffix(fmt.Sprintf(
			"ON CONFLICT (%s, %s, %s) DO NOTHING",
			messageIDColumnName, userIDColumnName, reactionIDColumnName,
		)).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	res, err := r.tx.ExecContext(ctx, stmt, params...)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return customerrors.ErrReactionAlreadyExists
	}

	return nil
}

func (r *Repository) RemoveMessageReaction(
	ctx context.Context,
	dto domains.MessageReactionDTO,
) error {
	stmt, params, err := sq.
		Delete(messageReactionsTableName).
		Where(sq.Eq{
			messageIDColumnName:  dto.MessageID,
			userIDColumnName:     dto.UserID,
			reactionIDColumnName: dto.ReactionID,
		}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	res, err := r.tx.ExecContext(ctx, stmt, params...)
	if err != nil {
		return err
	}

	n, err := res.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return customerrors.ErrReactionNotSet
	}

	return nil
}

func (r *Repository) ListReactionsForMessages(
	ctx context.Context,
	messageIDs []uint64,
) (map[uint64][]domains.MessageReactionSummary, error) {
	if len(messageIDs) == 0 {
		return map[uint64][]domains.MessageReactionSummary{}, nil
	}

	stmt, params, err := sq.
		Select(
			fmt.Sprintf("%s.%s", messageReactionsTableName, messageIDColumnName),
			fmt.Sprintf("%s.%s", messageReactionsTableName, reactionIDColumnName),
			fmt.Sprintf("%s.%s", reactionsTableName, emojiColumnName),
			fmt.Sprintf("%s.%s", reactionsTableName, sortOrderColumnName),
			fmt.Sprintf("%s.%s", messageReactionsTableName, userIDColumnName),
		).
		From(messageReactionsTableName).
		Join(fmt.Sprintf(
			"%s ON %s.%s = %s.%s",
			reactionsTableName,
			reactionsTableName, idColumnName,
			messageReactionsTableName, reactionIDColumnName,
		)).
		Where(sq.Eq{
			fmt.Sprintf("%s.%s", messageReactionsTableName, messageIDColumnName): messageIDs,
		}).
		OrderBy(
			fmt.Sprintf("%s.%s %s", reactionsTableName, sortOrderColumnName, asc),
		).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	rows, err := r.tx.QueryContext(ctx, stmt, params...)
	if err != nil {
		return nil, err
	}

	defer func() {
		if err = rows.Close(); err != nil {
			logging.LogErrorContext(ctx, r.logger, "Failed to close SQL rows", err)
		}
	}()

	// SQL отдаёт по строке на триплет (message_id, reaction_id, user_id): для одного
	// сообщения одна и та же реакция придёт несколькими строками — по одной на юзера.
	// Пример rows:
	//   msg=10, reaction=1 (👍), user=5
	//   msg=10, reaction=1 (👍), user=7
	//   msg=10, reaction=2 (❤️), user=5
	// Нужно собрать result[10] = [{👍, [5,7]}, {❤️, [5]}].
	//
	// positions[msgID][reactionID] хранит индекс уже добавленной сводки в result[msgID].
	// Когда встречаем следующую строку для той же (msg, reaction) — по индексу дописываем
	// userID в существующую запись, а не создаём дубль. В примере после первой строки:
	//   result[10] = [{👍, [5]}], positions[10][1] = 0
	// Вторая строка (msg=10, reaction=1, user=7) → positions[10][1] уже есть, индекс 0 →
	//   result[10][0].UserIDs = append(..., 7) → result[10] = [{👍, [5,7]}].
	result := make(map[uint64][]domains.MessageReactionSummary)
	positions := make(map[uint64]map[uint64]int)

	for rows.Next() {
		rowPg := MessageReactionRowPg{}
		columns := pg.GetEntityColumns(&rowPg)

		if err = rows.Scan(columns...); err != nil {
			return nil, err
		}

		byReaction, ok := positions[rowPg.MessageID]
		if !ok {
			byReaction = make(map[uint64]int)
			positions[rowPg.MessageID] = byReaction
		}

		if pos, exists := byReaction[rowPg.ReactionID]; exists {
			result[rowPg.MessageID][pos].UserIDs = append(
				result[rowPg.MessageID][pos].UserIDs, rowPg.UserID,
			)

			continue
		}

		result[rowPg.MessageID] = append(result[rowPg.MessageID], domains.MessageReactionSummary{
			Reaction: domains.Reaction{
				ID:        rowPg.ReactionID,
				Emoji:     rowPg.Emoji,
				SortOrder: rowPg.SortOrder,
			},
			UserIDs: []uint64{rowPg.UserID},
		})
		byReaction[rowPg.ReactionID] = len(result[rowPg.MessageID]) - 1
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	return result, nil
}

type MessageReactionRowPg struct {
	MessageID  uint64
	ReactionID uint64
	Emoji      string
	SortOrder  uint64
	UserID     uint64
}
