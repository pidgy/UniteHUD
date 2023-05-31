package splash

import (
	"fmt"
	"image"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/img"
	"github.com/pidgy/unitehud/notify"
)

var (
	defaultMat             = gocv.IMRead(fmt.Sprintf(`%s/splash/default.png`, config.Current.Assets()), gocv.IMReadColor) // Global matrix is more efficient?
	defaultImg image.Image = nil

	deviceMat             = gocv.IMRead(fmt.Sprintf(`%s/splash/device.png`, config.Current.Assets()), gocv.IMReadColor) // Global matrix is more efficient?
	deviceImg image.Image = nil

	invalidMat             = gocv.IMRead(fmt.Sprintf(`%s/splash/invalid.png`, config.Current.Assets()), gocv.IMReadColor)
	invalidImg image.Image = nil

	loadingMat             = gocv.IMRead(fmt.Sprintf(`%s/splash/loading.png`, config.Current.Assets()), gocv.IMReadColor)
	loadingImg image.Image = nil
)

func Default() image.Image {
	if defaultImg == nil {
		i, err := defaultMat.ToImage()
		if err != nil {
			notify.Warn("Failed to convert splash image (%v)", err)
			return img.Empty
		}
		defaultImg = i
	}

	return defaultImg
}

func Device() image.Image {
	if deviceImg == nil {
		i, err := deviceMat.ToImage()
		if err != nil {
			notify.Warn("Failed to convert splash image (%v)", err)
			return img.Empty
		}
		deviceImg = i
	}
	return deviceImg
}

func DeviceMat() *gocv.Mat {
	return &deviceMat
}

func Invalid() image.Image {
	if invalidImg == nil {
		i, err := invalidMat.ToImage()
		if err != nil {
			notify.Warn("Failed to convert splash image (%v)", err)
			return img.Empty
		}
		invalidImg = i
	}
	return invalidImg
}

func Loading() image.Image {
	if loadingImg == nil {
		i, err := loadingMat.ToImage()
		if err != nil {
			notify.Warn("Failed to convert splash image (%v)", err)
			return img.Empty
		}
		loadingImg = i
	}

	return loadingImg
}
