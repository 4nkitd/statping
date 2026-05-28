package tray

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"math"
)

// Traffic-light menu bar icons. A vertical housing with three lamps; the lamp
// matching the current state is lit (the others are dimmed). Generated at
// startup so we don't carry hardcoded PNG blobs.
var (
	greenIcon  []byte
	yellowIcon []byte
	redIcon    []byte
)

func init() {
	greenIcon = makeTrafficLight("green")
	yellowIcon = makeTrafficLight("yellow")
	redIcon = makeTrafficLight("red")
}

// icon geometry (in points; supersampled when rendered)
const (
	iconW  = 16
	iconH  = 22
	iconSS = 4 // supersample factor for anti-aliasing
)

func makeTrafficLight(active string) []byte {
	w, h := iconW*iconSS, iconH*iconSS
	img := image.NewRGBA(image.Rect(0, 0, w, h))

	housing := color.RGBA{0x3a, 0x40, 0x4b, 0xff}
	fillRoundRect(img, 2.5*iconSS, 1*iconSS, 13.5*iconSS, 21*iconSS, 4*iconSS, housing)

	type lamp struct {
		name string
		cy   float64
		on   color.RGBA
		off  color.RGBA
	}
	lamps := []lamp{
		{"red", 6, color.RGBA{0xff, 0x45, 0x3a, 0xff}, color.RGBA{0x4a, 0x22, 0x20, 0xff}},
		{"yellow", 11, color.RGBA{0xff, 0xcc, 0x00, 0xff}, color.RGBA{0x4a, 0x42, 0x18, 0xff}},
		{"green", 16, color.RGBA{0x34, 0xc7, 0x59, 0xff}, color.RGBA{0x1c, 0x3c, 0x26, 0xff}},
	}

	cx := 8.0 * iconSS
	r := 2.7 * iconSS
	for _, l := range lamps {
		cy := l.cy * iconSS
		if l.name == active {
			// soft glow behind the lit lamp
			glow := l.on
			glow.A = 0x66
			fillCircle(img, cx, cy, r*1.45, glow)
			fillCircle(img, cx, cy, r, l.on)
			// small highlight
			hl := color.RGBA{0xff, 0xff, 0xff, 0x99}
			fillCircle(img, cx-r*0.3, cy-r*0.3, r*0.35, hl)
		} else {
			fillCircle(img, cx, cy, r, l.off)
		}
	}

	dst := downsample(img, iconSS)

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
	// corner regions
	cx := px
	cy := py
	if px < x0+rad && py < y0+rad {
		cx, cy = x0+rad, y0+rad
	} else if px > x1-rad && py < y0+rad {
		cx, cy = x1-rad, y0+rad
	} else if px < x0+rad && py > y1-rad {
		cx, cy = x0+rad, y1-rad
	} else if px > x1-rad && py > y1-rad {
		cx, cy = x1-rad, y1-rad
	} else {
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
	// simple source-over blend
	dst := img.RGBAAt(x, y)
	sa := float64(c.A) / 255
	img.SetRGBA(x, y, color.RGBA{
		R: uint8(float64(c.R)*sa + float64(dst.R)*(1-sa)),
		G: uint8(float64(c.G)*sa + float64(dst.G)*(1-sa)),
		B: uint8(float64(c.B)*sa + float64(dst.B)*(1-sa)),
		A: uint8(float64(c.A) + float64(dst.A)*(1-sa)),
	})
}

// downsample averages ss*ss blocks (premultiplied) for anti-aliasing.
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
			outA := sumA / n
			var out color.RGBA
			out.A = uint8(outA)
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
