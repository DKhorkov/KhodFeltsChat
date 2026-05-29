package file_storage_test

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	filestorage "github.com/DKhorkov/kfc/internal/repositories/file_storage"
	"github.com/DKhorkov/libs/logging/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/mock/gomock"
)

func TestRepository_Upload(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	basePath := t.TempDir()
	repo := filestorage.New(basePath, mocks.NewMockLogger(ctrl))

	data := []byte("test file content")
	err := repo.Upload(context.Background(), "test.jpg", bytes.NewReader(data))
	require.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(basePath, "test.jpg"))
	require.NoError(t, err)
	assert.Equal(t, data, content)
}

func TestRepository_Download(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	basePath := t.TempDir()
	repo := filestorage.New(basePath, mocks.NewMockLogger(ctrl))

	expected := []byte("test file content")
	err := os.WriteFile(filepath.Join(basePath, "test.jpg"), expected, 0o644)
	require.NoError(t, err)

	result, err := repo.Download(context.Background(), "test.jpg")
	require.NoError(t, err)
	assert.Equal(t, expected, result)
}

func TestRepository_Download_NotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	basePath := t.TempDir()
	repo := filestorage.New(basePath, mocks.NewMockLogger(ctrl))

	_, err := repo.Download(context.Background(), "nonexistent.jpg")
	assert.Error(t, err)
}

func TestRepository_Delete(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	basePath := t.TempDir()
	repo := filestorage.New(basePath, mocks.NewMockLogger(ctrl))

	filePath := filepath.Join(basePath, "test.jpg")
	err := os.WriteFile(filePath, []byte("content"), 0o644)
	require.NoError(t, err)

	err = repo.Delete(context.Background(), "test.jpg")
	require.NoError(t, err)

	_, err = os.Stat(filePath)
	assert.True(t, os.IsNotExist(err))
}

func TestRepository_Delete_NotFound(t *testing.T) {
	t.Parallel()

	ctrl := gomock.NewController(t)
	basePath := t.TempDir()
	repo := filestorage.New(basePath, mocks.NewMockLogger(ctrl))

	err := repo.Delete(context.Background(), "nonexistent.jpg")
	assert.NoError(t, err, "deleting non-existent file should not error")
}
