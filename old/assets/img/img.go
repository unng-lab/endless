package img

import (
	"bytes"
	"embed"
	"image"
	_ "image/jpeg"
	"image/png"
	_ "image/png"
	"log"
	"path/filepath"
	"strconv"

	"github.com/hajimehoshi/ebiten/v2"
	"github.com/nfnt/resize"
)

var (
	//go:embed *
	images embed.FS
)

var storage = make(map[string]*ebiten.Image)

func Img(name string, w, h uint64) (*ebiten.Image, error) {
	fl := fileName(name, w, h)
	if v, ok := storage[fl]; !ok {
		file, err := images.ReadFile(name)
		if err != nil {
			log.Fatal(err)
		}
		var img image.Image
		if filepath.Ext(name) == ".png" {
			img, err = png.Decode(bytes.NewReader(file))
			if err != nil {
				log.Fatal(err)
			}
		} else {
			// decode jpeg into image.Image
			img, _, err = image.Decode(bytes.NewReader(file))
			if err != nil {
				log.Fatal(err)
			}
		}

		m := ebiten.NewImageFromImage(resize.Resize(uint(w), uint(h), img, resize.Lanczos3))
		storage[fl] = m
		return m, nil

	} else {
		return v, nil
	}
}

func fileName(name string, w, h uint64) string {
	return name + "_" + strconv.FormatUint(w, 10) + "_" + strconv.FormatUint(h, 10)
}
