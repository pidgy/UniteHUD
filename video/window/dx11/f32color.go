package dx11

import (
	"image/color"
	"math"
)

//go:generate go run ./f32colorgen -out tables.go

// RGBA is a 32 bit floating point linear premultiplied color space.
type RGBA struct {
	R, G, B, A float32
}

// Array returns rgba values in a [4]float32 array.
func (rgba RGBA) Array() [4]float32 {
	return [4]float32{rgba.R, rgba.G, rgba.B, rgba.A}
}

// Float32 returns r, g, b, a values.
func (col RGBA) Float32() (r, g, b, a float32) {
	return col.R, col.G, col.B, col.A
}

// SRGBA converts from linear to sRGB color space.
func (col RGBA) SRGB() color.NRGBA {
	if col.A == 0 {
		return color.NRGBA{}
	}
	return color.NRGBA{
		R: uint8(linearTosRGB(col.R/col.A)*255 + .5),
		G: uint8(linearTosRGB(col.G/col.A)*255 + .5),
		B: uint8(linearTosRGB(col.B/col.A)*255 + .5),
		A: uint8(col.A*255 + .5),
	}
}

// Luminance calculates the relative luminance of a linear RGBA color.
// Normalized to 0 for black and 1 for white.
//
// See https://www.w3.org/TR/WCAG20/#relativeluminancedef for more details
func (col RGBA) Luminance() float32 {
	return 0.2126*col.R + 0.7152*col.G + 0.0722*col.B
}

// Opaque returns the color without alpha component.
func (col RGBA) Opaque() RGBA {
	col.A = 1.0
	return col
}

// LinearFromSRGB converts from col in the sRGB colorspace to RGBA.
func LinearFromSRGB(col color.NRGBA) RGBA {
	af := float32(col.A) / 0xFF
	return RGBA{
		R: srgb8ToLinear[col.R] * af, // sRGBToLinear(float32(col.R)/0xff) * af,
		G: srgb8ToLinear[col.G] * af, // sRGBToLinear(float32(col.G)/0xff) * af,
		B: srgb8ToLinear[col.B] * af, // sRGBToLinear(float32(col.B)/0xff) * af,
		A: af,
	}
}

// NRGBAToRGBA converts from non-premultiplied sRGB color to premultiplied sRGB color.
//
// Each component in the result is `sRGBToLinear(c * alpha)`, where `c`
// is the linear color.
func NRGBAToRGBA(col color.NRGBA) color.RGBA {
	if col.A == 0xFF {
		return color.RGBA(col)
	}
	c := LinearFromSRGB(col)
	return color.RGBA{
		R: uint8(linearTosRGB(c.R)*255 + .5),
		G: uint8(linearTosRGB(c.G)*255 + .5),
		B: uint8(linearTosRGB(c.B)*255 + .5),
		A: col.A,
	}
}

// NRGBAToLinearRGBA converts from non-premultiplied sRGB color to premultiplied linear RGBA color.
//
// Each component in the result is `c * alpha`, where `c` is the linear color.
func NRGBAToLinearRGBA(col color.NRGBA) color.RGBA {
	if col.A == 0xFF {
		return color.RGBA(col)
	}
	c := LinearFromSRGB(col)
	return color.RGBA{
		R: uint8(c.R*255 + .5),
		G: uint8(c.G*255 + .5),
		B: uint8(c.B*255 + .5),
		A: col.A,
	}
}

// RGBAToNRGBA converts from premultiplied sRGB color to non-premultiplied sRGB color.
func RGBAToNRGBA(col color.RGBA) color.NRGBA {
	if col.A == 0xFF {
		return color.NRGBA(col)
	}

	linear := RGBA{
		R: sRGBToLinear(float32(col.R) / 0xff),
		G: sRGBToLinear(float32(col.G) / 0xff),
		B: sRGBToLinear(float32(col.B) / 0xff),
		A: float32(col.A) / 0xff,
	}

	return linear.SRGB()
}

