package main

import (
	"fmt"
	"image"
	"io"
	"os"
	"path/filepath"

	_ "image/gif"
	_ "image/jpeg"
	_ "image/png"
)

// These define pixel dimensions for a single sprite frame.
const (
	SpriteWidth  = 8
	SpriteHeight = 8
)

func main() {
	config := parseArgs()
	img := loadImage(config)

	out, close := makeWriter(config)
	defer close()

	translate(out, img)
}

// translate reads sprite data from the given image and writes them to the output.
func translate(out io.Writer, img image.Image) {
	r := img.Bounds()
	w := r.Dx() / SpriteWidth
	h := r.Dy() / SpriteHeight

	fmt.Fprintf(out, "const SpriteCount = %d\n\n:Sprites\n", w*h)

	str := make([]byte, 7, 16)
	copy(str, []byte("d32 $16#"))

	for y := 0; y < h; y++ {
		sy := r.Min.Y + y*SpriteHeight

		for x := 0; x < w; x++ {
			sx := r.Min.X + x*SpriteWidth

			for py := sy; py < sy+SpriteHeight; py++ {
				for px := sx; px < sx+SpriteWidth; px++ {
					pix := img.At(px, py)
					r, _, _, _ := pix.RGBA()

					if r == 0 {
						str = append(str, '0')
					} else {
						str = append(str, '1')
					}
				}

				fmt.Fprintln(out, string(str))
				str = str[:7]
			}

			fmt.Fprintln(out, "")
		}
	}
}

// loadImage loads an image from the input file.
func loadImage(c *Config) image.Image {
	fd, err := os.Open(c.Input)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	defer fd.Close()

	img, _, err := image.Decode(fd)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	r := img.Bounds()
	if r.Dx() < SpriteWidth || r.Dy() < SpriteHeight {
		fmt.Fprintf(os.Stderr, "source image is too small; expected at least %d x %d pixels", SpriteWidth, SpriteHeight)
		os.Exit(1)
	}

	return img
}

// makeWriter creates an output writer and a cleanup function for it.
func makeWriter(c *Config) (io.Writer, func()) {
	if c.Output == "" {
		return os.Stdout, func() {}
	}

	dir, _ := filepath.Split(c.Output)
	err := os.MkdirAll(dir, 0744)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	fd, err := os.Create(c.Output)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	return fd, func() { fd.Close() }
}
