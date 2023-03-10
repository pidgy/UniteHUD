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

	"github.com/skratchdot/open-golang/open"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/match"
	"github.com/pidgy/unitehud/notify"
	"github.com/pidgy/unitehud/server"
	"github.com/pidgy/unitehud/stats"
	"github.com/pidgy/unitehud/team"
)

var (
	Dir = fmt.Sprintf("tmp/%d_%02d_%02d_%02d_%02d/", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute())

	now = time.Now()

	logs = fmt.Sprintf("%d.log", time.Now().Unix())

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

func Log() {
	err := createAllIfNotExist()
	if err != nil {
		notify.Error("Failed to create %s directory (%v)", Dir, err)
		return
	}

	f, err := os.OpenFile(fmt.Sprintf("%s/log/unitehud_%s", Dir, logs), os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		notify.Error("Failed to open log file (%v)", Dir, err)
		return
	}
	defer f.Close()

	for _, p := range notify.Feeds() {
		_, err := f.WriteString(fmt.Sprintf("%s\n", p.String()))
		if err != nil {
			notify.Error("Failed to write event logs (%v)", Dir, err)
		}
	}

	for _, line := range stats.Lines() {
		_, err := f.WriteString(fmt.Sprintf("%s\n", line))
		if err != nil {
			notify.Error("Failed to append statistic logs (%v)", Dir, err)
		}
	}
}

func Open() error {
	err := createAllIfNotExist()
	if err != nil {
		notify.Error("Failed to create %s directory (%v)", Dir, err)
		return err
	}

	return open.Run("tmp")
}

func ProfileStart() {
	var err error

	cpu, err = os.Create("cpu.prof")
	if err != nil {

	}

	err = pprof.StartCPUProfile(cpu)
	if err != nil {

	}

	ram, err = os.Create("mem.prof")
	if err != nil {

	}

	runtime.GC()

	err = pprof.WriteHeapProfile(ram)
	if err != nil {

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
