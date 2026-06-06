package file_storage

import (
	"context"
	"io"

	"github.com/DKhorkov/kfc/internal/config"
	"github.com/DKhorkov/kfc/internal/interfaces"
)

type UseCases struct {
	fileStorageService interfaces.FileStorageService
	fileStorageConfig  config.FileStorageConfig
}

func New(
	fileStorageService interfaces.FileStorageService,
	fileStorageConfig config.FileStorageConfig,
) *UseCases {
	return &UseCases{
		fileStorageService: fileStorageService,
		fileStorageConfig:  fileStorageConfig,
	}
}

func (u *UseCases) Upload(ctx context.Context, path string, data io.Reader) (string, error) {
	if err := u.fileStorageService.Upload(ctx, path, data); err != nil {
		return "", err
	}

	return u.fileStorageConfig.BaseDownloadURL + "/" + path, nil
}

func (u *UseCases) Download(ctx context.Context, path string) ([]byte, error) {
	return u.fileStorageService.Download(ctx, path)
}

func (u *UseCases) Delete(ctx context.Context, path string) error {
	return u.fileStorageService.Delete(ctx, path)
}
