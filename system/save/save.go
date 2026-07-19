package save

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const. Then remove this comment.

import (
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/skratchdot/open-golang/open"

	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/system/win32"
)

var (
	Directory      = now.Format(time.DateOnly)
	ImageDirectory = fmt.Sprintf("%s/img", Directory)
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
	cpu, ram   *os.File
	counts     = map[string]int64{}
	countsLock = &sync.Mutex{}
	now        = time.Now()
)

func Image(img image.Image, value int, team, result, clock string) error {
	if img.Bounds().Empty() {
		return nil
	}

	subdir := filepath.Join(exe.Directory(), saved, Directory, images, team)
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

	f, err := os.Create(file)
	if err != nil {
		return err
	}
	defer f.Close()

	err = png.Encode(f, img)
	if err != nil {
		return fmt.Errorf("failed to create %s (%v)", file, err)
	}

	return nil
}

func Logs(feeds, lines []string, templates map[string]int) error {
	_, err := createAllIfNotExist()
	if err != nil {
		return fmt.Errorf("save: failed to create directory: %s: %v", Directory, err)
	}

	dir := filepath.Join(exe.Directory(), saved, Directory, log)

	f, err := os.OpenFile(dir, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0644)
	if err != nil {
		return fmt.Errorf("save: <ini:f:open> log file: %s: %v", Directory, err)
	}
	defer f.Close()

	for _, p := range feeds {
		_, err := fmt.Fprintf(f, "%s\n", p)
		if err != nil {
			return fmt.Errorf("save: failed to write event log: %s: %v", Directory, err)
		}
	}

	for _, line := range lines {
		_, err := fmt.Fprintf(f, "%s\n", line)
		if err != nil {
			return fmt.Errorf("save: failed to append statistic log: %s: %v", Directory, err)
		}
	}

	return saveTemplateStatistics(templates)
}

func Open() error {
	d, err := createAllIfNotExist()
	if err != nil {
		return fmt.Errorf("save: failed to create: %s: %v", Directory, err)
	}
	return open.Run(d)
}

func OpenCurrentLog() error {
	d, err := createAllIfNotExist()
	if err != nil {
		return fmt.Errorf("save: failed to create: %s: %v", Directory, err)
	}
	return open.Run(fmt.Sprintf("%s/%s/%s", d, Directory, log))
}

func OpenImage(name string) error {
	if !strings.HasSuffix(name, ".png") {
		name = fmt.Sprintf("%s.png", name)
	}

	path := filepath.Join(exe.Directory(), saved, ImageDirectory, name)

	p := strings.Split(path, `\`)
	for i := 1; i < len(p); i++ {
		err := createDirIfNotExist(strings.Join(p[0:i], "/"))
		if err != nil {
			return err
		}
	}

	argv, err := syscall.UTF16PtrFromString(os.Getenv("windir") + "\\system32\\cmd.exe /C " + fmt.Sprintf("\"%s\"", path))
	if err != nil {
		return fmt.Errorf("<ini:f:open> %s (%v)", path, err)
	}

	var sI syscall.StartupInfo
	var pI syscall.ProcessInformation

	err = syscall.CreateProcess(nil, argv, nil, nil, true, win32.CreateProcessFlags.NoWindow, nil, nil, &sI, &pI)
	if err != nil {
		return fmt.Errorf("<ini:f:open> %s (%v)", path, err)
	}

	return nil
}

func OpenLogDirectory() error {
	d, err := createAllIfNotExist()
	if err != nil {
		return fmt.Errorf("save: failed to create: %s: %v", Directory, err)
	}
	return open.Run(fmt.Sprintf("%s/%s", d, Directory))
}

func PNG(img image.Image, name string) error {
	if !strings.HasSuffix(name, ".png") {
		name = fmt.Sprintf("%s.png", name)
	}

	path := name
	if !strings.HasPrefix(name, "./") {
		path = filepath.Join(exe.Directory(), saved, ImageDirectory, name)

		p := strings.Split(path, `\`)
		for i := 1; i < len(p); i++ {
			err := createDirIfNotExist(strings.Join(p[0:i], "/"))
			if err != nil {
				return err
			}
		}
	}

	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	return png.Encode(f, img)
}

func KitchenTime() string {
	return time.Now().Format("15_04_05")
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
		err := createDirIfNotExist(filepath.Join(exe.Directory(), saved, Directory, subdir))
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
	dir := filepath.Join(exe.Directory(), saved)

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

func saveTemplateStatistics(t map[string]int) error {
	// Append and save statistics from today.
	today := filepath.Join(exe.Directory(), saved, Directory, templates)

	raw, err := os.ReadFile(today)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("save: <ini:f:open> %s: %v", templates, err)
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
		return fmt.Errorf("save: <ini:f:save> %s: %v", today, err)
	}

	// Append and save statistics from all time.
	all := filepath.Join(exe.Directory(), saved, templates)

	raw, err = os.ReadFile(all)
	if err != nil {
		if !os.IsNotExist(err) {
			return fmt.Errorf("save: <ini:f:open> %s: %v", templates, err)
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
		return fmt.Errorf("save: <ini:f:save> %s: %v", all, err)
	}

	return nil
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
