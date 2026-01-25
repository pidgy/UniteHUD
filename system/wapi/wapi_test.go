package wapi

import (
	"testing"
)

func TestEnumerateWindows(t *testing.T) {
	windows, err := GetAllWindows()
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
	m, err := GetAllMonitors()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Monitors: %d", len(m.Active))
}

func TestGetMonitorInfo(t *testing.T) {
	mi, err := GetMonitorInfo()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("%s", mi)
}

func TestGetMonitorHandleFromIndex(t *testing.T) {
	hwnd, err := GetMonitorHandleFromIndex(0)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("HWND: %d", hwnd)
}

func TestGetMonitorIndexFromMonitorInfo(t *testing.T) {
	mi, err := GetMonitorInfo()
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("MonitorInfo: %s", mi)

	i, err := GetMonitorIndexFromMonitorInfo(mi)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Index: %d", i)
}
