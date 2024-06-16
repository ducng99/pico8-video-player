package image

import (
	jpeg "image/jpeg"
	"os"

	"r.tomng.dev/video2p8/colour"
)

func GetP8Colours(jpg_file string) ([]byte, error) {
	f, err := os.Open(jpg_file)
	if err != nil {
		return nil, err
	}

	image, err := jpeg.Decode(f)
	if err != nil {
		return nil, err
	}

	bounds := image.Bounds()
	width, height := bounds.Max.X, bounds.Max.Y

	pixels := make([]byte, width*height)
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			r, g, b, _ := image.At(x, y).RGBA()
			colour := colour.CompressRGBToP8Colour(r, g, b)
			pixels[y*width+x] = colour
		}
	}

	return pixels, nil
}
