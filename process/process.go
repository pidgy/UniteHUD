package process

import (
	"os"
	"path"
	"strings"
	"syscall"
	"unsafe"

	"golang.org/x/sys/windows"
)

const TH32CS_SNAPPROCESS = 0x00000002

type Process struct {
	ID       int
	ParentID int
	Exe      string
}

func Replace() error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}

	args := strings.Split(exe, "\\")
	exe = args[len(args)-1]

	err = kill(path.Base(exe))
	if err != nil {
		return err
	}

	return nil
}

func kill(exe string) error {
	ps, err := all()
	if err != nil {
		return err
	}

	this := os.Getpid()

	for _, p := range ps {
		if strings.ToLower(p.Exe) == strings.ToLower(exe) && p.ID != this {
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
	handle, err := windows.CreateToolhelp32Snapshot(TH32CS_SNAPPROCESS, 0)
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
