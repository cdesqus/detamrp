package settings

import (
	"bytes"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"

	_ "golang.org/x/image/webp"
)

func validateBrandingImage(content []byte, _ string, maxBytes int) (string, error) {
	if len(content) == 0 || len(content) > maxBytes {
		return "", errors.New("image size is invalid")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(content))
	if err != nil {
		return "", errors.New("image content is invalid")
	}
	if config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > 40_000_000 {
		return "", errors.New("image dimensions are invalid")
	}
	if _, _, err := image.Decode(bytes.NewReader(content)); err != nil {
		return "", errors.New("image content is invalid")
	}
	switch format {
	case "png":
		return "image/png", nil
	case "jpeg":
		return "image/jpeg", nil
	case "webp":
		return "image/webp", nil
	default:
		return "", errors.New("image type is not supported")
	}
}

func mediaURL(kind string) *string {
	value := "/public/branding/" + kind
	return &value
}
