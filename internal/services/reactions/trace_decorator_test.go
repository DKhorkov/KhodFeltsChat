package reactions_test

import (
	"context"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	service "github.com/DKhorkov/kfc/internal/services/reactions"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/DKhorkov/libs/tracing"
	mocktracing "github.com/DKhorkov/libs/tracing/mocks"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

func setupServiceDecorator(t *testing.T) (
	*service.TraceDecorator,
	*mockservices.MockReactionsService,
) {
	t.Helper()

	ctrl := gomock.NewController(t)
	provider := mocktracing.NewMockProvider(ctrl)
	base := mockservices.NewMockReactionsService(ctrl)
	span := mocktracing.NewMockSpan()

	provider.EXPECT().
		Span(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
			return ctx, span
		}).
		AnyTimes()

	dec := service.NewTraceDecorator(provider, tracing.SpanConfig{}, base)

	return dec, base
}

func TestServiceTraceDecorator_ListReactions(t *testing.T) {
	t.Parallel()

	dec, base := setupServiceDecorator(t)
	expected := []domains.Reaction{{ID: 1, Emoji: "👍"}}
	base.EXPECT().ListReactions(gomock.Any()).Return(expected, nil)

	got, err := dec.ListReactions(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestServiceTraceDecorator_GetReactionByID(t *testing.T) {
	t.Parallel()

	dec, base := setupServiceDecorator(t)
	expected := &domains.Reaction{ID: 1, Emoji: "👍"}
	base.EXPECT().GetReactionByID(gomock.Any(), uint64(1)).Return(expected, nil)

	got, err := dec.GetReactionByID(context.Background(), 1)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestServiceTraceDecorator_AddMessageReaction(t *testing.T) {
	t.Parallel()

	dec, base := setupServiceDecorator(t)
	dto := domains.MessageReactionDTO{MessageID: 1, ReactionID: 2, UserID: 3}
	base.EXPECT().AddMessageReaction(gomock.Any(), dto).Return(nil)

	assert.NoError(t, dec.AddMessageReaction(context.Background(), dto))
}

func TestServiceTraceDecorator_RemoveMessageReaction(t *testing.T) {
	t.Parallel()

	dec, base := setupServiceDecorator(t)
	dto := domains.MessageReactionDTO{MessageID: 1, ReactionID: 2, UserID: 3}
	base.EXPECT().RemoveMessageReaction(gomock.Any(), dto).Return(nil)

	assert.NoError(t, dec.RemoveMessageReaction(context.Background(), dto))
}

func TestServiceTraceDecorator_ListReactionsForMessages(t *testing.T) {
	t.Parallel()

	dec, base := setupServiceDecorator(t)
	ids := []uint64{10}
	expected := map[uint64][]domains.MessageReactionSummary{
		10: {{Reaction: domains.Reaction{ID: 1, Emoji: "👍"}, UserIDs: []uint64{7}}},
	}
	base.EXPECT().ListReactionsForMessages(gomock.Any(), ids).Return(expected, nil)

	got, err := dec.ListReactionsForMessages(context.Background(), ids)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}
