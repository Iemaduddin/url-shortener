package qr

import (
	"bytes"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"testing"
)

func TestNormalizeConfigDefaults(t *testing.T) {
	got := normalizeConfig(Config{})
	if got.Foreground != defaultForeground {
		t.Fatalf("foreground = %q, want %q", got.Foreground, defaultForeground)
	}
	if got.Background != defaultBackground {
		t.Fatalf("background = %q, want %q", got.Background, defaultBackground)
	}
	if got.FrameText != defaultFrameText {
		t.Fatalf("frame text = %q, want %q", got.FrameText, defaultFrameText)
	}
}

func TestParseColor(t *testing.T) {
	got := parseColor("#ff0000")
	want := color.RGBA{R: 255, A: 255}
	if got != want {
		t.Fatalf("parseColor = %+v, want %+v", got, want)
	}
	if got := parseColor("not-a-color"); got != (color.RGBA{A: 255}) {
		t.Fatalf("invalid color = %+v, want black", got)
	}
}

func TestRenderAddsFrameText(t *testing.T) {
	data, config, err := render("https://example.com", Config{})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if config.FrameText != defaultFrameText {
		t.Fatalf("frame text = %q, want %q", config.FrameText, defaultFrameText)
	}
	img, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatalf("decode: %v", err)
	}
	if img.Bounds().Dx() != 512 {
		t.Fatalf("width = %d, want 512", img.Bounds().Dx())
	}
	if img.Bounds().Dy() <= 512 {
		t.Fatalf("height = %d, want frame text band added", img.Bounds().Dy())
	}
}

func TestRenderAcceptsLogo(t *testing.T) {
	logo := image.NewRGBA(image.Rect(0, 0, 40, 40))
	draw.Draw(logo, logo.Bounds(), image.NewUniform(color.RGBA{R: 255, A: 255}), image.Point{}, draw.Src)
	var logoPNG bytes.Buffer
	if err := png.Encode(&logoPNG, logo); err != nil {
		t.Fatalf("encode logo: %v", err)
	}

	data, config, err := render("https://example.com", Config{Logo: logoPNG.Bytes()})
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if len(config.Logo) == 0 {
		t.Fatal("normalized logo is empty")
	}
	if _, err := png.Decode(bytes.NewReader(data)); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestNormalizeLogoRejectsInvalidData(t *testing.T) {
	if _, err := normalizeLogo([]byte("not an image")); err == nil {
		t.Fatal("expected invalid logo to fail")
	}
}
