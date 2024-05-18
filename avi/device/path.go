package device

import (
	"regexp"
)

// Path represents a unique string that can be used to identify a Device.
type Path struct {
	Root      string
	Type      string
	VendorID  string
	ProductID string
	Revision  string
	DeviceID  string

	raw string
}

/*
USB\VID_v(4)&PID_d(4)&REV_r(4)
Where:

v(4) is the 4-digit vendor code that the USB committee assigns to the vendor.
d(4) is the 4-digit product code that the vendor assigns to the device.
r(4) is the revision code.
*/

type types struct {
	usb, pci *regexp.Regexp
}

var (
	reg = struct {
		root               *regexp.Regexp
		vid, pid, rev, dev types
	}{
		root: regexp.MustCompile(`(?P<ROOT>.+?)(:?)(?P<TYPE>usb|pci|sw?)(#|:)`),

		vid: types{
			usb: regexp.MustCompile(`(vid_)(?P<VID>.*?)(&|#)`),
			pci: regexp.MustCompile(`(ven_)(?P<VID>.*?)(&|#)`),
		},
		pid: types{
			usb: regexp.MustCompile(`(pid_)(?P<PID>.*?)(&|#)`),
			pci: regexp.MustCompile(`(pid_)(?P<PID>.*?)(&|#)`),
		},
		rev: types{
			usb: regexp.MustCompile(`(rev_)(?P<REV>.*?)(&|#)`),
			pci: regexp.MustCompile(`(rev_)(?P<REV>.*?)(&|#)`),
		},
		dev: types{
			usb: regexp.MustCompile(`(dev_)(?P<DEV>.*?)(&|#)`),
			pci: regexp.MustCompile(`(dev_)(?P<DEV>.*?)(&|#)`),
		},
	}
)

func NewPath(r string) Path {
	p := Path{
		Root:      "unknown",
		Type:      "unknown",
		VendorID:  "unknown",
		ProductID: "unknown",
		Revision:  "unknown",
		DeviceID:  "unknown",

		raw: r,
	}

	m := reg.root.FindStringSubmatch(p.raw)
	if m != nil {
		p.Root = m[reg.root.SubexpIndex("ROOT")]
		p.Type = m[reg.root.SubexpIndex("TYPE")]
	}

	switch p.Type {
	case "usb":
		m = reg.vid.usb.FindStringSubmatch(p.raw)
		if m != nil {
			p.VendorID = m[reg.vid.usb.SubexpIndex("VID")]
		}
		m = reg.pid.usb.FindStringSubmatch(p.raw)
		if m != nil {
			p.ProductID = m[reg.pid.usb.SubexpIndex("PID")]
		}
		m = reg.rev.usb.FindStringSubmatch(p.raw)
		if m != nil {
			p.Revision = m[reg.rev.usb.SubexpIndex("REV")]
		}
		m = reg.dev.usb.FindStringSubmatch(p.raw)
		if m != nil {
			p.DeviceID = m[reg.dev.usb.SubexpIndex("DEV")]
		}
	case "pci":
		m = reg.vid.pci.FindStringSubmatch(p.raw)
		if m != nil {
			p.VendorID = m[reg.vid.pci.SubexpIndex("VID")]
		}
		m = reg.pid.pci.FindStringSubmatch(p.raw)
		if m != nil {
			p.ProductID = m[reg.pid.pci.SubexpIndex("PID")]
		}
		m = reg.rev.pci.FindStringSubmatch(p.raw)
		if m != nil {
			p.Revision = m[reg.rev.pci.SubexpIndex("REV")]
		}
		m = reg.dev.pci.FindStringSubmatch(p.raw)
		if m != nil {
			p.DeviceID = m[reg.dev.pci.SubexpIndex("DEV")]
		}
	}

	return p
}
