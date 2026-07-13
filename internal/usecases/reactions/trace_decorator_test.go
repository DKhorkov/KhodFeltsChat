package reactions_test

import (
	"context"
	"testing"

	"github.com/DKhorkov/kfc/internal/domains"
	reactionsusecases "github.com/DKhorkov/kfc/internal/usecases/reactions"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
	"github.com/DKhorkov/libs/tracing"
	mocktracing "github.com/DKhorkov/libs/tracing/mocks"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

func setupUseCaseDecorator(t *testing.T) (
	*reactionsusecases.TraceDecorator,
	*mockusecases.MockReactionsUseCases,
) {
	t.Helper()

	ctrl := gomock.NewController(t)
	provider := mocktracing.NewMockProvider(ctrl)
	base := mockusecases.NewMockReactionsUseCases(ctrl)
	span := mocktracing.NewMockSpan()

	provider.EXPECT().
		Span(gomock.Any(), gomock.Any(), gomock.Any()).
		DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
			return ctx, span
		}).
		AnyTimes()

	dec := reactionsusecases.NewTraceDecorator(provider, tracing.SpanConfig{}, base)

	return dec, base
}

func TestUseCaseTraceDecorator_ListReactions(t *testing.T) {
	t.Parallel()

	dec, base := setupUseCaseDecorator(t)
	expected := []domains.Reaction{{ID: 1, Emoji: "👍"}}
	base.EXPECT().ListReactions(gomock.Any()).Return(expected, nil)

	got, err := dec.ListReactions(context.Background())
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestUseCaseTraceDecorator_AddReaction(t *testing.T) {
	t.Parallel()

	dec, base := setupUseCaseDecorator(t)
	dto := domains.MessageReactionDTO{MessageID: 1, ReactionID: 2, UserID: 3}
	expected := &domains.Reaction{ID: 2, Emoji: "👍"}
	base.EXPECT().AddReaction(gomock.Any(), dto).Return(expected, nil)

	got, err := dec.AddReaction(context.Background(), dto)
	assert.NoError(t, err)
	assert.Equal(t, expected, got)
}

func TestUseCaseTraceDecorator_RemoveReaction(t *testing.T) {
	t.Parallel()

	dec, base := setupUseCaseDecorator(t)
	dto := domains.MessageReactionDTO{MessageID: 1, ReactionID: 2, UserID: 3}
	base.EXPECT().RemoveReaction(gomock.Any(), dto).Return(nil)

	assert.NoError(t, dec.RemoveReaction(context.Background(), dto))
}
