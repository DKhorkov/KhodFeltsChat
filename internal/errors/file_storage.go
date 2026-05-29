package errors

import "errors"

var (
	ErrFileNotFound       = errors.New("file not found")
	ErrInvalidImageFormat = errors.New("invalid image format: supported formats are JPEG, PNG, WebP, GIF")
	ErrFileTooLarge       = errors.New("file too large")
)
