package splash

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/config"
	"github.com/pidgy/unitehud/core/notify"
)

var (
	defaultMat             = gocv.IMRead(fmt.Sprintf(`%s/splash/default.png`, config.Current.Assets()), gocv.IMReadColor) // Global matrix is more efficient?
	defaultImg image.Image = nil

	deviceMat              = gocv.IMRead(fmt.Sprintf(`%s/splash/device.png`, config.Current.Assets()), gocv.IMReadColor) // Global matrix is more efficient?
	deviceImg  image.Image = nil
	deviceRGBA *image.RGBA = nil

	invalidMat              = gocv.IMRead(fmt.Sprintf(`%s/splash/invalid.png`, config.Current.Assets()), gocv.IMReadColor)
	invalidImg  image.Image = nil
	invalidRGBA *image.RGBA = nil

	loadingMat             = gocv.IMRead(fmt.Sprintf(`%s/splash/loading.png`, config.Current.Assets()), gocv.IMReadColor)
	loadingImg image.Image = nil

	projectorMat             = gocv.IMRead(fmt.Sprintf(`%s/splash/projector.png`, config.Current.Assets()), gocv.IMReadColor)
	projectorImg image.Image = nil
)

func init() {
	notify.Preview = Projector()

	if projectorMat.Empty() {
		m, err := gocv.ImageToMatRGBA(defaultPNG)
		if err == nil {
			_ = projectorMat.Close()
			projectorMat = m
		}
	}
}

func AsRGBA(i image.Image) *image.RGBA {
	if i == nil {
		return &image.RGBA{}
	}

	rgba, ok := i.(*image.RGBA)
	if !ok {
		return &image.RGBA{Rect: i.Bounds()}
	}

	return rgba
}

func Default() image.Image {
	if defaultImg == nil {
		if defaultMat.Empty() {
			defaultMat = defaultPNGToMat()
		}

		i, err := defaultMat.ToImage()
		if err != nil {
			notify.Warn("Splash: Failed to convert splash image (%v)", err)
			return defaultPNG
		}
		defaultImg = i
	}

	return defaultImg
}

func Device() image.Image {
	if deviceImg == nil {
		if deviceMat.Empty() {
			deviceMat = defaultPNGToMat()
		}

		i, err := deviceMat.ToImage()
		if err != nil {
			notify.Warn("Splash: Failed to convert splash image (%v)", err)
			return defaultPNG
		}
		deviceImg = i
	}
	return deviceImg
}

func DeviceMat() *gocv.Mat {
	if deviceMat.Empty() {
		deviceMat = defaultPNGToMat()
	}

	return &deviceMat
}

func DeviceRGBA() *image.RGBA {
	if deviceRGBA == nil {
		if deviceMat.Empty() {
			deviceMat = defaultPNGToMat()
		}

		i, err := deviceMat.ToImage()
		if err != nil {
			notify.Warn("Splash: Failed to convert splash image (%v)", err)
			return defaultPNG
		}
		deviceRGBA = AsRGBA(i)
	}

	return deviceRGBA
}

func Invalid() image.Image {
	if invalidImg == nil {
		if invalidMat.Empty() {
			invalidMat = defaultPNGToMat()
		}

		i, err := invalidMat.ToImage()
		if err != nil {
			notify.Warn("Splash: Failed to convert splash image (%v)", err)
			return defaultPNG
		}
		invalidImg = i
	}

	return invalidImg
}

func InvalidRGBA() *image.RGBA {
	if invalidRGBA == nil {
		if invalidMat.Empty() {
			invalidMat = defaultPNGToMat()
		}

		i, err := invalidMat.ToImage()
		if err != nil {
			notify.Warn("Splash: Failed to convert splash image (%v)", err)
			return defaultPNG
		}
		invalidRGBA = AsRGBA(i)
	}
	return invalidRGBA
}

func Loading() image.Image {
	if loadingImg == nil {
		if loadingMat.Empty() {
			loadingMat = defaultPNGToMat()
		}

		i, err := loadingMat.ToImage()
		if err != nil {
			notify.Warn("Splash: Failed to convert splash image (%v)", err)
			return defaultPNG
		}
		loadingImg = i
	}

	return loadingImg
}

func Projector() image.Image {
	if projectorImg == nil {
		if projectorMat.Empty() {
			projectorMat = defaultPNGToMat()
		}

		i, err := projectorMat.ToImage()
		if err != nil {
			notify.Warn("Failed to convert splash image (%v)", err)
			return defaultPNG
		}
		projectorImg = i
	}

	return projectorImg
}
