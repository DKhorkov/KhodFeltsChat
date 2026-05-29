package common

import (
	"bytes"
	"image"
	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
	"net/url"
	"path"
	"strings"

	"golang.org/x/image/draw"
	"golang.org/x/image/webp"
)

func DecodeImage(data []byte) (image.Image, error) {
	reader := bytes.NewReader(data)

	img, _, err := image.Decode(reader)
	if err == nil {
		return img, nil
	}

	reader.Reset(data)

	img, err = webp.Decode(reader)
	if err == nil {
		return img, nil
	}

	return nil, err
}

func ResizeImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)

	return dst
}

func ExtractUUIDFromURL(data string) string {
	u, err := url.Parse(data)
	if err != nil {
		return ""
	}

	base := path.Base(u.Path)

	return strings.TrimSuffix(base, ".jpg")
}
