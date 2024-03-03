package process

import (
	"os"
	"path"
	"runtime"
	"strings"
	"syscall"
	"time"
	"unsafe"

	"github.com/pidgy/unitehud/core/global"
	"golang.org/x/sys/windows"
)

const TH32CSSnapProcess = 0x00000002

type Process struct {
	ID       int
	ParentID int
	Exe      string
}

var (
	handle syscall.Handle

	ctime, etime, ktime, utime syscall.Filetime
	prev, usage                = ctime.Nanoseconds(), ktime.Nanoseconds() + utime.Nanoseconds()
	cpus                       = float64(runtime.NumCPU()) - 2

	memory runtime.MemStats
)

func CPU() (float64, error) {
	err := syscall.GetProcessTimes(handle, &ctime, &etime, &ktime, &utime)
	if err != nil {
		return 0.0, err
	}

	now := time.Now().UnixNano()
	diff := now - prev

	current := ktime.Nanoseconds() + utime.Nanoseconds()
	diff2 := current - usage

	prev = now
	usage = current

	return (100 * float64(diff2) / float64(diff)) / cpus, nil
}

func RAM() float64 {
	runtime.ReadMemStats(&memory)
	return (float64(memory.Sys) / 1024 / 1024)
}

func Start() error {
	err := replace()
	if err != nil {
		return err
	}

	handle, err = syscall.GetCurrentProcess()
	if err != nil {
		return err
	}

	return nil
}

func Uptime() time.Time {
	return time.Time{}.Add(time.Since(global.Uptime))
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

func replace() error {
	for _, exe := range []string{"UniteHUD.exe", "UniteHUD_Debug.exe"} {
		err := kill(path.Base(exe))
		if err != nil {
			return err
		}
	}

	return nil
}
