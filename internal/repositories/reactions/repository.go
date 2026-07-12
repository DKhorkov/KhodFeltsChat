package reactions

import (
	"context"
	"database/sql"
	"errors"
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
	createdAtColumnName  = "created_at"

	returningIDSuffix = "RETURNING id"

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
		Select(idColumnName, emojiColumnName).
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
		var re domains.Reaction
		if err = rows.Scan(&re.ID, &re.Emoji); err != nil {
			return nil, err
		}

		reactions = append(reactions, re)
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
		Select(idColumnName, emojiColumnName).
		From(reactionsTableName).
		Where(sq.Eq{idColumnName: id}).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return nil, err
	}

	var re domains.Reaction
	if err = r.tx.QueryRowContext(ctx, stmt, params...).Scan(&re.ID, &re.Emoji); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, customerrors.ErrReactionNotFound
		}

		return nil, err
	}

	return &re, nil
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
			"ON CONFLICT (%s, %s, %s) DO NOTHING %s",
			messageIDColumnName, userIDColumnName, reactionIDColumnName,
			returningIDSuffix,
		)).
		PlaceholderFormat(sq.Dollar).
		ToSql()
	if err != nil {
		return err
	}

	var id uint64
	if err = r.tx.QueryRowContext(ctx, stmt, params...).Scan(&id); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return customerrors.ErrReactionAlreadyExists
		}

		return err
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
			fmt.Sprintf("%s.%s %s", messageReactionsTableName, messageIDColumnName, asc),
			fmt.Sprintf("%s.%s %s", reactionsTableName, sortOrderColumnName, asc),
			fmt.Sprintf("%s.%s %s", messageReactionsTableName, createdAtColumnName, asc),
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

	agg := make(map[uint64]map[uint64]*domains.MessageReactionSummary)
	keepOrder := make(map[uint64][]uint64)

	for rows.Next() {
		var (
			msgID, reactionID, userID uint64
			emoji                     string
		)

		if err = rows.Scan(&msgID, &reactionID, &emoji, &userID); err != nil {
			return nil, err
		}

		byReaction, ok := agg[msgID]
		if !ok {
			byReaction = make(map[uint64]*domains.MessageReactionSummary)
			agg[msgID] = byReaction
		}

		summary, ok := byReaction[reactionID]
		if !ok {
			summary = &domains.MessageReactionSummary{
				Reaction: domains.Reaction{ID: reactionID, Emoji: emoji},
				UserIDs:  []uint64{},
			}
			byReaction[reactionID] = summary

			keepOrder[msgID] = append(keepOrder[msgID], reactionID)
		}

		summary.UserIDs = append(summary.UserIDs, userID)
	}

	if err = rows.Err(); err != nil {
		return nil, err
	}

	result := make(map[uint64][]domains.MessageReactionSummary, len(agg))

	for msgID, order := range keepOrder {
		summaries := make([]domains.MessageReactionSummary, 0, len(order))
		for _, rid := range order {
			summaries = append(summaries, *agg[msgID][rid])
		}

		result[msgID] = summaries
	}

	return result, nil
}
