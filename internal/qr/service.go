package qr

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/jpeg"
	"image/png"
	"strings"
	"sync"

	qrcode "github.com/skip2/go-qrcode"
	"golang.org/x/image/font"
	"golang.org/x/image/font/gofont/goregular"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
	_ "golang.org/x/image/webp"
)

const (
	defaultForeground = "#000000"
	defaultBackground = "#FFFFFF"
	defaultFrameText  = "Scan untuk membuka URL"
)

type Service struct {
	repository *Repository
}

func NewService(repository *Repository) *Service { return &Service{repository: repository} }
func (s *Service) Get(ctx context.Context, id string) ([]byte, error) {
	return s.repository.Get(ctx, id)
}

func (s *Service) GetConfig(ctx context.Context, id string) (SavedConfig, error) {
	return s.repository.GetConfig(ctx, id)
}

func (s *Service) GetLogo(ctx context.Context, id string) ([]byte, error) {
	return s.repository.GetLogo(ctx, id)
}

func (s *Service) Preview(value string, config Config) ([]byte, error) {
	pngData, _, err := render(value, config)
	return pngData, err
}

func (s *Service) Generate(ctx context.Context, id, value string, config Config) ([]byte, error) {
	pngData, normalized, err := render(value, config)
	if err != nil {
		return nil, err
	}
	if err := s.repository.Save(ctx, id, pngData, normalized); err != nil {
		return nil, err
	}
	return pngData, nil
}

func render(value string, config Config) ([]byte, Config, error) {
	config = normalizeConfig(config)
	if len(config.Logo) > 5*1024*1024 {
		return nil, config, fmt.Errorf("logo exceeds 5 MB")
	}
	if len(config.Logo) > 0 {
		var err error
		config.Logo, err = normalizeLogo(config.Logo)
		if err != nil {
			return nil, config, err
		}
	}

	level := qrcode.Medium
	if config.Logo != nil {
		level = qrcode.High
	}
	code, err := qrcode.New(value, level)
	if err != nil {
		return nil, config, fmt.Errorf("create qr: %w", err)
	}

	code.ForegroundColor = parseColor(config.Foreground)
	code.BackgroundColor = parseColor(config.Background)
	pngData, err := code.PNG(512)
	if err != nil {
		return nil, config, fmt.Errorf("encode qr: %w", err)
	}
	if config.Logo != nil {
		pngData, err = overlayLogo(pngData, config.Logo)
		if err != nil {
			return nil, config, err
		}
	}
	pngData, err = addFrameText(pngData, config.FrameText, parseColor(config.Background), parseColor(config.Foreground))
	if err != nil {
		return nil, config, err
	}
	return pngData, config, nil
}

func normalizeLogo(source []byte) ([]byte, error) {
	decoded, format, err := image.Decode(bytes.NewReader(source))
	if err != nil || (format != "png" && format != "jpeg" && format != "webp") {
		return nil, fmt.Errorf("logo must be PNG, JPEG, or WebP")
	}
	bounds := decoded.Bounds()
	if bounds.Dx() > 512 || bounds.Dy() > 512 {
		decoded = resizeImage(decoded, 512, 512)
	}
	var output bytes.Buffer
	switch format {
	case "jpeg":
		err = jpeg.Encode(&output, decoded, &jpeg.Options{Quality: 82})
	default:
		err = png.Encode(&output, decoded)
	}
	if err != nil || output.Len() > 256*1024 {
		return nil, fmt.Errorf("logo exceeds 256 KB after processing")
	}
	return output.Bytes(), nil
}

func normalizeConfig(config Config) Config {
	if config.Foreground == "" {
		config.Foreground = defaultForeground
	}
	if config.Background == "" {
		config.Background = defaultBackground
	}
	if config.FrameText == "" {
		config.FrameText = defaultFrameText
	}
	return config
}

var (
	frameFontOnce sync.Once
	frameFont     *opentype.Font
	frameFontErr  error
)

func loadFrameFont() (*opentype.Font, error) {
	frameFontOnce.Do(func() {
		frameFont, frameFontErr = opentype.Parse(goregular.TTF)
	})
	return frameFont, frameFontErr
}

