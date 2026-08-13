package ogimage

import (
	"bytes"
	_ "embed"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"log/slog"
	"strings"

	"github.com/yaps-sh/yaps/internal/paste"
	"golang.org/x/image/font"
	"golang.org/x/image/font/opentype"
	"golang.org/x/image/math/fixed"
)

//go:embed assets/IBMPlexMono-Regular.ttf
var plexTTF []byte

const (
	imgWidth  = 1200
	imgHeight = 630

	fontSize       = 22
	codeLineHeight = 30
	codeStartY     = 100
	codePadX       = 32
	topBarH        = 56
	accentH        = 2
	maxLines       = 16
	maxLineRunes   = 92
)

var (
	colBg       = color.RGBA{R: 0x0d, G: 0x0f, B: 0x12, A: 0xff}
	colBgRaised = color.RGBA{R: 0x14, G: 0x17, B: 0x1c, A: 0xff}
	colAccent   = color.RGBA{R: 0xf0, G: 0xa8, B: 0x3c, A: 0xff}
	colFg       = color.RGBA{R: 0xe8, G: 0xe6, B: 0xe3, A: 0xff}
	colFgMuted  = color.RGBA{R: 0x8b, G: 0x8f, B: 0x96, A: 0xff}
	colFgFaint  = color.RGBA{R: 0x56, G: 0x5a, B: 0x61, A: 0xff}
)

var regularFace font.Face

func init() {
	slog.Debug("ogimage: loaded IBM Plex Mono Regular", "size_bytes", len(plexTTF))

	ttf, err := opentype.Parse(plexTTF)
	if err != nil {
		panic(fmt.Sprintf("ogimage: parse IBM Plex Mono: %v", err))
	}
	face, err := opentype.NewFace(
		ttf, &opentype.FaceOptions{
			Size:    fontSize,
			DPI:     72,
			Hinting: font.HintingFull,
		},
	)
	if err != nil {
		panic(fmt.Sprintf("ogimage: new face: %v", err))
	}
	regularFace = face
}

func Generate(entry *paste.Entry) ([]byte, error) {
	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))
	draw.Draw(img, img.Bounds(), image.NewUniform(colBg), image.Point{}, draw.Src)

	draw.Draw(
		img, image.Rect(0, 0, imgWidth, topBarH),
		image.NewUniform(colBgRaised), image.Point{}, draw.Src,
	)
	draw.Draw(
		img, image.Rect(0, topBarH, imgWidth, topBarH+accentH),
		image.NewUniform(colAccent), image.Point{}, draw.Src,
	)

	drawString(img, "yaps.sh", codePadX, 36, colFg)

	lang := entry.DetectedLanguage
	if lang == "" {
		lang = "plaintext"
	}
	langW := measureString(lang)
	badgeX := imgWidth - codePadX - langW - 12
	badgeRect := image.Rect(badgeX-12, 18, imgWidth-codePadX, 44)
	draw.Draw(img, badgeRect, image.NewUniform(colBg), image.Point{}, draw.Src)
	drawString(img, lang, badgeX, 36, colFgMuted)

	lines := buildSnippet(entry.Content)
	bottom := imgHeight - 30
	for i, ln := range lines {
		y := codeStartY + i*codeLineHeight
		if y > bottom {
			break
		}
		col := colFg
		if ln == "…" {
			col = colFgFaint
		}
		drawString(img, ln, codePadX, y, col)
	}

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("ogimage: encode png: %w", err)
	}
	return buf.Bytes(), nil
}

func drawString(dst *image.RGBA, s string, x, y int, col color.Color) {
	d := &font.Drawer{
		Dst:  dst,
		Src:  image.NewUniform(col),
		Face: regularFace,
		Dot:  fixed.P(x, y),
	}
	d.DrawString(s)
}

func measureString(s string) int {
	d := &font.Drawer{Face: regularFace}
	return d.MeasureString(s).Round()
}

func buildSnippet(content string) []string {
	content = strings.TrimSpace(content)
	if content == "" {
		return []string{"(empty paste)"}
	}
	raw := strings.Split(content, "\n")
	var out []string
	for _, ln := range raw {
		if len(out) >= maxLines {
			out[maxLines-1] = "…"
			return out
		}
		r := []rune(ln)
		if len(r) > maxLineRunes {
			ln = string(r[:maxLineRunes-1]) + "…"
		}
		out = append(out, ln)
	}
	return out
}
