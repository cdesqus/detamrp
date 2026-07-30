package settings

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func testPNG(t *testing.T) []byte {
	t.Helper()
	var out bytes.Buffer
	img := image.NewRGBA(image.Rect(0, 0, 2, 2))
	img.Set(0, 0, color.Black)
	if err := png.Encode(&out, img); err != nil {
		t.Fatal(err)
	}
	return out.Bytes()
}

func TestValidateBrandingImageDetectsDecodedType(t *testing.T) {
	mime, err := validateBrandingImage(testPNG(t), "image/jpeg", logoMaxBytes)
	if err != nil || mime != "image/png" {
		t.Fatalf("mime=%q err=%v", mime, err)
	}
}

func TestValidateBrandingImageRejectsInvalidAndOversizedContent(t *testing.T) {
	if _, err := validateBrandingImage([]byte("not an image"), "image/png", logoMaxBytes); err == nil {
		t.Fatal("invalid image accepted")
	}
	if _, err := validateBrandingImage(make([]byte, logoMaxBytes+1), "image/png", logoMaxBytes); err == nil {
		t.Fatal("oversized image accepted")
	}
}
