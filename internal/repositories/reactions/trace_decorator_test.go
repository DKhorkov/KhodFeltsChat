package reactions_test

import (
	"context"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/kfc/internal/interfaces"
	"github.com/DKhorkov/kfc/internal/repositories/reactions"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	"github.com/DKhorkov/libs/tracing"
	mocktracing "github.com/DKhorkov/libs/tracing/mocks"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

func TestNewTraceDecorator(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		traceProvider tracing.Provider
		spanConfig    tracing.SpanConfig
		base          interfaces.ReactionsRepository
	}{
		{
			name:          "create trace decorator with valid params",
			traceProvider: mocktracing.NewMockProvider(gomock.NewController(t)),
			spanConfig: tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			},
			base: mockrepositories.NewMockReactionsRepository(gomock.NewController(t)),
		},
		{
			name:          "create trace decorator with nil base",
			traceProvider: mocktracing.NewMockProvider(gomock.NewController(t)),
			spanConfig:    tracing.SpanConfig{},
			base:          nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decorator := reactions.NewTraceDecorator(tt.traceProvider, tt.spanConfig, tt.base)
			assert.NotNil(t, decorator)
		})
	}
}

func setupDecorator(t *testing.T) (
	*reactions.TraceDecorator,
	*mockrepositories.MockReactionsRepository,
	*mocktracing.MockProvider,
) {
	t.Helper()

	ctrl := gomock.NewController(t)
	provider := mocktracing.NewMockProvider(ctrl)
	base := mockrepositories.NewMockReactionsRepository(ctrl)
	span := mocktracing.NewMockSpan()

	provider.EXPECT().
		Span(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
			return ctx, span
		}).
		AnyTimes()

	dec := reactions.NewTraceDecorator(provider, tracing.SpanConfig{}, base)

	return dec, base, provider
}

func TestTraceDecorator_ListReactions(t *testing.T) {
	t.Parallel()

	dec, base, _ := setupDecorator(t)
	expected := []domains.Reaction{{ID: 1, Emoji: "👍"}}
	base.EXPECT().ListReactions(gomock.Any()).Return(expected, nil)

	result, err := dec.ListReactions(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTraceDecorator_GetReactionByID(t *testing.T) {
	t.Parallel()

	dec, base, _ := setupDecorator(t)
	expected := &domains.Reaction{ID: 1, Emoji: "👍"}
	base.EXPECT().GetReactionByID(gomock.Any(), uint64(1)).Return(expected, nil)

	result, err := dec.GetReactionByID(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTraceDecorator_GetReactionByID_NotFound(t *testing.T) {
	t.Parallel()

	dec, base, _ := setupDecorator(t)
	base.EXPECT().
		GetReactionByID(gomock.Any(), uint64(999)).
		Return(nil, customerrors.ErrReactionNotFound)

	result, err := dec.GetReactionByID(context.Background(), 999)
	assert.Nil(t, result)
	assert.ErrorIs(t, err, customerrors.ErrReactionNotFound)
}

func TestTraceDecorator_AddMessageReaction(t *testing.T) {
	t.Parallel()

	dec, base, _ := setupDecorator(t)
	dto := domains.MessageReactionDTO{MessageID: 1, ReactionID: 2, UserID: 3}
	base.EXPECT().AddMessageReaction(gomock.Any(), dto).Return(nil)

	assert.NoError(t, dec.AddMessageReaction(context.Background(), dto))
}

func TestTraceDecorator_AddMessageReaction_Duplicate(t *testing.T) {
	t.Parallel()

	dec, base, _ := setupDecorator(t)
	dto := domains.MessageReactionDTO{MessageID: 1, ReactionID: 2, UserID: 3}
	base.EXPECT().
		AddMessageReaction(gomock.Any(), dto).
		Return(customerrors.ErrReactionAlreadyExists)

	err := dec.AddMessageReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrReactionAlreadyExists)
}

func TestTraceDecorator_RemoveMessageReaction_Deleted(t *testing.T) {
	t.Parallel()

	dec, base, _ := setupDecorator(t)
	dto := domains.MessageReactionDTO{MessageID: 1, ReactionID: 2, UserID: 3}
	base.EXPECT().RemoveMessageReaction(gomock.Any(), dto).Return(nil)

	assert.NoError(t, dec.RemoveMessageReaction(context.Background(), dto))
}

func TestTraceDecorator_RemoveMessageReaction_NotSet(t *testing.T) {
	t.Parallel()

	dec, base, _ := setupDecorator(t)
	dto := domains.MessageReactionDTO{MessageID: 1, ReactionID: 2, UserID: 3}
	base.EXPECT().
		RemoveMessageReaction(gomock.Any(), dto).
		Return(customerrors.ErrReactionNotSet)

	err := dec.RemoveMessageReaction(context.Background(), dto)
	assert.ErrorIs(t, err, customerrors.ErrReactionNotSet)
}

func TestTraceDecorator_ListReactionsForMessages(t *testing.T) {
	t.Parallel()

	dec, base, _ := setupDecorator(t)
	ids := []uint64{1, 2}
	expected := map[uint64][]domains.MessageReactionSummary{
		1: {{Reaction: domains.Reaction{ID: 1, Emoji: "👍"}, UserIDs: []uint64{10}}},
	}
	base.EXPECT().ListReactionsForMessages(gomock.Any(), ids).Return(expected, nil)

	result, err := dec.ListReactionsForMessages(context.Background(), ids)
	assert.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestTraceDecorator_PropagatesBaseError(t *testing.T) {
	t.Parallel()

	dec, base, _ := setupDecorator(t)
	base.EXPECT().ListReactions(gomock.Any()).Return(nil, errors.New("boom"))

	result, err := dec.ListReactions(context.Background())
	assert.Error(t, err)
	assert.Nil(t, result)
}
