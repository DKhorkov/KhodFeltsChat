package file_storage_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	filestorage "github.com/DKhorkov/kfc/internal/repositories/file_storage"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	"github.com/DKhorkov/libs/tracing"
	mocktracing "github.com/DKhorkov/libs/tracing/mocks"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/otel/trace"
	"go.uber.org/mock/gomock"
)

func TestTraceDecorator_Upload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		path          string
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockFileStorageRepository, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name: "successful upload with tracing",
			path: "test.jpg",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockFileStorageRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					Upload(gomock.Any(), "test.jpg", gomock.Any()).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "upload error with tracing",
			path: "test.jpg",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockFileStorageRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					Upload(gomock.Any(), "test.jpg", gomock.Any()).
					Return(errors.New("disk full"))
			},
			expectedError: errors.New("disk full"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockFileStorageRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := filestorage.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			err := decorator.Upload(context.Background(), tt.path, bytes.NewReader([]byte("data")))

			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTraceDecorator_Download(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		path           string
		setupMocks     func(*mocktracing.MockProvider, *mockrepositories.MockFileStorageRepository, *mocktracing.MockSpan)
		expectedResult []byte
		expectedError  error
	}{
		{
			name: "successful download with tracing",
			path: "test.jpg",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockFileStorageRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					Download(gomock.Any(), "test.jpg").
					Return([]byte("file content"), nil)
			},
			expectedResult: []byte("file content"),
			expectedError:  nil,
		},
		{
			name: "file not found with tracing",
			path: "nonexistent.jpg",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockFileStorageRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					Download(gomock.Any(), "nonexistent.jpg").
					Return(nil, errors.New("file not found"))
			},
			expectedResult: nil,
			expectedError:  errors.New("file not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockFileStorageRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := filestorage.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			result, err := decorator.Download(context.Background(), tt.path)

			assert.Equal(t, tt.expectedResult, result)

			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestTraceDecorator_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		path          string
		setupMocks    func(*mocktracing.MockProvider, *mockrepositories.MockFileStorageRepository, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name: "successful delete with tracing",
			path: "test.jpg",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockFileStorageRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					Delete(gomock.Any(), "test.jpg").
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "delete error with tracing",
			path: "test.jpg",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockrepositories.MockFileStorageRepository,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					Delete(gomock.Any(), "test.jpg").
					Return(errors.New("permission denied"))
			},
			expectedError: errors.New("permission denied"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)

			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockrepositories.NewMockFileStorageRepository(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			spanConfig := tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{Name: "start"},
					End:   tracing.SpanEventConfig{Name: "end"},
				},
			}

			if tt.setupMocks != nil {
				tt.setupMocks(mockProvider, mockBase, mockSpan)
			}

			decorator := filestorage.NewTraceDecorator(mockProvider, spanConfig, mockBase)

			err := decorator.Delete(context.Background(), tt.path)

			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
