package process

import (
	"fmt"
	"os"
	"path"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"golang.org/x/sys/windows"

	"github.com/pidgy/unitehud/core/notify"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/system/wapi"
)

type part struct {
	value float64
	label string

	float       float64
	prev, usage int64
}

type process struct {
	ID       int
	ParentID int
	Exe      string
}

type stats struct {
	CPU part
	RAM part
}

var (
	Usage = stats{CPU: part{value: 0, label: "CPU 0%"}, RAM: part{value: 0, label: "RAM 0MB"}}
)

func init() {}

func Open() error {
	err := replace()
	if err != nil {
		return err
	}

	h, err := syscall.GetCurrentProcess()
	if err != nil {
		return err
	}

	go poll(h)

	return nil
}

func (p *part) String() string {
	return p.label
}

func (p *part) Float64() float64 {
	return p.value
}

func Uptime() string {
	u := time.Time{}.Add(time.Since(exe.Uptime))
	return fmt.Sprintf("%02d:%02d:%02d", u.Hour(), u.Minute(), u.Second())
}

func all() ([]process, error) {
	handle, err := windows.CreateToolhelp32Snapshot(wapi.CreateToolhelp32SnapshotFlags.SnapProcess, 0)
	if err != nil {
		return nil, err
	}
	defer windows.CloseHandle(handle)

	var entry windows.ProcessEntry32
	entry.Size = uint32(unsafe.Sizeof(entry))
	// get the first process
	err = windows.Process32First(handle, &entry)
	if err != nil {
		return nil, err
	}

	results := make([]process, 0, 50)
	for {
		results = append(results, from(&entry))

		err = windows.Process32Next(handle, &entry)
		if err != nil {
			// windows sends ERROR_NO_MORE_FILES on last process
			if err == syscall.ERROR_NO_MORE_FILES {
				return results, nil
			}
			return nil, err
		}
	}
}

func from(e *windows.ProcessEntry32) process {
	// Find when the string ends for decoding
	end := 0
	for {
		if e.ExeFile[end] == 0 {
			break
		}
		end++
	}

	return process{
		ID:       int(e.ProcessID),
		ParentID: int(e.ParentProcessID),
		Exe:      syscall.UTF16ToString(e.ExeFile[:end]),
	}
}

func kill(exe string) error {
	ps, err := all()
	if err != nil {
		return err
	}

	this := os.Getpid()

	for _, p := range ps {
		if strings.EqualFold(p.Exe, exe) && p.ID != this {
			p, err := os.FindProcess(p.ID)
			if err != nil {
				return err
			}

			return p.Kill()
		}
	}

	return nil
}

func (p *part) cpu(h syscall.Handle) {
	var ctime, etime, ktime, utime syscall.Filetime

	if p.float == 0 {
		p.float = float64(runtime.NumCPU()) - 2
	}

	err := syscall.GetProcessTimes(h, &ctime, &etime, &ktime, &utime)
	if err != nil {
		notify.Error("[Process] Failed to poll process statistics (%v)", err)
	}

	now := time.Now().UnixNano()

	current := ktime.Nanoseconds() + utime.Nanoseconds()
	delta := 100 * float64(current-p.usage) / float64(now-p.prev)

	p.prev = now
	p.usage = current

	v := delta / p.float
	if v > 0 {
		p.value = v
		p.label = fmt.Sprintf("CPU %.1f%s", p.value, "%")
	}
}

func (p *part) ram() {
	memory := runtime.MemStats{}
	runtime.ReadMemStats(&memory)

	v := float64(memory.Sys) / 1024 / 1024
	if v > 1000 {
		p.value = v / 1000
		p.label = fmt.Sprintf("RAM %.1fGB", p.value)
	} else {
		p.value = v
		p.label = fmt.Sprintf("RAM %.1fMB", p.value)
	}
}

func poll(h syscall.Handle) {
	// cpus := float64(runtime.NumCPU()) - 2
	// prev, usage := int64(0), int64(0)

	Usage.CPU.cpu(h)

	num := 0
	for ; ; time.Sleep(time.Second * 2) {
		Usage.CPU.cpu(h)
		Usage.RAM.ram()

		if exe.Debug && runtime.NumGoroutine() != num {
			num = runtime.NumGoroutine()
			notify.Debug("[Process] Hyperthreads: %d", num)
		}
	}
}

func replace() error {
	for _, exe := range []string{"UniteHUD.exe", "UniteHUD_Debug.exe"} {
		err := kill(path.Base(exe))
		if err != nil {
			return err
		}
	}

	return nil
}
