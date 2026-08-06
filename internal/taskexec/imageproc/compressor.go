package imageproc

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/jpeg"
	_ "image/png"
	"strings"

	_ "image/gif"

	"golang.org/x/image/draw"
	_ "golang.org/x/image/webp"
)

const (
	DefaultMaxLongSide    = 2048
	DefaultJPEGQuality    = 85
	DefaultSmallThreshold = 500 * 1024
)

type CompressOptions struct {
	MaxLongSide    int
	JPEGQuality    int
	SkipSmall      bool
	SmallThreshold int
}

func defaultOptions(opts *CompressOptions) *CompressOptions {
	if opts == nil {
		opts = &CompressOptions{
			SkipSmall: true,
		}
	}
	if opts.MaxLongSide <= 0 {
		opts.MaxLongSide = DefaultMaxLongSide
	}
	if opts.JPEGQuality <= 0 || opts.JPEGQuality > 100 {
		opts.JPEGQuality = DefaultJPEGQuality
	}
	if opts.SmallThreshold <= 0 {
		opts.SmallThreshold = DefaultSmallThreshold
	}
	return opts
}

type CompressResult struct {
	DataURL        string
	OriginalSize   int
	CompressedSize int
	Format         string
	Width          int
	Height         int
}

func Compress(dataURL string, opts *CompressOptions) (*CompressResult, error) {
	opts = defaultOptions(opts)

	raw, format, err := decodeDataURL(dataURL)
	if err != nil {
		return nil, err
	}

	originalSize := len(raw)

	if opts.SkipSmall && originalSize < opts.SmallThreshold {
		return &CompressResult{
			DataURL:        dataURL,
			OriginalSize:   originalSize,
			CompressedSize: originalSize,
			Format:         format,
		}, nil
	}

	img, _, err := image.Decode(bytes.NewReader(raw))
	if err != nil {
		return nil, fmt.Errorf("image decode: %w", err)
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	longSide := width
	if height > width {
		longSide = height
	}

	if longSide > opts.MaxLongSide {
		scale := float64(opts.MaxLongSide) / float64(longSide)
		newWidth := int(float64(width) * scale)
		newHeight := int(float64(height) * scale)
		img = scaleImage(img, newWidth, newHeight)
		width = newWidth
		height = newHeight
	}

	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: opts.JPEGQuality}); err != nil {
		return nil, fmt.Errorf("jpeg encode: %w", err)
	}

	compressed := buf.Bytes()
	encoded := base64.StdEncoding.EncodeToString(compressed)
	newDataURL := fmt.Sprintf("data:image/jpeg;base64,%s", encoded)

	return &CompressResult{
		DataURL:        newDataURL,
		OriginalSize:   originalSize,
		CompressedSize: len(compressed),
		Format:         "jpeg",
		Width:          width,
		Height:         height,
	}, nil
}

func IsImage(dataURL string) bool {
	_, _, err := decodeDataURL(dataURL)
	return err == nil
}

func decodeDataURL(dataURL string) ([]byte, string, error) {
	const prefix = "data:"
	if !strings.HasPrefix(dataURL, prefix) {
		return nil, "", fmt.Errorf("not a data URL")
	}

	commaIdx := strings.Index(dataURL, ",")
	if commaIdx < 0 {
		return nil, "", fmt.Errorf("invalid data URL: no comma separator")
	}

	header := dataURL[len(prefix):commaIdx]
	base64Data := dataURL[commaIdx+1:]

	format := "unknown"
	if strings.Contains(header, "image/jpeg") || strings.Contains(header, "image/jpg") {
		format = "jpeg"
	} else if strings.Contains(header, "image/png") {
		format = "png"
	} else if strings.Contains(header, "image/gif") {
		format = "gif"
	} else if strings.Contains(header, "image/webp") {
		format = "webp"
	}

	raw, err := base64.StdEncoding.DecodeString(base64Data)
	if err != nil {
		return nil, "", fmt.Errorf("base64 decode: %w", err)
	}

	return raw, format, nil
}

func scaleImage(src image.Image, width, height int) image.Image {
	dst := image.NewRGBA(image.Rect(0, 0, width, height))
	draw.CatmullRom.Scale(dst, dst.Bounds(), src, src.Bounds(), draw.Over, nil)
	return dst
}
