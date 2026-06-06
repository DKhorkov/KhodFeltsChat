package file_storage

import (
	"context"
	"io"

	"github.com/DKhorkov/kfc/internal/interfaces"
)

type Service struct {
	fileStorageRepository interfaces.FileStorageRepository
}

func New(
	fileStorageRepository interfaces.FileStorageRepository,
) *Service {
	return &Service{
		fileStorageRepository: fileStorageRepository,
	}
}

func (s *Service) Upload(ctx context.Context, path string, data io.Reader) error {
	return s.fileStorageRepository.Upload(ctx, path, data)
}

func (s *Service) Download(ctx context.Context, path string) ([]byte, error) {
	return s.fileStorageRepository.Download(ctx, path)
}

func (s *Service) Delete(ctx context.Context, path string) error {
	return s.fileStorageRepository.Delete(ctx, path)
}
