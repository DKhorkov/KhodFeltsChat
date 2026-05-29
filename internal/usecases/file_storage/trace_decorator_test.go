package file_storage_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/interfaces"
	filestorage "github.com/DKhorkov/kfc/internal/usecases/file_storage"
	mockusecases "github.com/DKhorkov/kfc/mocks/usecases"
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
		base          interfaces.FileStorageUseCases
	}{
		{
			name:          "create trace decorator with valid params",
			traceProvider: mocktracing.NewMockProvider(gomock.NewController(t)),
			spanConfig: tracing.SpanConfig{
				Name: "test-span",
				Opts: []trace.SpanStartOption{},
				Events: tracing.SpanEventsConfig{
					Start: tracing.SpanEventConfig{
						Name: "start",
						Opts: []trace.EventOption{},
					},
					End: tracing.SpanEventConfig{
						Name: "end",
						Opts: []trace.EventOption{},
					},
				},
			},
			base: mockusecases.NewMockFileStorageUseCases(gomock.NewController(t)),
		},
		{
			name:          "create trace decorator with nil base",
			traceProvider: mocktracing.NewMockProvider(gomock.NewController(t)),
			spanConfig:    tracing.SpanConfig{},
			base:          nil,
		},
		{
			name:          "create trace decorator with nil provider",
			traceProvider: nil,
			spanConfig:    tracing.SpanConfig{},
			base:          mockusecases.NewMockFileStorageUseCases(gomock.NewController(t)),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			decorator := filestorage.NewTraceDecorator(tt.traceProvider, tt.spanConfig, tt.base)

			assert.NotNil(t, decorator)
		})
	}
}

func TestTraceDecorator_Upload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		path          string
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockFileStorageUseCases, *mocktracing.MockSpan)
		expectedURL   string
		expectedError error
	}{
		{
			name: "successful upload with tracing",
			path: "test.jpg",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockFileStorageUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					Upload(gomock.Any(), "test.jpg", gomock.Any()).
					Return("http://localhost/files/download/test.jpg", nil)
			},
			expectedURL:   "http://localhost/files/download/test.jpg",
			expectedError: nil,
		},
		{
			name: "upload error with tracing",
			path: "test.jpg",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockFileStorageUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					Upload(gomock.Any(), "test.jpg", gomock.Any()).
					Return("", errors.New("disk full"))
			},
			expectedURL:   "",
			expectedError: errors.New("disk full"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockFileStorageUseCases(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			tt.setupMocks(mockProvider, mockBase, mockSpan)

			decorator := filestorage.NewTraceDecorator(
				mockProvider,
				tracing.SpanConfig{
					Events: tracing.SpanEventsConfig{
						Start: tracing.SpanEventConfig{Name: "start"},
						End:   tracing.SpanEventConfig{Name: "end"},
					},
				},
				mockBase,
			)

			result, err := decorator.Upload(
				context.Background(),
				tt.path,
				bytes.NewReader([]byte("data")),
			)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedURL, result)
		})
	}
}

func TestTraceDecorator_Download(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		path          string
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockFileStorageUseCases, *mocktracing.MockSpan)
		expectedData  []byte
		expectedError error
	}{
		{
			name: "successful download with tracing",
			path: "test.jpg",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockFileStorageUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(ctx context.Context, _ string, _ ...trace.SpanStartOption) (context.Context, trace.Span) {
						return ctx, mockSpan
					})

				mockBase.EXPECT().
					Download(gomock.Any(), "test.jpg").
					Return([]byte("file-data"), nil)
			},
			expectedData:  []byte("file-data"),
			expectedError: nil,
		},
		{
			name: "download error with tracing",
			path: "missing.jpg",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockFileStorageUseCases,
				mockSpan *mocktracing.MockSpan,
			) {
				mockProvider.EXPECT().
					Span(gomock.Any(), gomock.Any(), gomock.Any()).
					Return(context.Background(), mockSpan)

				mockBase.EXPECT().
					Download(gomock.Any(), "missing.jpg").
					Return(nil, errors.New("not found"))
			},
			expectedData:  nil,
			expectedError: errors.New("not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockProvider := mocktracing.NewMockProvider(ctrl)
			mockBase := mockusecases.NewMockFileStorageUseCases(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			tt.setupMocks(mockProvider, mockBase, mockSpan)

			decorator := filestorage.NewTraceDecorator(
				mockProvider,
				tracing.SpanConfig{
					Events: tracing.SpanEventsConfig{
						Start: tracing.SpanEventConfig{Name: "start"},
						End:   tracing.SpanEventConfig{Name: "end"},
					},
				},
				mockBase,
			)

			result, err := decorator.Download(context.Background(), tt.path)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}

			assert.Equal(t, tt.expectedData, result)
		})
	}
}

func TestTraceDecorator_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		path          string
		setupMocks    func(*mocktracing.MockProvider, *mockusecases.MockFileStorageUseCases, *mocktracing.MockSpan)
		expectedError error
	}{
		{
			name: "successful delete with tracing",
			path: "test.jpg",
			setupMocks: func(
				mockProvider *mocktracing.MockProvider,
				mockBase *mockusecases.MockFileStorageUseCases,
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
				mockBase *mockusecases.MockFileStorageUseCases,
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
			mockBase := mockusecases.NewMockFileStorageUseCases(ctrl)
			mockSpan := mocktracing.NewMockSpan()

			tt.setupMocks(mockProvider, mockBase, mockSpan)

			decorator := filestorage.NewTraceDecorator(
				mockProvider,
				tracing.SpanConfig{
					Events: tracing.SpanEventsConfig{
						Start: tracing.SpanEventConfig{Name: "start"},
						End:   tracing.SpanEventConfig{Name: "end"},
					},
				},
				mockBase,
			)

			err := decorator.Delete(context.Background(), tt.path)

			if tt.expectedError != nil {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
