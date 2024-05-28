package iface

import (
	"fmt"
	"syscall"
	"unsafe"

	"github.com/rodrigocfd/windigo/win"
	"github.com/rodrigocfd/windigo/win/co"
	"github.com/rodrigocfd/windigo/win/com/com"
	"github.com/rodrigocfd/windigo/win/com/com/comco"
	"github.com/rodrigocfd/windigo/win/com/com/comvt"
	"github.com/rodrigocfd/windigo/win/errco"
)

var (
	CLSID = struct {
		AudioInputDeviceCategory,
		VideoInputDeviceCategory,
		SystemDeviceEnum co.CLSID
	}{
		AudioInputDeviceCategory: co.CLSID("33D9A762-90C8-11d0-BD43-00A0C911CE86"),
		VideoInputDeviceCategory: co.CLSID("860BB310-5D01-11d0-BD3B-00A0C911CE86"),
		SystemDeviceEnum:         co.CLSID("62BE5D10-60EB-11d0-BD3B-00A0C911CE86"),
	}

	IID = struct {
		ICreateDevEnum co.IID
		IEnumMoniker   co.IID
		IPropertyBag   co.IID
	}{
		ICreateDevEnum: co.IID("29840822-5B84-11D0-BD3B-00A0C911CE86"),
		IEnumMoniker:   co.IID("0000000f-0000-0000-C000-000000000046"),
		IPropertyBag:   co.IID("55272A00-42CB-11CE-8135-00AA004BB851"),
	}
)

/*

func (me *_IPicture) Attributes() comco.PICATTR {
	var attr comco.PICATTR
	ret, _, _ := syscall.SyscallN(
		(*comvt.IPicture)(unsafe.Pointer(*me.Ptr())).Get_Attributes,
		uintptr(unsafe.Pointer(me.Ptr())),
		uintptr(unsafe.Pointer(&attr)))

	if hr := errco.ERROR(ret); hr == errco.S_OK {
		return attr
	} else {
		panic(hr)
	}
}

*/
// Constructs a COM object from the base IUnknown.
//
// ⚠️ You must defer IPicture.Release().
// func NewIPicture(base IUnknown) IPicture {
// 	return &_IPicture{IUnknown: base}
// }
type ICreateDevEnum struct {
	com.IUnknown
}

type IEnumMoniker struct {
	com.IUnknown
}

type IMoniker struct {
	com.IUnknown
}

type IPropertyBag struct {
	com.IUnknown
}

// ICreateDevEnum : public IUnknown
// {
// public:
//
//		CreateClassEnumerator(REFCLSID clsidDeviceClass, IEnumMoniker **ppEnumMoniker, DWORD dwFlags) = 0;
//	};
func NewICreateDevEnum() (*ICreateDevEnum, error) {
	unk := com.CoCreateInstance(CLSID.SystemDeviceEnum, nil, comco.CLSCTX_INPROC_SERVER, IID.ICreateDevEnum)
	if !com.IsObj(unk) {
		return nil, fmt.Errorf("com object: CoCreateInstance(SystemDeviceEnum, CLSCTX_INPROC_SERVER, ICreateDevEnum)")
	}

	enum := unk.QueryInterface(IID.ICreateDevEnum)
	if !com.IsObj(unk) {
		return nil, fmt.Errorf("com object: QueryInterface(ICreateDevEnum)")
	}

	return &ICreateDevEnum{enum}, nil
}

func (me *ICreateDevEnum) CreateClassEnumerator(class co.CLSID) (*IEnumMoniker, error) {
	type vt struct {
		comvt.IUnknown
		CreateClassEnumerator uintptr
	}
	var out **comvt.IUnknown

	ret, _, _ := syscall.SyscallN(
		(*vt)(unsafe.Pointer(*me.Ptr())).CreateClassEnumerator, uintptr(unsafe.Pointer(me.Ptr())),
		uintptr(unsafe.Pointer(win.GuidFromClsid(class))),
		uintptr(unsafe.Pointer(&out)),
		0,
	)

	hr := errco.ERROR(ret)
	if hr != errco.S_OK {
		return nil, fmt.Errorf("ICreateDevEnum.CreateClassEnumerator: %v", hr)
	}

	return &IEnumMoniker{IUnknown: com.NewIUnknown(out)}, nil
}

func (me *IEnumMoniker) Next() (*IMoniker, bool) {
	type vt struct {
		comvt.IUnknown
		Next  uintptr
		Skip  uintptr
		Reset uintptr
		Clone uintptr
	}

	m := &IMoniker{}

	ret, _, _ := syscall.SyscallN(
		(*vt)(unsafe.Pointer(*me.Ptr())).Next, uintptr(unsafe.Pointer(me.Ptr())),
		1,
		uintptr(unsafe.Pointer(&m)),
		0,
	)

	hr := errco.ERROR(ret)
	if hr != errco.S_OK {
		return nil, false
	}

	return m, true
}

func (me *IMoniker) BindToStorage() (*IPropertyBag, error) {
	type vt struct {
		BindToObject        uintptr
		BindToStorage       uintptr
		Reduce              uintptr
		ComposeWith         uintptr
		Enum                uintptr
		IsEqual             uintptr
		Hash                uintptr
		IsRunning           uintptr
		GetTimeOfLastChange uintptr
		Inverse             uintptr
		CommonPrefixWith    uintptr
		RelativePathTo      uintptr
		GetDisplayName      uintptr
		ParseDisplayName    uintptr
		IsSystemMoniker     uintptr
	}

	b := &IPropertyBag{}

	ret, _, _ := syscall.SyscallN(
		(*vt)(unsafe.Pointer(*me.Ptr())).BindToStorage, uintptr(unsafe.Pointer(me.Ptr())),
		0,
		0,
		// uintptr(unsafe.Pointer(&b)),
		uintptr(unsafe.Pointer(win.GuidFromIid(co.IID(IID.IPropertyBag)))),
		uintptr(unsafe.Pointer(&b)),
	)

	hr := errco.ERROR(ret)
	if hr != errco.S_OK {
		return nil, fmt.Errorf("IMoniker.BindToStorage: %v", hr)
	}

	return b, nil
}
