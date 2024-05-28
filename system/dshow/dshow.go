package main

import (
	"fmt"

	"github.com/rodrigocfd/windigo/win/com/com"
	"github.com/rodrigocfd/windigo/win/com/com/comco"

	"dshow/iface"
)

func main() {
	com.CoInitializeEx(comco.COINIT_APARTMENTTHREADED)
	defer com.CoUninitialize()

	c, err := iface.NewICreateDevEnum()
	if err != nil {
		panic(err)
	}
	defer c.Release()

	e, err := c.CreateClassEnumerator(iface.CLSID.VideoInputDeviceCategory)
	if err != nil {
		panic(err)
	}
	defer e.Release()

	m, ok := e.Next()
	if !ok {
		panic("failed to call IEnumMoniker.Next")
	}
	defer m.Release()

	b, err := m.BindToStorage()
	if err != nil {
		panic(err)
	}
	defer b.Release()

	// var moniker *IEnumMoniker

	// icde := newICreateDevEnum(enum)
	// icde.CreateClassEnumerator(clsidVideoInputDeviceCategory, &moniker)

	// filters := dshow.NewIEnumFilters(enum)
	// defer filters.Release()

	// _, ok := filters.Next()

	fmt.Printf("ok\n")
}
