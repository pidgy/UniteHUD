package process

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const. Then remove this comment.

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
	"github.com/pidgy/unitehud/system/win32"
)

type part struct {
	float64 float64
	string  string

	float       float64
	prev, usage int64
}

type process struct {
	ID       int
	ParentID int
	Exe      string
}

type stats struct {
	CPU     part
	RAM     part
	Threads part
}

var (
	Usage = stats{CPU: part{float64: 0, string: "CPU 0%"}, RAM: part{float64: 0, string: "RAM 0MB"}, Threads: part{float64: 0, string: "Routines: 0"}}
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

func (p part) String() string {
	return p.string
}

func (p part) Float64() float64 {
	return p.float64
}

func Uptime() string {
	u := time.Time{}.Add(time.Since(exe.Uptime))
	return fmt.Sprintf("%02d:%02d:%02d", u.Hour(), u.Minute(), u.Second())
}

func all() ([]process, error) {
	handle, err := windows.CreateToolhelp32Snapshot(win32.CreateToolhelp32SnapshotFlags.SnapProcess, 0)
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
		p.float64 = v
		p.string = fmt.Sprintf("CPU: %.1f%s", p.float64, "%")
	}
}

func (p *part) ram() {
	memory := runtime.MemStats{}
	runtime.ReadMemStats(&memory)

	v := float64(memory.Sys) / 1024 / 1024
	if v > 1000 {
		p.float64 = v / 1000
		p.string = fmt.Sprintf("RAM: %.1fGB", p.float64)
	} else {
		p.float64 = v
		p.string = fmt.Sprintf("RAM: %.1fMB", p.float64)
	}
}

func (p *part) threads() {
	if n := float64(runtime.NumGoroutine()); n != p.float64 {
		p.float64 = n
		p.string = fmt.Sprintf("Threads: %.0fε", p.float64)
		notify.Debug("[Process] %s", p.string)
	}
}

func poll(h syscall.Handle) {
	Usage.CPU.cpu(h)

	for ; ; time.Sleep(time.Second * 2) {
		Usage.CPU.cpu(h)
		Usage.RAM.ram()
		Usage.Threads.threads()
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
