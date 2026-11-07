// Command icon renders the application icon PNGs used for the Windows resource.
package main

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"path/filepath"

	"outpost/internal/art"
)

func main() {
	src := art.IconArt()
	side := max(src.W, src.H) + 2
	c := art.NewCanvas(side, side)
	c.Stamp(src, (side-src.W)/2, (side-src.H)/2)
	fmt.Println("icon art", src.W, "x", src.H)
	for _, s := range []int{256, 128, 48, 32, 16} {
		img := image.NewRGBA(image.Rect(0, 0, s, s))
		if k := s / side; k >= 1 {
			// Crisp integer upscale, centred.
			off := (s - side*k) / 2
			for y := 0; y < side*k; y++ {
				for x := 0; x < side*k; x++ {
					img.Set(off+x, off+y, c.Img.At(x/k, y/k))
				}
			}
		} else {
			// Area-average downscale (premultiplied) for the tiny sizes.
			f := float64(side) / float64(s)
			for y := 0; y < s; y++ {
				for x := 0; x < s; x++ {
					var r, g, b, a, n float64
					for sy := int(float64(y) * f); sy < int(float64(y+1)*f+0.999) && sy < side; sy++ {
						for sx := int(float64(x) * f); sx < int(float64(x+1)*f+0.999) && sx < side; sx++ {
							p := c.Get(sx, sy)
							al := float64(p.A) / 255
							r += float64(p.R) * al
							g += float64(p.G) * al
							b += float64(p.B) * al
							a += al
							n++
						}
					}
					if a > 0 {
						img.Set(x, y, color.RGBA{uint8(r / a), uint8(g / a), uint8(b / a), uint8(255 * a / n)})
					}
				}
			}
		}
		f, err := os.Create(filepath.Join(os.Args[1], fmt.Sprintf("icon%d.png", s)))
		if err != nil {
			panic(err)
		}
		if err := png.Encode(f, img); err != nil {
			panic(err)
		}
		f.Close()
	}
}
