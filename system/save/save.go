package save

import (
	"encoding/json"
	"fmt"
	"image"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/skratchdot/open-golang/open"
	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/global"
)

const (
	// Directories.
	saved  = "saved"
	images = "img"

	// Files.
	log       = "unitehud.log"
	templates = "templates.json"
)

var (
	Directory = fmt.Sprintf("%4d-%02d-%02d", now.Year(), now.Month(), now.Day())

	cpu, ram   *os.File
	counts     = map[string]int64{}
	countsLock = &sync.Mutex{}
	now        = time.Now()
)

func Image(img image.Image, mat gocv.Mat, crop image.Rectangle, value int, team, result, clock string) error {
	if mat.Empty() {
		return nil
	}

	subdir := filepath.Join(global.WorkingDirectory(), saved, Directory, images, team)
	err := createDirIfNotExist(subdir)
	if err != nil {
		return fmt.Errorf("failed to create directory: %s: %v", subdir, err)
	}

	subdir = filepath.Join(subdir, result)
	err = createDirIfNotExist(subdir)
	if err != nil {
		return fmt.Errorf("failed to create directory: %s: %v", subdir, err)
	}

	file := name(team, subdir, clock, value)

	if !gocv.IMWrite(file, mat.Region(crop)) {
		return fmt.Errorf("failed to save image: %s", file)
	}

	return nil
}

func Logs(feeds, lines []string, templates map[string]int) error {
	_, err := createAllIfNotExist()
	if err != nil {
		return fmt.Errorf("save: failed to create directory: %s: %v", Directory, err)
	}

	dir := filepath.Join(global.WorkingDirectory(), saved, Directory, log)

	f, err := os.OpenFile(dir, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("save: failed to open log file: %s: %v", Directory, err)
	}
	defer f.Close()

	for _, p := range feeds {
		_, err := f.WriteString(fmt.Sprintf("%s\n", p))
		if err != nil {
			return fmt.Errorf("save: failed to write event log: %s: %v", Directory, err)
		}
	}

	for _, line := range lines {
		_, err := f.WriteString(fmt.Sprintf("%s\n", line))
		if err != nil {
			return fmt.Errorf("save: failed to append statistic log: %s: %v", Directory, err)
		}
	}

	return templateStatistics(templates)
}

func Open() error {
	d, err := createAllIfNotExist()
	if err != nil {
		return fmt.Errorf("save: failed to create %s: %v", Directory, err)
	}
	return open.Run(d)
}

func OpenCurrentLog() error {
	d, err := createAllIfNotExist()
	if err != nil {
		return fmt.Errorf("save: failed to create directory: %s: %v", Directory, err)
	}
	return open.Run(fmt.Sprintf("%s/%s/%s", d, Directory, log))
}

func OpenLogDirectory() error {
	d, err := createAllIfNotExist()
	if err != nil {
		return fmt.Errorf("save: failed to create directory: %s: %v", Directory, err)
	}
	return open.Run(fmt.Sprintf("%s/%s", d, Directory))
}

func createAllIfNotExist() (string, error) {
	d, err := createTmpIfNotExist()
	if err != nil {
		return "", err
	}

	for _, subdir := range []string{
		"/",
		fmt.Sprintf("/%s", images),
	} {
		err := createDirIfNotExist(filepath.Join(global.WorkingDirectory(), saved, Directory, subdir))
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
	dir := filepath.Join(global.WorkingDirectory(), saved)

	err := os.Mkdir(dir, 0755)
	if err != nil {
		if os.IsExist(err) {
			return dir, nil
		}
		return "", err
	}

	return dir, nil
}

func name(name, subdir, clock string, value int) string {
	countsLock.Lock()
	defer countsLock.Unlock()

	counts[name]++

	return fmt.Sprintf(
		"%s/%d@%s_#%d.png",
		subdir,
		value,
		strings.ReplaceAll(clock, ":", ""),
		counts[name],
	)
}

func sortedJSON(r json.RawMessage) json.RawMessage {
	var i interface{}
	err := json.Unmarshal(r, &i)
	if err != nil {
		return r
	}

	b, err := json.MarshalIndent(i, "", "    ")
	if err != nil {
		return r
	}
	return b
}

func templateStatistics(t map[string]int) error {
	// Append and save statistics from today.
	today := filepath.Join(global.WorkingDirectory(), saved, Directory, templates)

	raw, err := os.ReadFile(today)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("save: failed to open %s: %v", templates, err)
		}
	}
	if len(raw) == 0 {
		raw = []byte("{}")
	}

	current := make(map[string]int)

	err = json.Unmarshal(raw, &current)
	if err != nil {
		return fmt.Errorf("save: failed to unpack %s: %v", templates, err)
	}

	for k, v := range t {
		current[k] += v
	}

	raw, err = json.Marshal(current)
	if err != nil {
		return fmt.Errorf("save: failed to pack %s: %v", templates, err)
	}

	err = os.WriteFile(today, sortedJSON(raw), os.ModePerm)
	if err != nil {
		return fmt.Errorf("save: failed to save %s: %v", today, err)
	}

	// Append and save statistics from all time.
	all := filepath.Join(global.WorkingDirectory(), saved, templates)

	raw, err = os.ReadFile(all)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("save: failed to open %s: %v", templates, err)
		}
	}
	if len(raw) == 0 {
		raw = []byte("{}")
	}

	total := make(map[string]int)

	err = json.Unmarshal(raw, &total)
	if err != nil {
		return fmt.Errorf("save: failed to unpack %s: %v", templates, err)
	}

	for k, v := range t {
		total[k] += v
	}

	raw, err = json.Marshal(total)
	if err != nil {
		return fmt.Errorf("save: failed to pack %s: %v", templates, err)
	}

	err = os.WriteFile(all, sortedJSON(raw), os.ModePerm)
	if err != nil {
		return fmt.Errorf("save: failed to save %s: %v", all, err)
	}

	return nil
}
