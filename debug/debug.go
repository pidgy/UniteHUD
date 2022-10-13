package debug

import (
	"fmt"
	"image"
	"os"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/skratchdot/open-golang/open"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/team"
)

var (
	Dir = fmt.Sprintf("tmp/%d_%02d_%02d_%02d_%02d/", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute())

	now = time.Now()

	logs    = fmt.Sprintf("%d.log", time.Now().Unix())
	logq    = make(chan string, 1024)
	logging = false

	cpu, ram *os.File

	counts     = map[string]int64{}
	countsLock = &sync.Mutex{}
)

func Capture(img image.Image, mat gocv.Mat, t *team.Team, p image.Point, value int, r match.Result) string {
	if mat.Empty() {
		return ""
	}

	subdir := fmt.Sprintf("%s/capture/%s", Dir, t.Name)
	err := createDirIfNotExist(subdir)
	if err != nil {
		notify.Error("[DEBUG] failed to create directory \"%s\" (%v)", subdir, err)
		return ""
	}

	subdir = fmt.Sprintf("%s/%s", subdir, r.String())
	err = createDirIfNotExist(subdir)
	if err != nil {
		notify.Error("[DEBUG] failed to create directory \"%s\" (%v)", subdir, err)
		return ""
	}

	file := filename(t.Name, subdir, value)

	gocv.IMWrite(file, mat.Region(t.Crop(p)))

	return file
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
	err := createTmpIfNotExist()
	if err != nil {
		return err
	}

	return open.Run("tmp")
}

func LoggingStart() error {
	err := createAllIfNotExist()
	if err != nil {
		log.Error().Err(err).Msgf("failed to create %s directory", Dir)
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

func createAllIfNotExist() error {
	err := createTmpIfNotExist()
	if err != nil {
		return err
	}

	for _, subdir := range []string{
		"/",
		"/log",
		"/capture",
	} {
		err := createDirIfNotExist(Dir + subdir)
		if err != nil {
			return err
		}
	}

	return nil
}

func createDirIfNotExist(subdir string) error {
	_, err := os.Stat(Dir)
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

	err = os.Mkdir(dir+"/"+subdir, 0755)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}

		return err
	}

	return nil
}

func createTmpIfNotExist() error {
	dir, err := os.Getwd()
	if err != nil {
		log.Error().Err(err).Msg("failed to find working directory")
		return err
	}

	err = os.Mkdir(fmt.Sprintf("%s/tmp", dir), 0755)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}

		return err
	}

	return nil
}

func filename(name, subdir string, value int) string {
	countsLock.Lock()
	defer countsLock.Unlock()

	counts[name]++

	p := fmt.Sprintf(
		"%s/%d@%s_#%d.png",
		subdir,
		value,
		strings.ReplaceAll(server.Clock(), ":", ""),
		counts[name],
	)

	return p
}

func spin() {
	f, err := os.OpenFile(fmt.Sprintf("%s/log/unitehud_%s", Dir, logs), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
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
