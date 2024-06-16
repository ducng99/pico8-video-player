package colour

import "math"

type Colour struct {
	R, G, B uint32
	H, S, L float64
}

var (
	P8_COLOUR_0  = Colour{0, 0, 0, 0, 0, 0}
	P8_COLOUR_1  = Colour{29, 43, 83, 150, 116, 53}
	P8_COLOUR_2  = Colour{126, 37, 83, 219, 131, 77}
	P8_COLOUR_3  = Colour{0, 135, 81, 104, 240, 64}
	P8_COLOUR_4  = Colour{171, 82, 54, 10, 125, 106}
	P8_COLOUR_5  = Colour{95, 87, 79, 20, 22, 82}
	P8_COLOUR_6  = Colour{194, 195, 199, 152, 10, 185}
	P8_COLOUR_7  = Colour{255, 241, 232, 16, 240, 229}
	P8_COLOUR_8  = Colour{255, 0, 77, 228, 240, 120}
	P8_COLOUR_9  = Colour{255, 163, 0, 26, 240, 120}
	P8_COLOUR_10 = Colour{255, 236, 39, 36, 240, 138}
	P8_COLOUR_11 = Colour{0, 228, 54, 89, 240, 107}
	P8_COLOUR_12 = Colour{41, 173, 255, 135, 240, 139}
	P8_COLOUR_13 = Colour{131, 118, 156, 174, 39, 129}
	P8_COLOUR_14 = Colour{255, 119, 168, 226, 240, 136}
	P8_COLOUR_15 = Colour{255, 204, 170, 16, 240, 200}
)

func NewWithRGB(r, g, b uint32) Colour {
	// Normalize the RGB values to the range [0, 1]
	rf := float64(r) / 255.0
	gf := float64(g) / 255.0
	bf := float64(b) / 255.0

	// Find the minimum and maximum values
	maxVal := math.Max(math.Max(rf, gf), bf)
	minVal := math.Min(math.Min(rf, gf), bf)

	// Calculate Lightness
	l := (maxVal + minVal) / 2

	var s, h float64

	if maxVal == minVal {
		// Achromatic case
		s = 0
		h = 0
	} else {
		// Chromatic case
		delta := maxVal - minVal

		// Calculate Saturation
		if l > 0.5 {
			s = delta / (2.0 - maxVal - minVal)
		} else {
			s = delta / (maxVal + minVal)
		}

		// Calculate Hue
		switch maxVal {
		case rf:
			h = (gf - bf) / delta
			if gf < bf {
				h += 6
			}
		case gf:
			h = (bf-rf)/delta + 2
		case bf:
			h = (rf-gf)/delta + 4
		}
		h /= 6
	}

	h *= 360 // Convert hue to degrees
	return Colour{r, g, b, h, s, l}
}

// Calculate the distance between two colours in HSL space
// using Euclidean distance.
//
// Returns a float64 representing the distance between the two colours,
// lower values indicate closer colours.
func (c Colour) HslDistance(target Colour) float64 {
	// Calculate the differences
	dh := math.Abs(c.H - target.H)
	if dh > 180 {
		dh = 360 - dh
	}
	ds := c.S - target.S
	dl := c.L - target.L

	// Compute the Euclidean distance
	distance := math.Sqrt(dh*dh + ds*ds + dl*dl)
	return distance
}

// Compress RGB to a PICO-8 4-bit colour.
// This function converts the RGB values to HSL and then compares them to the PICO-8 colours.
func CompressRGBToP8Colour(r, g, b uint32) byte {
	// Get the closest colour using delta-e
	closestColour := P8_COLOUR_0
	smallestDeltaE := math.MaxFloat64
	for _, p8Colour := range []Colour{
		P8_COLOUR_0,
		P8_COLOUR_1,
		P8_COLOUR_2,
		P8_COLOUR_3,
		P8_COLOUR_4,
		P8_COLOUR_5,
		P8_COLOUR_6,
		P8_COLOUR_7,
		P8_COLOUR_8,
		P8_COLOUR_9,
		P8_COLOUR_10,
		P8_COLOUR_11,
		P8_COLOUR_12,
		P8_COLOUR_13,
		P8_COLOUR_14,
		P8_COLOUR_15,
	} {
		// Convert current RGB to HSL
		currentColour := NewWithRGB(r, g, b)
		// Compare colours using HSL distance
		deltaE := p8Colour.HslDistance(currentColour)
		if deltaE < smallestDeltaE {
			smallestDeltaE = deltaE
			closestColour = p8Colour
		}
	}

	// Map the closest colour to the PICO-8 colour
	switch closestColour {
	case P8_COLOUR_0:
		return 0
	case P8_COLOUR_1:
		return 1
	case P8_COLOUR_2:
		return 2
	case P8_COLOUR_3:
		return 3
	case P8_COLOUR_4:
		return 4
	case P8_COLOUR_5:
		return 5
	case P8_COLOUR_6:
		return 6
	case P8_COLOUR_7:
		return 7
	case P8_COLOUR_8:
		return 8
	case P8_COLOUR_9:
		return 9
	case P8_COLOUR_10:
		return 10
	case P8_COLOUR_11:
		return 11
	case P8_COLOUR_12:
		return 12
	case P8_COLOUR_13:
		return 13
	case P8_COLOUR_14:
		return 14
	case P8_COLOUR_15:
		return 15
	default:
		return 0
	}
}
