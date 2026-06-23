package ron

import colorlib "github.com/SCKelemen/color"

const (
	// VocabularyColorV1 is the RON color typed vocabulary URI.
	VocabularyColorV1 = "https://ron.dev/vocab/color/v1"
)

// ColorSpace names a color vocabulary color space.
type ColorSpace string

const (
	ColorSpaceRGB    ColorSpace = "rgb"
	ColorSpaceRGBA   ColorSpace = "rgba"
	ColorSpaceHSL    ColorSpace = "hsl"
	ColorSpaceHSLA   ColorSpace = "hsla"
	ColorSpaceHSV    ColorSpace = "hsv"
	ColorSpaceHSVA   ColorSpace = "hsva"
	ColorSpaceHWB    ColorSpace = "hwb"
	ColorSpaceHWBA   ColorSpace = "hwba"
	ColorSpaceLAB    ColorSpace = "lab"
	ColorSpaceLABA   ColorSpace = "laba"
	ColorSpaceLCH    ColorSpace = "lch"
	ColorSpaceLCHA   ColorSpace = "lcha"
	ColorSpaceOKLAB  ColorSpace = "oklab"
	ColorSpaceOKLABA ColorSpace = "oklaba"
	// ColorSpaceOKLCH is the canonical color vocabulary color space.
	ColorSpaceOKLCH  ColorSpace = "oklch"
	ColorSpaceOKLCHA ColorSpace = "oklcha"
	ColorSpaceXYZ    ColorSpace = "xyz"
	ColorSpaceXYZA   ColorSpace = "xyza"
)

// Color is a color vocabulary #clr value.
type Color struct {
	Space    ColorSpace
	Channels []float64
	Value    colorlib.Color
}

func NewOKLCHColor(lightness, chroma, hue float64) Color {
	return Color{
		Space: ColorSpaceOKLCH,
		Channels: []float64{
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
	if len(c.Channels) < 3 {
		return colorlib.NewOKLCH(0, 0, 0, 1)
	}
	alpha := 1.0
	if len(c.Channels) == 4 {
		alpha = c.Channels[3]
	}
	return colorlib.NewOKLCH(c.Channels[0], c.Channels[1], c.Channels[2], alpha)
}

func (opts optionState) isColorTag(tag string) bool {
	if !opts.vocabularyEnabled(vocabularyColor, VocabularyColorV1) {
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
	if !ok || len(values) < 1 {
		return nil, newError("invalid #clr payload")
	}
	spaceName, ok := values[0].(string)
	if !ok {
		return nil, newError("invalid #clr payload")
	}
	space := ColorSpace(spaceName)
	var channelCount int
	switch space {
	case ColorSpaceRGB, ColorSpaceHSL, ColorSpaceHSV, ColorSpaceHWB, ColorSpaceLAB, ColorSpaceLCH, ColorSpaceOKLAB, ColorSpaceOKLCH, ColorSpaceXYZ:
		channelCount = 3
	case ColorSpaceRGBA, ColorSpaceHSLA, ColorSpaceHSVA, ColorSpaceHWBA, ColorSpaceLABA, ColorSpaceLCHA, ColorSpaceOKLABA, ColorSpaceOKLCHA, ColorSpaceXYZA:
		channelCount = 4
	default:
		return nil, newError("invalid #clr payload")
	}
	if len(values) != channelCount+1 {
		return nil, newError("invalid #clr payload")
	}
	channels := make([]float64, channelCount)
	for i := range channels {
		value, ok := numberAsFloat64(values[i+1])
		if !ok {
			return nil, newError("invalid #clr payload")
		}
		channels[i] = value
	}
	if colorSpaceHasAlpha(space) && (channels[len(channels)-1] < 0 || channels[len(channels)-1] > 1) {
		return nil, newError("invalid #clr payload")
	}
	value, ok := colorValue(space, channels)
	if !ok {
		return nil, newError("invalid #clr payload")
	}

	return Color{
		Space:    space,
		Channels: channels,
		Value:    value,
	}, nil
}

func colorMember(value Color) objectMember {
	payload := make([]any, 0, len(value.Channels)+1)
	payload = append(payload, string(value.Space))
	for _, channel := range value.Channels {
		payload = append(payload, channel)
	}
	return objectMember{
		Key:   "#clr",
		Value: payload,
	}
}

func colorSpaceHasAlpha(space ColorSpace) bool {
	switch space {
	case ColorSpaceRGBA, ColorSpaceHSLA, ColorSpaceHSVA, ColorSpaceHWBA, ColorSpaceLABA, ColorSpaceLCHA, ColorSpaceOKLABA, ColorSpaceOKLCHA, ColorSpaceXYZA:
		return true
	default:
		return false
	}
}

func colorValue(space ColorSpace, channels []float64) (colorlib.Color, bool) {
	alpha := 1.0
	if colorSpaceHasAlpha(space) {
		alpha = channels[3]
	}
	switch space {
	case ColorSpaceRGB, ColorSpaceRGBA:
		return colorlib.NewRGBA(channels[0], channels[1], channels[2], alpha), true
	case ColorSpaceHSL, ColorSpaceHSLA:
		return colorlib.NewHSL(channels[0], channels[1], channels[2], alpha), true
	case ColorSpaceHSV, ColorSpaceHSVA:
		return colorlib.NewHSV(channels[0], channels[1], channels[2], alpha), true
	case ColorSpaceHWB, ColorSpaceHWBA:
		return colorlib.NewHWB(channels[0], channels[1], channels[2], alpha), true
	case ColorSpaceLAB, ColorSpaceLABA:
		return colorlib.NewLAB(channels[0], channels[1], channels[2], alpha), true
	case ColorSpaceLCH, ColorSpaceLCHA:
		return colorlib.NewLCH(channels[0], channels[1], channels[2], alpha), true
	case ColorSpaceOKLAB, ColorSpaceOKLABA:
		return colorlib.NewOKLAB(channels[0], channels[1], channels[2], alpha), true
	case ColorSpaceOKLCH, ColorSpaceOKLCHA:
		return colorlib.NewOKLCH(channels[0], channels[1], channels[2], alpha), true
	case ColorSpaceXYZ, ColorSpaceXYZA:
		return colorlib.NewXYZ(channels[0], channels[1], channels[2], alpha), true
	default:
		return nil, false
	}
}
