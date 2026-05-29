package file_storage_test

import (
	"bytes"
	"context"
	"errors"
	"testing"

	"github.com/DKhorkov/kfc/internal/config"
	filestorage "github.com/DKhorkov/kfc/internal/usecases/file_storage"
	mockservices "github.com/DKhorkov/kfc/mocks/services"
	"github.com/stretchr/testify/assert"
	"go.uber.org/mock/gomock"
)

func TestUseCases_Upload(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockFileStorageService func(*mockservices.MockFileStorageService)
	}

	type args struct {
		ctx  context.Context
		path string
		data *bytes.Reader
	}

	fileStorageConfig := config.FileStorageConfig{
		BaseDownloadURL: "http://localhost:8080/api/files/download",
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    string
		wantErr bool
		err     error
	}{
		{
			name: "successfully upload",
			fields: fields{
				mockFileStorageService: func(fs *mockservices.MockFileStorageService) {
					fs.EXPECT().
						Upload(gomock.Any(), "test.jpg", gomock.Any()).
						Return(nil)
				},
			},
			args: args{
				ctx:  context.Background(),
				path: "test.jpg",
				data: bytes.NewReader([]byte("data")),
			},
			want:    "http://localhost:8080/api/files/download/test.jpg",
			wantErr: false,
		},
		{
			name: "service returns error",
			fields: fields{
				mockFileStorageService: func(fs *mockservices.MockFileStorageService) {
					fs.EXPECT().
						Upload(gomock.Any(), "test.jpg", gomock.Any()).
						Return(errors.New("disk full"))
				},
			},
			args: args{
				ctx:  context.Background(),
				path: "test.jpg",
				data: bytes.NewReader([]byte("data")),
			},
			want:    "",
			wantErr: true,
			err:     errors.New("disk full"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockService := mockservices.NewMockFileStorageService(ctrl)

			if tt.fields.mockFileStorageService != nil {
				tt.fields.mockFileStorageService(mockService)
			}

			u := filestorage.New(mockService, fileStorageConfig)
			got, err := u.Upload(tt.args.ctx, tt.args.path, tt.args.data)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
				assert.Empty(t, got)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUseCases_Download(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockFileStorageService func(*mockservices.MockFileStorageService)
	}

	type args struct {
		ctx  context.Context
		path string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		want    []byte
		wantErr bool
		err     error
	}{
		{
			name: "successfully download",
			fields: fields{
				mockFileStorageService: func(fs *mockservices.MockFileStorageService) {
					fs.EXPECT().
						Download(gomock.Any(), "test.jpg").
						Return([]byte("file-data"), nil)
				},
			},
			args: args{
				ctx:  context.Background(),
				path: "test.jpg",
			},
			want:    []byte("file-data"),
			wantErr: false,
		},
		{
			name: "service returns error",
			fields: fields{
				mockFileStorageService: func(fs *mockservices.MockFileStorageService) {
					fs.EXPECT().
						Download(gomock.Any(), "missing.jpg").
						Return(nil, errors.New("file not found"))
				},
			},
			args: args{
				ctx:  context.Background(),
				path: "missing.jpg",
			},
			want:    nil,
			wantErr: true,
			err:     errors.New("file not found"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockService := mockservices.NewMockFileStorageService(ctrl)

			if tt.fields.mockFileStorageService != nil {
				tt.fields.mockFileStorageService(mockService)
			}

			u := filestorage.New(mockService, config.FileStorageConfig{})
			got, err := u.Download(tt.args.ctx, tt.args.path)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.want, got)
			}
		})
	}
}

func TestUseCases_Delete(t *testing.T) {
	t.Parallel()

	type fields struct {
		mockFileStorageService func(*mockservices.MockFileStorageService)
	}

	type args struct {
		ctx  context.Context
		path string
	}

	tests := []struct {
		name    string
		fields  fields
		args    args
		wantErr bool
		err     error
	}{
		{
			name: "successfully delete",
			fields: fields{
				mockFileStorageService: func(fs *mockservices.MockFileStorageService) {
					fs.EXPECT().
						Delete(gomock.Any(), "test.jpg").
						Return(nil)
				},
			},
			args: args{
				ctx:  context.Background(),
				path: "test.jpg",
			},
			wantErr: false,
		},
		{
			name: "service returns error",
			fields: fields{
				mockFileStorageService: func(fs *mockservices.MockFileStorageService) {
					fs.EXPECT().
						Delete(gomock.Any(), "test.jpg").
						Return(errors.New("permission denied"))
				},
			},
			args: args{
				ctx:  context.Background(),
				path: "test.jpg",
			},
			wantErr: true,
			err:     errors.New("permission denied"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctrl := gomock.NewController(t)
			mockService := mockservices.NewMockFileStorageService(ctrl)

			if tt.fields.mockFileStorageService != nil {
				tt.fields.mockFileStorageService(mockService)
			}

			u := filestorage.New(mockService, config.FileStorageConfig{})
			err := u.Delete(tt.args.ctx, tt.args.path)

			if tt.wantErr {
				assert.Error(t, err)
				if tt.err != nil {
					assert.Contains(t, err.Error(), tt.err.Error())
				}
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