// linearTosRGB transforms color value from linear to sRGB.
func linearTosRGB(c float32) float32 {
	// Formula from EXT_sRGB.
	switch {
	case c <= 0:
		return 0
	case 0 < c && c < 0.0031308:
		return 12.92 * c
	case 0.0031308 <= c && c < 1:
		return 1.055*float32(math.Pow(float64(c), 0.41666)) - 0.055
	}

	return 1
}

// sRGBToLinear transforms color value from sRGB to linear.
func sRGBToLinear(c float32) float32 {
	// Formula from EXT_sRGB.
	if c <= 0.04045 {
		return c / 12.92
	} else {
		return float32(math.Pow(float64((c+0.055)/1.055), 2.4))
	}
}

// MulAlpha applies the alpha to the color.
func MulAlpha(c color.NRGBA, alpha uint8) color.NRGBA {
	c.A = uint8(uint32(c.A) * uint32(alpha) / 0xFF)
	return c
}

// Disabled blends color towards the luminance and multiplies alpha.
// Blending towards luminance will desaturate the color.
// Multiplying alpha blends the color together more with the background.
func Disabled(c color.NRGBA) (d color.NRGBA) {
	const r = 80 // blend ratio
	lum := approxLuminance(c)
	d = mix(c, color.NRGBA{A: c.A, R: lum, G: lum, B: lum}, r)
	d = MulAlpha(d, 128+32)
	return
}

// Hovered blends dark colors towards white, and light colors towards
// black. It is approximate because it operates in non-linear sRGB space.
func Hovered(c color.NRGBA) (h color.NRGBA) {
	if c.A == 0 {
		// Provide a reasonable default for transparent widgets.
		return color.NRGBA{A: 0x44, R: 0x88, G: 0x88, B: 0x88}
	}
	const ratio = 0x20
	m := color.NRGBA{R: 0xff, G: 0xff, B: 0xff, A: c.A}
	if approxLuminance(c) > 128 {
		m = color.NRGBA{A: c.A}
	}
	return mix(m, c, ratio)
}

// mix mixes c1 and c2 weighted by (1 - a/256) and a/256 respectively.
func mix(c1, c2 color.NRGBA, a uint8) color.NRGBA {
	ai := int(a)
	return color.NRGBA{
		R: byte((int(c1.R)*ai + int(c2.R)*(256-ai)) / 256),
		G: byte((int(c1.G)*ai + int(c2.G)*(256-ai)) / 256),
		B: byte((int(c1.B)*ai + int(c2.B)*(256-ai)) / 256),
		A: byte((int(c1.A)*ai + int(c2.A)*(256-ai)) / 256),
	}
}

// approxLuminance is a fast approximate version of RGBA.Luminance.
func approxLuminance(c color.NRGBA) byte {
	const (
		r = 13933 // 0.2126 * 256 * 256
		g = 46871 // 0.7152 * 256 * 256
		b = 4732  // 0.0722 * 256 * 256
		t = r + g + b
	)
	return byte((r*int(c.R) + g*int(c.G) + b*int(c.B)) / t)
}

