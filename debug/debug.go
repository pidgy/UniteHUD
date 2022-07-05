package debug

import (
	"fmt"
	"image"
	"image/png"
	"os"
	"runtime"
	"runtime/pprof"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/skratchdot/open-golang/open"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/team"
)

var (
	dir = "tmp"

	logs    = fmt.Sprintf("%d.log", time.Now().Unix())
	logq    = make(chan string, 1024)
	logging = false

	cpu, ram *os.File
)

func Capture(img image.Image, mat gocv.Mat, t *team.Team, p image.Point, name string, value int) string {
	if name == "" {
		name = "capture"
	}

	subdir := fmt.Sprintf("%s/capture/%s/", dir, t.Name)
	file := fmt.Sprintf("%d_%s-%d", value, name, time.Now().UnixNano())
	path := fmt.Sprintf("%s/%s.png", subdir, file)

	f, err := os.Create(path)
	if err != nil {
		log.Error().Err(err).Msg("failed to create missed image")
		return ""
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode missed image")
		return ""
	}

	err = png.Encode(f, img)
	if err != nil {
		log.Error().Err(err).Msg("failed to encode missed image")
		return ""
	}

	if mat.Empty() {
		return ""
	}

	gocv.IMWrite(fmt.Sprintf("%s/%s_crop.png", subdir, file), mat.Region(t.Crop(p)))

	return path
}

func Close() {
	close(logq)
}

func Log(format string, a ...interface{}) {
	if !config.Current.Record {
		return
	}

	txt := fmt.Sprintf(format, a...)
	logq <- fmt.Sprintf("[%s] | %s\n", time.Now().Format(time.Stamp), txt)
}

func Open() error {
	err := createIfNotExist()
	if err != nil {
		return err
	}

	return open.Run(dir)
}

func LoggingStart() error {
	err := createIfNotExist()
	if err != nil {
		log.Error().Err(err).Msg("failed to create tmp directory")
		return err
	}

	if !logging {
		go spin()
		logging = true
	}

	Log("Start")

	return nil
}

func LoggingStop() {
	Log("End")
}

func ProfileStart() {
	var err error

	cpu, err = os.Create("cpu.prof")
	if err != nil {
		log.Panic().Err(err).Msg("failed to create cpu profile")
	}

	err = pprof.StartCPUProfile(cpu)
	if err != nil {
		log.Panic().Err(err).Msg("failed to start CPU profile")
	}

	ram, err = os.Create("mem.prof")
	if err != nil {
		log.Panic().Err(err).Msg("failed to create RAM profile")
	}

	runtime.GC()

	err = pprof.WriteHeapProfile(ram)
	if err != nil {
		log.Panic().Err(err).Msg("failed to write RAM profile")
	}
}

func ProfileStop() {
	pprof.StopCPUProfile()
	cpu.Close()
	ram.Close()
}

func createIfNotExist() error {
	_, err := os.Stat(dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

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
		"/tmp/capture/first",
	} {
		err := os.Mkdir(dir+subdir, 0755)
		if err != nil {
			if os.IsExist(err) {
				continue
			}
			return err
		}
	}

	return nil
}

func spin() {
	f, err := os.OpenFile(fmt.Sprintf("%s/log/unitehud_%s", dir, logs), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Error().Err(err).Msg("failed to open log file")
		return
	}
	defer f.Close()

	for txt := range logq {
		_, err := f.WriteString(txt)
		if err != nil {
			log.Error().Err(err).Str("file", logs).Msg("failed to write log")
		}
	}
}
