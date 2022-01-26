package dev

import (
	"fmt"
	"image"
	"image/png"
	"math/rand"
	"os"
	"time"

	"github.com/skratchdot/open-golang/open"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/rgba"
	"github.com/rs/zerolog/log"
)

var logFilename = fmt.Sprintf("%d.log", time.Now().Unix())
var lastlog = ""
var dir = "tmp"

func Capture(img image.Image, mat gocv.Mat, subdir string, order string, duplicate bool, value int) {
	subdir = fmt.Sprintf("%s/capture/%s/", dir, subdir)
	file := fmt.Sprintf("%d@%d_%s_%d", time.Now().UnixNano(), rand.Int()%99, order, value)

	if duplicate {
		if !config.Current.RecordDuplicates && !config.Current.Record {
			return
		}

		file += "_duplicate"
	}

	path := fmt.Sprintf("%s/%s.png", subdir, file)

	f, err := os.Create(path)
	if err != nil {
		log.Error().Err(err).Msg("failed to create missed image")
		return
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode missed image")
		return
	}

	img, err = mat.ToImage()
	if err != nil {
		log.Error().Err(err).Msg("failed to convert matrix to image")
		return
	}

	crop := fmt.Sprintf("%s/%s_crop.png", subdir, file)

	f, err = os.Create(crop)
	if err != nil {
		log.Error().Err(err).Msg("failed to create missed image")
		return
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode missed image")
		return
	}

	notify.Feed(rgba.Yellow, "Saved as %s in %s", file, subdir)
}

func End() {
	if lastlog == "end" {
		return
	}

	Log("end")
}

func Log(txt string) {
	f, err := os.OpenFile(fmt.Sprintf("%s/log/unitehud_%s", dir, logFilename), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error().Err(err).Msg("failed to open log file")
		return
	}
	defer f.Close()

	lastlog = txt

	txt = fmt.Sprintf("%s | %s\n", time.Now().Format(time.Kitchen), txt)

	_, err = f.WriteString(txt)
	if err != nil {
		log.Error().Err(err).Msg("failed to find working directory")
		return
	}
}

func New() error {
	var err error

	dir, err := os.Getwd()
	if err != nil {
		log.Error().Err(err).Msg("failed to find working directory")
		return err
	}

	for _, subdir := range []string{
		"/tmp",
		"/tmp/log",
		"/tmp/capture",
		"/tmp/capture/purple",
		"/tmp/capture/orange",
		"/tmp/capture/self",
		"/tmp/capture/time",
	} {
		err := os.Mkdir(dir+subdir, 0755)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return err
		}
	}

	logFilename = fmt.Sprintf("%d.log", time.Now().Unix())

	return nil
}

func Open() error {
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		return fmt.Errorf("failed to find %s directory", dir)
	}

	return open.Run(dir)
}

func Start() {
	if lastlog == "start" {
		return
	}

	Log("start")
}
