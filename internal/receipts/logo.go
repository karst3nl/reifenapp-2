package receipts

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"image"
	"image/png"
	"strings"
)

const MaxLogoBytes = 2 * 1024 * 1024
const MaxTemplateLogoBytes = 4 * 1024 * 1024

// Logos are normalized in the browser. Never resolve file paths, remote URLs or SVG here.
func decodeLogo(value string) ([]byte, image.Config, error) {
	const prefix = "data:image/png;base64,"
	invalid := fmt.Errorf("Logo muss ein gültiges PNG sein (maximal 2 MiB, 2048 × 2048 Pixel). Bitte erneut importieren.")
	if !strings.HasPrefix(value, prefix) || len(value) > len(prefix)+base64.StdEncoding.EncodedLen(MaxLogoBytes) {
		return nil, image.Config{}, invalid
	}
	data, err := base64.StdEncoding.Strict().DecodeString(strings.TrimPrefix(value, prefix))
	if err != nil || len(data) > MaxLogoBytes {
		return nil, image.Config{}, invalid
	}
	cfg, err := png.DecodeConfig(bytes.NewReader(data))
	if err != nil || cfg.Width < 1 || cfg.Height < 1 || cfg.Width > 2048 || cfg.Height > 2048 {
		return nil, image.Config{}, invalid
	}
	if _, err = png.Decode(bytes.NewReader(data)); err != nil {
		return nil, image.Config{}, invalid
	}
	return data, cfg, nil
}
