package win32

import (
	"testing"
)

func TestEnumerateWindows(t *testing.T) {
	windows, err := FindWindows()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Windows: %d", len(windows.Infos))

	for _, i := range windows.Infos {
		if i.Style&WindowStyles.Visible == WindowStyles.Visible {
			t.Logf("%s: style: %d, status: visible=%t max=%t overlapped=%t visible=(%d)%t",
				i.Title, i.Style, i.Style.Visible(), i.Style.Maximized(), i.Style.OverlappedWindow(), i.Status, i.Status.Visible())
		}
	}
}

func TestEnumerateDisplayMonitors(t *testing.T) {
	m, err := FindMonitors()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Monitors: %d", len(m.Active))
}

func TestGetMonitorInfo(t *testing.T) {
	mi, err := NewMonitorInfo()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", mi)
}

func TestGetMonitorHandleFromIndex(t *testing.T) {
	hwnd, err := MonitorHandleFromIndex(0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("HWND: %d", hwnd)
}

func TestGetMonitorIndexFromMonitorInfo(t *testing.T) {
	m, err := NewMonitorInfo()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("MonitorInfo: %s", m)

	i, err := m.Index()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Index: %d", i)
}