func addFrameText(source []byte, text string, background, foreground color.RGBA) ([]byte, error) {
	base, err := png.Decode(bytes.NewReader(source))
	if err != nil {
		return nil, err
	}
	bounds := base.Bounds()
	const bandHeight = 64
	canvas := image.NewRGBA(image.Rect(0, 0, bounds.Dx(), bounds.Dy()+bandHeight))
	draw.Draw(canvas, canvas.Bounds(), image.NewUniform(background), image.Point{}, draw.Src)
	draw.Draw(canvas, bounds, base, bounds.Min, draw.Src)
	band := image.Rect(0, bounds.Dy(), bounds.Dx(), bounds.Dy()+bandHeight)
	if err := drawFrameText(canvas, text, foreground, band); err != nil {
		return nil, err
	}
	var output bytes.Buffer
	if err := png.Encode(&output, canvas); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func drawFrameText(dst *image.RGBA, text string, foreground color.RGBA, band image.Rectangle) error {
	parsed, err := loadFrameFont()
	if err != nil {
		return err
	}
	face, err := opentype.NewFace(parsed, &opentype.FaceOptions{
		Size:    20,
		DPI:     72,
		Hinting: font.HintingFull,
	})
	if err != nil {
		return err
	}
	defer face.Close()

	label := truncateToWidth(face, text, band.Dx()-32)
	width := font.MeasureString(face, label).Ceil()
	metrics := face.Metrics()
	x := band.Min.X + (band.Dx()-width)/2
	y := band.Min.Y + (band.Dy()-metrics.Height.Ceil())/2 + metrics.Ascent.Ceil()
	drawer := font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(foreground),
		Face: face,
		Dot:  fixed.P(x, y),
	}
	drawer.DrawString(label)
	return nil
}

func truncateToWidth(face font.Face, text string, maxWidth int) string {
	if font.MeasureString(face, text).Ceil() <= maxWidth {
		return text
	}
	runes := []rune(text)
	const ellipsis = "…"
	for i := len(runes); i > 0; i-- {
		candidate := string(runes[:i]) + ellipsis
		if font.MeasureString(face, candidate).Ceil() <= maxWidth {
			return candidate
		}
	}
	return ellipsis
}

func parseColor(value string) color.RGBA {
	value = strings.TrimPrefix(value, "#")
	if len(value) != 6 {
		return color.RGBA{A: 255}
	}
	var result color.RGBA
	fmt.Sscanf(value, "%02x%02x%02x", &result.R, &result.G, &result.B)
	result.A = 255
	return result
}

func overlayLogo(source, logo []byte) ([]byte, error) {
	base, err := png.Decode(bytes.NewReader(source))
	if err != nil {
		return nil, err
	}
	overlay, _, err := image.Decode(bytes.NewReader(logo))
	if err != nil {
		return nil, err
	}
	bounds := base.Bounds()
	target := image.NewRGBA(bounds)
	draw.Draw(target, bounds, base, image.Point{}, draw.Src)
	size := bounds.Dx() / 5
	resized := resizeImage(overlay, size, size)
	offset := image.Pt((bounds.Dx()-size)/2, (bounds.Dy()-size)/2)
	padding := size / 6
	paddingRect := image.Rect(offset.X-padding, offset.Y-padding, offset.X+size+padding, offset.Y+size+padding)
	draw.Draw(target, paddingRect, image.NewUniform(color.White), image.Point{}, draw.Src)
	draw.Draw(target, image.Rect(offset.X, offset.Y, offset.X+size, offset.Y+size), resized, image.Point{}, draw.Over)
	var output bytes.Buffer
	if err := png.Encode(&output, target); err != nil {
		return nil, err
	}
	return output.Bytes(), nil
}

func resizeImage(source image.Image, width, height int) image.Image {
	target := image.NewRGBA(image.Rect(0, 0, width, height))
	sourceBounds := source.Bounds()
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			sourceX := sourceBounds.Min.X + x*sourceBounds.Dx()/width
			sourceY := sourceBounds.Min.Y + y*sourceBounds.Dy()/height
			target.Set(x, y, source.At(sourceX, sourceY))
		}
	}
	return target
}
