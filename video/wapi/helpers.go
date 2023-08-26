package wapi

import "image"

func SetWindowPosNone(hwnd uintptr, pt image.Point, size image.Point) {
	helpSetWindowPos(hwnd, pt, size, SetWindowPosFlags.None)
}

func SetWindowPosNoSize(hwnd uintptr, pt image.Point) {
	helpSetWindowPos(hwnd, pt, image.Pt(0, 0), SetWindowPosFlags.NoSize)
}

func SetWindowPosShow(hwnd uintptr, pt image.Point, size image.Point) {
	helpSetWindowPos(hwnd, pt, size, SetWindowPosFlags.ShowWindow)
}

func helpSetWindowPos(hwnd uintptr, pt image.Point, size image.Point, flags uintptr) {
	go SetWindowPos.Call(
		hwnd,
		uintptr(0),
		uintptr(pt.X),
		uintptr(pt.Y),
		uintptr(size.X),
		uintptr(size.Y),
		flags,
	)
}
