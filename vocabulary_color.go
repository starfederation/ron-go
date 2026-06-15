package ron

import colorlib "github.com/SCKelemen/color"

const (
	// VocabularyColorV1 is the RON color typed vocabulary URI.
	VocabularyColorV1 = "https://ron.dev/vocab/color/v1"
)

// ColorSpace names a color vocabulary color space.
type ColorSpace string

const (
	// ColorSpaceOKLCH is the canonical color vocabulary color space.
	ColorSpaceOKLCH ColorSpace = "oklch"
)

// Color is a color vocabulary #clr value.
type Color struct {
	Space    ColorSpace
	Channels [3]float64
	Value    colorlib.Color
}

func NewOKLCHColor(lightness, chroma, hue float64) Color {
	return Color{
		Space: ColorSpaceOKLCH,
		Channels: [3]float64{
			lightness,
			chroma,
			hue,
		},
		Value: colorlib.NewOKLCH(lightness, chroma, hue, 1),
	}
}

func (c Color) OKLCH() *colorlib.OKLCH {
	if c.Value != nil {
		return colorlib.ToOKLCH(c.Value)
	}

	return colorlib.NewOKLCH(c.Channels[0], c.Channels[1], c.Channels[2], 1)
}

func (opts optionState) isColorTag(tag string) bool {
	if _, ok := opts.vocabularies[VocabularyColorV1]; !ok {
		return false
	}
	switch tag {
	case "#clr":
		return true
	default:
		return false
	}
}

func (opts optionState) parseColorPayload(tag string, payload any) (any, error) {
	if tag != "#clr" {
		return nil, newError("unsupported color tag")
	}
	values, ok := payload.([]any)
	if !ok || len(values) != 4 {
		return nil, newError("invalid #clr payload")
	}
	space, ok := values[0].(string)
	if !ok || ColorSpace(space) != ColorSpaceOKLCH {
		return nil, newError("invalid #clr payload")
	}
	var channels [3]float64
	for i := range channels {
		value, ok := numberAsFloat64(values[i+1])
		if !ok {
			return nil, newError("invalid #clr payload")
		}
		channels[i] = value
	}

	return Color{
		Space:    ColorSpace(space),
		Channels: channels,
		Value:    colorlib.NewOKLCH(channels[0], channels[1], channels[2], 1),
	}, nil
}

func colorTaggedMember(value any) (objectMember, bool) {
	switch value := value.(type) {
	case Color:
		return colorMember(value), true
	case *colorlib.OKLCH:
		if value == nil {
			return objectMember{}, false
		}
		return colorMember(Color{
			Space: ColorSpaceOKLCH,
			Channels: [3]float64{
				value.L,
				value.C,
				value.H,
			},
			Value: value,
		}), true
	case colorlib.Color:
		if value == nil {
			return objectMember{}, false
		}
		oklch := colorlib.ToOKLCH(value)
		return colorMember(Color{
			Space: ColorSpaceOKLCH,
			Channels: [3]float64{
				oklch.L,
				oklch.C,
				oklch.H,
			},
			Value: value,
		}), true
	default:
		return objectMember{}, false
	}
}

func colorMember(value Color) objectMember {
	return objectMember{
		Key: "#clr",
		Value: []any{
			string(value.Space),
			value.Channels[0],
			value.Channels[1],
			value.Channels[2],
		},
	}
}
