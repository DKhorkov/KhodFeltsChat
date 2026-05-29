package file_storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	customerrors "github.com/DKhorkov/kfc/internal/errors"
	"github.com/DKhorkov/libs/logging"
)

type Repository struct {
	basePath string
	logger   logging.Logger
}

func New(basePath string, logger logging.Logger) *Repository {
	return &Repository{
		basePath: basePath,
		logger:   logger,
	}
}

func (r *Repository) Upload(ctx context.Context, path string, data io.Reader) error {
	fullPath := filepath.Join(r.basePath, path)

	if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
		return err
	}

	file, err := os.Create(fullPath)
	if err != nil {
		return err
	}

	defer func() {
		if err = file.Close(); err != nil {
			logging.LogErrorContext(ctx, r.logger, "Failed to close file", err)
		}
	}()

	_, err = io.Copy(file, data)

	return err
}

func (r *Repository) Download(_ context.Context, path string) ([]byte, error) {
	fullPath := filepath.Join(r.basePath, path)

	data, err := os.ReadFile(fullPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("%w: %s", customerrors.ErrFileNotFound, path)
		}

		return nil, err
	}

	return data, nil
}

func (r *Repository) Delete(_ context.Context, path string) error {
	fullPath := filepath.Join(r.basePath, path)

	err := os.Remove(fullPath)
	if err != nil && !os.IsNotExist(err) {
		return err
	}

	return nil
}
