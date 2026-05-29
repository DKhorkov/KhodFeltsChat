package common_test

import (
	"bytes"
	"image"
	"image/color"
	"image/jpeg"
	"image/png"
	"testing"

	"github.com/DKhorkov/kfc/internal/common"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func createTestJPEG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: 255, G: 0, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	_ = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 85})

	return buf.Bytes()
}

func createTestPNG(width, height int) []byte {
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	for y := range height {
		for x := range width {
			img.Set(x, y, color.RGBA{R: 0, G: 255, B: 0, A: 255})
		}
	}

	var buf bytes.Buffer
	_ = png.Encode(&buf, img)

	return buf.Bytes()
}

func TestDecodeImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name    string
		data    []byte
		wantErr bool
	}{
		{
			name:    "decode JPEG",
			data:    createTestJPEG(10, 10),
			wantErr: false,
		},
		{
			name:    "decode PNG",
			data:    createTestPNG(10, 10),
			wantErr: false,
		},
		{
			name:    "invalid data",
			data:    []byte("not an image"),
			wantErr: true,
		},
		{
			name:    "empty data",
			data:    []byte{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			img, err := common.DecodeImage(tt.data)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Nil(t, img)
			} else {
				assert.NoError(t, err)
				assert.NotNil(t, img)
			}
		})
	}
}

func TestResizeImage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		srcWidth  int
		srcHeight int
		dstWidth  int
		dstHeight int
	}{
		{
			name:      "downscale",
			srcWidth:  100,
			srcHeight: 100,
			dstWidth:  50,
			dstHeight: 50,
		},
		{
			name:      "upscale",
			srcWidth:  10,
			srcHeight: 10,
			dstWidth:  100,
			dstHeight: 100,
		},
		{
			name:      "non-square",
			srcWidth:  200,
			srcHeight: 100,
			dstWidth:  256,
			dstHeight: 256,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			src := image.NewRGBA(image.Rect(0, 0, tt.srcWidth, tt.srcHeight))

			result := common.ResizeImage(src, tt.dstWidth, tt.dstHeight)

			require.NotNil(t, result)
			assert.Equal(t, tt.dstWidth, result.Bounds().Dx())
			assert.Equal(t, tt.dstHeight, result.Bounds().Dy())
		})
	}
}

func TestExtractUUIDFromURL(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		url  string
		want string
	}{
		{
			name: "full URL with UUID",
			url:  "http://localhost:8080/api/files/download/550e8400-e29b-41d4-a716-446655440000.jpg",
			want: "550e8400-e29b-41d4-a716-446655440000",
		},
		{
			name: "URL without extension",
			url:  "http://localhost:8080/api/files/download/some-file",
			want: "some-file",
		},
		{
			name: "just filename",
			url:  "test-uuid.jpg",
			want: "test-uuid",
		},
		{
			name: "empty string",
			url:  "",
			want: ".",
		},
		{
			name: "URL with path only",
			url:  "/files/download/abc-123.jpg",
			want: "abc-123",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := common.ExtractUUIDFromURL(tt.url)
			assert.Equal(t, tt.want, got)
		})
	}
}
