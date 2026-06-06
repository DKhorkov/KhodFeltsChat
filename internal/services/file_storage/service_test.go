package file_storage_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	filestorage "github.com/DKhorkov/kfc/internal/services/file_storage"
	mockrepositories "github.com/DKhorkov/kfc/mocks/repositories"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestService_Upload(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		path          string
		setupMocks    func(*mockrepositories.MockFileStorageRepository)
		expectedError error
	}{
		{
			name: "successful upload",
			path: "test.jpg",
			setupMocks: func(mockRepo *mockrepositories.MockFileStorageRepository) {
				mockRepo.EXPECT().
					Upload(gomock.Any(), "test.jpg", gomock.Any()).
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "upload error",
			path: "test.jpg",
			setupMocks: func(mockRepo *mockrepositories.MockFileStorageRepository) {
				mockRepo.EXPECT().
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
			mockRepo := mockrepositories.NewMockFileStorageRepository(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockRepo)
			}

			service := filestorage.New(mockRepo)
			err := service.Upload(context.Background(), tt.path, bytes.NewReader([]byte("data")))

			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestService_Download(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name           string
		path           string
		setupMocks     func(*mockrepositories.MockFileStorageRepository)
		expectedResult []byte
		expectedError  error
	}{
		{
			name: "successful download",
			path: "test.jpg",
			setupMocks: func(mockRepo *mockrepositories.MockFileStorageRepository) {
				mockRepo.EXPECT().
					Download(gomock.Any(), "test.jpg").
					Return([]byte("file content"), nil)
			},
			expectedResult: []byte("file content"),
			expectedError:  nil,
		},
		{
			name: "file not found",
			path: "nonexistent.jpg",
			setupMocks: func(mockRepo *mockrepositories.MockFileStorageRepository) {
				mockRepo.EXPECT().
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
			mockRepo := mockrepositories.NewMockFileStorageRepository(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockRepo)
			}

			service := filestorage.New(mockRepo)
			result, err := service.Download(context.Background(), tt.path)

			assert.Equal(t, tt.expectedResult, result)

			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				require.NoError(t, err)
			}
		})
	}
}

func TestService_Delete(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name          string
		path          string
		setupMocks    func(*mockrepositories.MockFileStorageRepository)
		expectedError error
	}{
		{
			name: "successful delete",
			path: "test.jpg",
			setupMocks: func(mockRepo *mockrepositories.MockFileStorageRepository) {
				mockRepo.EXPECT().
					Delete(gomock.Any(), "test.jpg").
					Return(nil)
			},
			expectedError: nil,
		},
		{
			name: "delete error",
			path: "test.jpg",
			setupMocks: func(mockRepo *mockrepositories.MockFileStorageRepository) {
				mockRepo.EXPECT().
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
			mockRepo := mockrepositories.NewMockFileStorageRepository(ctrl)

			if tt.setupMocks != nil {
				tt.setupMocks(mockRepo)
			}

			service := filestorage.New(mockRepo)
			err := service.Delete(context.Background(), tt.path)

			if tt.expectedError != nil {
				assert.EqualError(t, err, tt.expectedError.Error())
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
