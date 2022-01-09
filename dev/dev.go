package dev

import (
	"fmt"
	"image"
	"image/png"
	"math/rand"
	"os"
	"time"

	"github.com/rs/zerolog/log"
	"gocv.io/x/gocv"
)

var logFilename = fmt.Sprintf("%d.log", time.Now().Unix())
var lastlog = ""
var dir = "tmp/"

func Capture(img image.Image, mat gocv.Mat, subdir string, order string, duplicate bool, value int) {
	subdir = fmt.Sprintf("%s/capture/%s/", dir, subdir)
	file := fmt.Sprintf("%d@%d_%s_%d", time.Now().UnixNano(), rand.Int()%99, order, value)

	if duplicate {
		file += "_duplicate"
	}

	f, err := os.Create(fmt.Sprintf("%s/%s.png", subdir, file))
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

	f, err = os.Create(fmt.Sprintf("%s/%s_crop.png", subdir, file))
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

func Start() {
	if lastlog == "start" {
		return
	}

	Log("start")
}
