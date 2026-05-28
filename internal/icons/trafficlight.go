// Package icons renders the Statping traffic-light glyph as PNG bytes at any
// size. It is shared by the menu-bar tray (small portrait icons) and the
// notifier (larger square icons for desktop notifications).
package icons

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// TrafficLight renders a traffic-light icon of the given pixel size with the
// lamp matching active ("red", "yellow"/"amber", or "green") lit. Returns PNG bytes.
func TrafficLight(active string, w, h int) []byte {
	if active == "amber" {
		active = "yellow"
	}

	ss := 4
	if w > 64 || h > 64 {
		ss = 2
	}
	sw, sh := w*ss, h*ss
	img := image.NewRGBA(image.Rect(0, 0, sw, sh))

	// Centered portrait housing.
	hh := 0.90 * float64(sh)
	hw := hh * 0.58
	if hw > 0.92*float64(sw) {
		hw = 0.92 * float64(sw)
		hh = hw / 0.58
	}
	hx0 := (float64(sw) - hw) / 2
	hy0 := (float64(sh) - hh) / 2
	rad := hw * 0.28

	fillRoundRect(img, hx0, hy0, hx0+hw, hy0+hh, rad, color.RGBA{0x3a, 0x40, 0x4b, 0xff})

	cx := float64(sw) / 2
	r := hw * 0.26

	type lamp struct {
		name string
		cy   float64
		on   color.RGBA
		off  color.RGBA
	}
	lamps := []lamp{
		{"red", hy0 + hh*0.215, color.RGBA{0xff, 0x45, 0x3a, 0xff}, color.RGBA{0x4a, 0x22, 0x20, 0xff}},
		{"yellow", hy0 + hh*0.5, color.RGBA{0xff, 0xc2, 0x4b, 0xff}, color.RGBA{0x4a, 0x42, 0x18, 0xff}},
		{"green", hy0 + hh*0.785, color.RGBA{0x3f, 0xdc, 0x7f, 0xff}, color.RGBA{0x1c, 0x3c, 0x26, 0xff}},
	}

	for _, l := range lamps {
		if l.name == active {
			glow := l.on
			glow.A = 0x66
			fillCircle(img, cx, l.cy, r*1.42, glow)
			fillCircle(img, cx, l.cy, r, l.on)
			hl := color.RGBA{0xff, 0xff, 0xff, 0x99}
			fillCircle(img, cx-r*0.3, l.cy-r*0.3, r*0.33, hl)
		} else {
			fillCircle(img, cx, l.cy, r, l.off)
		}
	}

	dst := downsample(img, ss)

	var buf bytes.Buffer
	_ = png.Encode(&buf, dst)
	return buf.Bytes()
}

func fillCircle(img *image.RGBA, cx, cy, r float64, c color.RGBA) {
	minX := int(math.Floor(cx - r))
	maxX := int(math.Ceil(cx + r))
	minY := int(math.Floor(cy - r))
	maxY := int(math.Ceil(cy + r))
	r2 := r * r
	for y := minY; y <= maxY; y++ {
		for x := minX; x <= maxX; x++ {
			dx := float64(x) + 0.5 - cx
			dy := float64(y) + 0.5 - cy
			if dx*dx+dy*dy <= r2 {
				blend(img, x, y, c)
			}
		}
	}
}

func fillRoundRect(img *image.RGBA, x0, y0, x1, y1, rad float64, c color.RGBA) {
	for y := int(math.Floor(y0)); y < int(math.Ceil(y1)); y++ {
		for x := int(math.Floor(x0)); x < int(math.Ceil(x1)); x++ {
			fx, fy := float64(x)+0.5, float64(y)+0.5
			if !insideRoundRect(fx, fy, x0, y0, x1, y1, rad) {
				continue
			}
			blend(img, x, y, c)
		}
	}
}

func insideRoundRect(px, py, x0, y0, x1, y1, rad float64) bool {
	if px < x0 || px > x1 || py < y0 || py > y1 {
		return false
	}
	cx, cy := px, py
	switch {
	case px < x0+rad && py < y0+rad:
		cx, cy = x0+rad, y0+rad
	case px > x1-rad && py < y0+rad:
		cx, cy = x1-rad, y0+rad
	case px < x0+rad && py > y1-rad:
		cx, cy = x0+rad, y1-rad
	case px > x1-rad && py > y1-rad:
		cx, cy = x1-rad, y1-rad
	default:
		return true
	}
	dx, dy := px-cx, py-cy
	return dx*dx+dy*dy <= rad*rad
}

func blend(img *image.RGBA, x, y int, c color.RGBA) {
	b := img.Bounds()
	if x < b.Min.X || x >= b.Max.X || y < b.Min.Y || y >= b.Max.Y {
		return
	}
	if c.A == 0xff {
		img.SetRGBA(x, y, c)
		return
	}
	dst := img.RGBAAt(x, y)
	sa := float64(c.A) / 255
	img.SetRGBA(x, y, color.RGBA{
		R: uint8(float64(c.R)*sa + float64(dst.R)*(1-sa)),
		G: uint8(float64(c.G)*sa + float64(dst.G)*(1-sa)),
		B: uint8(float64(c.B)*sa + float64(dst.B)*(1-sa)),
		A: uint8(float64(c.A) + float64(dst.A)*(1-sa)),
	})
}

func downsample(src *image.RGBA, ss int) *image.RGBA {
	sw := src.Bounds().Dx()
	sh := src.Bounds().Dy()
	dw, dh := sw/ss, sh/ss
	dst := image.NewRGBA(image.Rect(0, 0, dw, dh))

	for dy := 0; dy < dh; dy++ {
		for dx := 0; dx < dw; dx++ {
			var sumRA, sumGA, sumBA, sumA float64
			for oy := 0; oy < ss; oy++ {
				for ox := 0; ox < ss; ox++ {
					p := src.RGBAAt(dx*ss+ox, dy*ss+oy)
					a := float64(p.A)
					sumRA += float64(p.R) * a
					sumGA += float64(p.G) * a
					sumBA += float64(p.B) * a
					sumA += a
				}
			}
			n := float64(ss * ss)
			out := color.RGBA{A: uint8(sumA / n)}
			if sumA > 0 {
				out.R = uint8(sumRA / sumA)
				out.G = uint8(sumGA / sumA)
				out.B = uint8(sumBA / sumA)
			}
			dst.SetRGBA(dx, dy, out)
		}
	}
	return dst
}
