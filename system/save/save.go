package save

import (
	"fmt"
	"image"
	"os"
	"path/filepath"
	"runtime"
	"runtime/pprof"
	"strings"
	"sync"
	"time"

	"github.com/skratchdot/open-golang/open"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/core/global"
	"github.com/pidgy/unitehud/core/match"
	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/core/stats"
	"github.com/pidgy/unitehud/core/team"
)

var (
	Directory = fmt.Sprintf("%d_%02d_%02d_%02d_%02d", now.Year(), now.Month(), now.Day(), now.Hour(), now.Minute())

	top    = "saved"
	images = "img"
	logs   = "log"

	now = time.Now()

	logfile = fmt.Sprintf("unitehud_%d.log", time.Now().Unix())

	cpu, ram *os.File

	counts     = map[string]int64{}
	countsLock = &sync.Mutex{}
)

func Image(img image.Image, mat gocv.Mat, t *team.Team, p image.Point, value int, r match.Result) string {
	if mat.Empty() {
		return ""
	}

	subdir := filepath.Join(global.WorkingDirectory(), top, Directory, images, t.Name)
	err := createDirIfNotExist(subdir)
	if err != nil {
		notify.Error("💾 failed to create directory \"%s\" (%v)", subdir, err)
		return ""
	}

	subdir = filepath.Join(subdir, r.String())
	err = createDirIfNotExist(subdir)
	if err != nil {
		notify.Error("💾 failed to create directory \"%s\" (%v)", subdir, err)
		return ""
	}

	file := name(t.Name, subdir, value)

	gocv.IMWrite(file, mat.Region(t.Crop(p)))

	return file
}

func Logs() {
	_, err := createAllIfNotExist()
	if err != nil {
		notify.Error("💾 Failed to create %s directory (%v)", Directory, err)
		return
	}

	dir := filepath.Join(global.WorkingDirectory(), top, Directory, logs, logfile)

	f, err := os.OpenFile(dir, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		notify.Error("💾 Failed to open log file in %s (%v)", Directory, err)
		return
	}
	defer f.Close()

	for _, p := range notify.Feeds() {
		_, err := f.WriteString(fmt.Sprintf("%s\n", p.String()))
		if err != nil {
			notify.Error("💾 Failed to write event logs in %s (%v)", Directory, err)
		}
	}

	for _, line := range stats.Lines() {
		_, err := f.WriteString(fmt.Sprintf("%s\n", line))
		if err != nil {
			notify.Error("💾 Failed to append statistic logs in %s (%v)", Directory, err)
		}
	}
}

func Open() error {
	d, err := createAllIfNotExist()
	if err != nil {
		notify.Error("💾 Failed to create \"%s/\" (%v)", Directory, err)
		return err
	}

	return open.Run(d)
}

func OpenLogDirectory() error {
	d, err := createAllIfNotExist()
	if err != nil {
		notify.Error("💾 Failed to create \"%s/\" directory (%v)", Directory, err)
		return err
	}
	return open.Run(fmt.Sprintf("%s/%s/%s", d, Directory, logs))
}

func ProfileStart() {
	var err error

	cpu, err = os.Create("cpu.prof")
	if err != nil {
		notify.Error("💾 Failed to create CPU profile (%v)", err)
		return
	}

	err = pprof.StartCPUProfile(cpu)
	if err != nil {
		notify.Error("💾 Failed to start CPU profile (%v)", err)
		return
	}

	ram, err = os.Create("mem.prof")
	if err != nil {
		notify.Error("💾 Failed to create RAM profile (%v)", err)
		return
	}

	runtime.GC()

	err = pprof.WriteHeapProfile(ram)
	if err != nil {
		notify.Error("💾 Failed to write RAM profile (%v)", err)
		return
	}
}

func ProfileStop() {
	pprof.StopCPUProfile()
	cpu.Close()
	ram.Close()
}

func createAllIfNotExist() (string, error) {
	d, err := createTmpIfNotExist()
	if err != nil {
		return "", err
	}

	for _, subdir := range []string{
		"/",
		fmt.Sprintf("/%s", logs),
		fmt.Sprintf("/%s", images),
	} {
		err := createDirIfNotExist(filepath.Join(global.WorkingDirectory(), top, Directory, subdir))
		if err != nil {
			return "", err
		}
	}

	return d, nil
}

func createDirIfNotExist(subdir string) error {
	_, err := os.Stat(subdir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
	}

	err = os.Mkdir(subdir, 0755)
	if err != nil {
		if os.IsExist(err) {
			return nil
		}

		return err
	}

	return nil
}

func createTmpIfNotExist() (string, error) {
	dir := filepath.Join(global.WorkingDirectory(), top)

	err := os.Mkdir(dir, 0755)
	if err != nil {
		if os.IsExist(err) {
			return dir, nil
		}
		return "", err
	}

	return dir, nil
}

func name(name, subdir string, value int) string {
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