// table corresponds to sRGBToLinear(float32(index)/0xff)
var srgb8ToLinear = [...]float32{
	0, 0.000303527, 0.000607054, 0.000910581, 0.001214108, 0.001517635, 0.001821162, 0.0021246888, 0.002428216, 0.002731743, 0.00303527, 0.0033465363, 0.0036765079, 0.004024718, 0.004391443, 0.004776954,
	0.005181518, 0.0056053926, 0.006048834, 0.0065120924, 0.0069954116, 0.007499033, 0.008023194, 0.008568126, 0.009134059, 0.00972122, 0.010329825, 0.010960096, 0.011612247, 0.012286489, 0.0129830325, 0.013702083,
	0.014443846, 0.015208517, 0.015996296, 0.016807377, 0.017641956, 0.01850022, 0.019382365, 0.020288566, 0.021219013, 0.022173887, 0.023153368, 0.024157634, 0.025186861, 0.026241226, 0.027320895, 0.028426042,
	0.029556837, 0.030713446, 0.031896036, 0.033104766, 0.03433981, 0.035601318, 0.03688945, 0.03820437, 0.039546244, 0.040915202, 0.042311415, 0.043735035, 0.04518621, 0.04666509, 0.048171826, 0.049706567,
	0.05126947, 0.05286066, 0.05448029, 0.056128502, 0.05780544, 0.059511248, 0.06124608, 0.06301004, 0.06480329, 0.06662596, 0.06847819, 0.07036012, 0.07227187, 0.07421359, 0.0761854, 0.078187436,
	0.080219835, 0.08228272, 0.08437622, 0.08650047, 0.08865561, 0.09084174, 0.09305899, 0.09530749, 0.09758737, 0.09989875, 0.102241755, 0.10461651, 0.10702312, 0.10946173, 0.11193245, 0.11443539,
	0.11697068, 0.11953844, 0.122138806, 0.12477185, 0.12743771, 0.1301365, 0.13286835, 0.13563335, 0.13843164, 0.14126332, 0.14412849, 0.14702728, 0.1499598, 0.15292618, 0.15592648, 0.15896088,
	0.16202942, 0.16513222, 0.16826941, 0.17144111, 0.1746474, 0.17788842, 0.18116425, 0.18447499, 0.18782078, 0.19120169, 0.19461782, 0.19806932, 0.20155625, 0.20507872, 0.20863685, 0.21223074,
	0.21586055, 0.21952623, 0.223228, 0.2269659, 0.23074009, 0.23455067, 0.23839766, 0.24228121, 0.24620141, 0.25015837, 0.25415218, 0.25818294, 0.26225075, 0.2663557, 0.27049786, 0.2746774,
	0.27889434, 0.28314883, 0.2874409, 0.29177073, 0.29613835, 0.30054384, 0.30498737, 0.30946898, 0.31398878, 0.31854683, 0.32314327, 0.32777816, 0.33245158, 0.33716366, 0.34191447, 0.3467041,
	0.35153273, 0.35640025, 0.3613069, 0.36625272, 0.37123778, 0.37626222, 0.3813261, 0.38642955, 0.39157256, 0.39675534, 0.40197787, 0.4072403, 0.4125427, 0.41788515, 0.42326775, 0.42869058,
	0.4341537, 0.43965724, 0.44520128, 0.45078585, 0.4564111, 0.46207705, 0.46778387, 0.47353154, 0.47932023, 0.48515, 0.4910209, 0.49693304, 0.5028866, 0.50888145, 0.5149178, 0.5209957,
	0.5271153, 0.53327656, 0.5394796, 0.5457246, 0.55201155, 0.5583405, 0.56471163, 0.5711249, 0.5775806, 0.58407855, 0.59061897, 0.5972019, 0.6038274, 0.6104957, 0.61720663, 0.6239605,
	0.6307572, 0.63759696, 0.64447975, 0.6514057, 0.6583749, 0.66538733, 0.6724432, 0.67954254, 0.6866855, 0.6938719, 0.7011021, 0.70837593, 0.71569365, 0.7230553, 0.7304609, 0.73791057,
	0.74540436, 0.7529423, 0.76052463, 0.7681513, 0.77582234, 0.7835379, 0.79129803, 0.79910284, 0.80695236, 0.8148467, 0.82278585, 0.83076996, 0.8387991, 0.84687334, 0.8549927, 0.8631573,
	0.8713672, 0.87962234, 0.8879232, 0.89626944, 0.90466136, 0.9130987, 0.92158204, 0.9301109, 0.9386859, 0.9473066, 0.9559735, 0.9646863, 0.9734455, 0.9822506, 0.9911022, 1,
}
