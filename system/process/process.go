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
	"github.com/pidgy/unitehud/global"
)

const TH32CSSnapProcess = 0x00000002

type Process struct {
	ID       int
	ParentID int
	Exe      string
}

type Stat struct {
	value float64
	label string
}

var (
	CPU, RAM = Stat{0, "CPU 0%"}, Stat{0, "RAM 0MB"}
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

func (s *Stat) String() string {
	return s.label
}

func (s *Stat) Float64() float64 {
	return s.value
}

func Uptime() string {
	u := time.Time{}.Add(time.Since(global.Uptime))
	return fmt.Sprintf("%02d:%02d:%02d", u.Hour(), u.Minute(), u.Second())
}

func all() ([]Process, error) {
	handle, err := windows.CreateToolhelp32Snapshot(TH32CSSnapProcess, 0)
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

	results := make([]Process, 0, 50)
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

func from(e *windows.ProcessEntry32) Process {
	// Find when the string ends for decoding
	end := 0
	for {
		if e.ExeFile[end] == 0 {
			break
		}
		end++
	}

	return Process{
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

func poll(h syscall.Handle) {
	cpus := float64(runtime.NumCPU()) - 2
	prev, usage := int64(0), int64(0)

	cpu := func() {
		var ctime, etime, ktime, utime syscall.Filetime

		err := syscall.GetProcessTimes(h, &ctime, &etime, &ktime, &utime)
		if err != nil {
			notify.Error("[Process] Failed to poll process statistics (%v)", err)
			return
		}

		now := time.Now().UnixNano()

		current := ktime.Nanoseconds() + utime.Nanoseconds()
		delta := 100 * float64(current-usage) / float64(now-prev)

		prev = now
		usage = current

		v := delta / cpus
		if v > 0 {
			CPU.value = v
			CPU.label = fmt.Sprintf("CPU %.1f%s", CPU.value, "%")
		}
	}

	ram := func() {
		memory := runtime.MemStats{}
		runtime.ReadMemStats(&memory)

		v := float64(memory.Sys) / 1024 / 1024
		if v > 1000 {
			RAM.value = v / 1000
			RAM.label = fmt.Sprintf("RAM %.1fGB", RAM.value)
		} else {
			RAM.value = v
			RAM.label = fmt.Sprintf("RAM %.1fMB", RAM.value)
		}
	}

	for ; ; time.Sleep(time.Second * 2) {
		cpu()
		ram()
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
