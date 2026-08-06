package imageproc

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"testing"
)

func makeTestImage(width, height int) string {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: uint8(x % 256), G: uint8(y % 256), B: 128, A: 255})
		}
	}
	var buf bytes.Buffer
	jpeg.Encode(&buf, img, &jpeg.Options{Quality: 95})
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return fmt.Sprintf("data:image/jpeg;base64,%s", encoded)
}

func TestCompress_SmallImage_Skip(t *testing.T) {
	dataURL := makeTestImage(100, 100)
	result, err := Compress(dataURL, nil)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	if result.CompressedSize != result.OriginalSize {
		t.Errorf("small image should be skipped, got %d vs %d", result.CompressedSize, result.OriginalSize)
	}
}

func TestCompress_LargeImage_Scale(t *testing.T) {
	dataURL := makeTestImage(3000, 2000)
	opts := &CompressOptions{
		MaxLongSide:    2048,
		JPEGQuality:    85,
		SkipSmall:      false,
		SmallThreshold: 500 * 1024,
	}
	result, err := Compress(dataURL, opts)
	if err != nil {
		t.Fatalf("Compress failed: %v", err)
	}
	if result.Width > 2048 && result.Height > 2048 {
		t.Errorf("long side should be <= 2048, got %dx%d", result.Width, result.Height)
	}
	if result.Format != "jpeg" {
		t.Errorf("expected jpeg format, got %s", result.Format)
	}
}

func TestIsImage_Valid(t *testing.T) {
	dataURL := makeTestImage(10, 10)
	if !IsImage(dataURL) {
		t.Error("IsImage should return true for valid image data URL")
	}
}

func TestIsImage_Invalid(t *testing.T) {
	if IsImage("not a data url") {
		t.Error("IsImage should return false for invalid input")
	}
	if IsImage("data:image/jpeg;base64,!!!not-base64") {
		t.Error("IsImage should return false for invalid base64")
	}
}

func TestCompress_InvalidDataURL(t *testing.T) {
	_, err := Compress("not a data url", nil)
	if err == nil {
		t.Error("Compress should fail for invalid data URL")
	}
}

func TestCompress_InvalidBase64(t *testing.T) {
	_, err := Compress("data:image/jpeg;base64,!!!invalid", nil)
	if err == nil {
		t.Error("Compress should fail for invalid base64")
	}
}

func TestCompress_NilOptions(t *testing.T) {
	dataURL := makeTestImage(200, 200)
	result, err := Compress(dataURL, nil)
	if err != nil {
		t.Fatalf("Compress with nil options failed: %v", err)
	}
	if result.DataURL == "" {
		t.Error("DataURL should not be empty")
	}
}
