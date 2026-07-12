package reactions_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	service "github.com/DKhorkov/kfc/internal/services/reactions"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	mockuow "github.com/DKhorkov/kfc/mocks/uow"
	pg "github.com/DKhorkov/libs/db/postgresql"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func newService(
	t *testing.T,
) (
	*service.Service,
	*mockuow.MockUnitOfWork,
	*mockrepositories.MockReactionsRepository,
) {
	t.Helper()

	ctrl := gomock.NewController(t)
	uow := mockuow.NewMockUnitOfWork(ctrl)
	repo := mockrepositories.NewMockReactionsRepository(ctrl)

	factory := func(_ pg.Transaction) interfaces.ReactionsRepository { return repo }
	svc := service.New(uow, factory)

	return svc, uow, repo
}

func expectUOWDoOnce(uow *mockuow.MockUnitOfWork) {
	uow.EXPECT().
		Do(gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, fn func(context.Context, pg.Transaction) error) error {
			tx := &struct{ pg.Transaction }{}

			return fn(ctx, tx)
		})
}

func TestService_ListReactions(t *testing.T) {
	t.Parallel()

	svc, uow, repo := newService(t)
	expected := []domains.Reaction{{ID: 1, Emoji: "👍"}, {ID: 2, Emoji: "❤️"}}

	expectUOWDoOnce(uow)
	repo.EXPECT().ListReactions(gomock.Any()).Return(expected, nil)

	got, err := svc.ListReactions(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestService_ListReactions_PropagatesError(t *testing.T) {
	t.Parallel()

	svc, uow, repo := newService(t)
	boom := errors.New("boom")

	expectUOWDoOnce(uow)
	repo.EXPECT().ListReactions(gomock.Any()).Return(nil, boom)

	got, err := svc.ListReactions(context.Background())
	assert.ErrorIs(t, err, boom)
	assert.Nil(t, got)
}

func TestService_GetReactionByID(t *testing.T) {
	t.Parallel()

	svc, uow, repo := newService(t)
	expected := &domains.Reaction{ID: 1, Emoji: "👍"}

	expectUOWDoOnce(uow)
	repo.EXPECT().GetReactionByID(gomock.Any(), uint64(1)).Return(expected, nil)

	got, err := svc.GetReactionByID(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestService_GetReactionByID_NotFound(t *testing.T) {
	t.Parallel()

	svc, uow, repo := newService(t)

	expectUOWDoOnce(uow)
	// Репо отдаёт sql.ErrNoRows — сервис маппит в доменный ErrReactionNotFound.
	repo.EXPECT().
		GetReactionByID(gomock.Any(), uint64(999)).
		Return(nil, sql.ErrNoRows)

	got, err := svc.GetReactionByID(context.Background(), 999)
	assert.ErrorIs(t, err, customerrors.ErrReactionNotFound)
	assert.Nil(t, got)
}

func TestService_AddMessageReaction(t *testing.T) {
	t.Parallel()

	svc, uow, repo := newService(t)
	dto := domains.MessageReactionDTO{MessageID: 1, ReactionID: 2, UserID: 3}

	expectUOWDoOnce(uow)
	repo.EXPECT().AddMessageReaction(gomock.Any(), dto).Return(nil)

	assert.NoError(t, svc.AddMessageReaction(context.Background(), dto))
}

func TestService_AddMessageReaction_Duplicate(t *testing.T) {
	t.Parallel()

	svc, uow, repo := newService(t)
	dto := domains.MessageReactionDTO{MessageID: 1, ReactionID: 2, UserID: 3}

	expectUOWDoOnce(uow)
	repo.EXPECT().
		AddMessageReaction(gomock.Any(), dto).
		Return(customerrors.ErrReactionAlreadyExists)

	err := svc.AddMessageReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrReactionAlreadyExists)
}

func TestService_RemoveMessageReaction_Success(t *testing.T) {
	t.Parallel()

	svc, uow, repo := newService(t)
	dto := domains.MessageReactionDTO{MessageID: 1, ReactionID: 2, UserID: 3}

	expectUOWDoOnce(uow)
	repo.EXPECT().RemoveMessageReaction(gomock.Any(), dto).Return(nil)

	assert.NoError(t, svc.RemoveMessageReaction(context.Background(), dto))
}

func TestService_RemoveMessageReaction_NotSet_PropagatesError(t *testing.T) {
	t.Parallel()

	svc, uow, repo := newService(t)
	dto := domains.MessageReactionDTO{MessageID: 1, ReactionID: 2, UserID: 3}

	expectUOWDoOnce(uow)
	repo.EXPECT().
		RemoveMessageReaction(gomock.Any(), dto).
		Return(customerrors.ErrReactionNotSet)

	err := svc.RemoveMessageReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrReactionNotSet)
}

func TestService_ListReactionsForMessages(t *testing.T) {
	t.Parallel()

	svc, uow, repo := newService(t)
	ids := []uint64{10, 20}
	expected := map[uint64][]domains.MessageReactionSummary{
		10: {{Reaction: domains.Reaction{ID: 1, Emoji: "👍"}, UserIDs: []uint64{1, 2}}},
	}

	expectUOWDoOnce(uow)
	repo.EXPECT().ListReactionsForMessages(gomock.Any(), ids).Return(expected, nil)

	got, err := svc.ListReactionsForMessages(context.Background(), ids)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestService_ListReactionsForMessages_RepoError(t *testing.T) {
	t.Parallel()

	svc, uow, repo := newService(t)
	ids := []uint64{10}
	boom := errors.New("boom")

	expectUOWDoOnce(uow)
	repo.EXPECT().ListReactionsForMessages(gomock.Any(), ids).Return(nil, boom)

	got, err := svc.ListReactionsForMessages(context.Background(), ids)
	assert.ErrorIs(t, err, boom)
	assert.Nil(t, got)
}
