package device

import (
	"regexp"
)

// Path represents a unique hardware ID, used to identify a Device across multiple categories.
type Path struct {
	Root      string
	Type      string
	VendorID  string
	ProductID string
	Revision  string
	DeviceID  string

	raw string
}

type regexPath struct {
	vid, pid, rev, dev *regexp.Regexp
}

var (
	root = regexp.MustCompile(`(?P<ROOT>.+?)(:?)(?P<TYPE>usb|pci|sw?)(#|:)`)

	pci = regexPath{
		vid: regexp.MustCompile(`(ven_)(?P<VID>.*?)(&|#)`),
		pid: regexp.MustCompile(`(pid_)(?P<PID>.*?)(&|#)`),
		rev: regexp.MustCompile(`(rev_)(?P<REV>.*?)(&|#)`),
		dev: regexp.MustCompile(`(dev_)(?P<DEV>.*?)(&|#)`),
	}

	usb = regexPath{
		vid: regexp.MustCompile(`(vid_)(?P<VID>.*?)(&|#)`),
		pid: regexp.MustCompile(`(pid_)(?P<PID>.*?)(&|#)`),
		rev: regexp.MustCompile(`(rev_)(?P<REV>.*?)(&|#)`),
		dev: regexp.MustCompile(`(dev_)(?P<DEV>.*?)(&|#)`),
	}
)

func NewPath(s string) Path {
	p := Path{
		Root:      "unknown",
		Type:      "unknown",
		VendorID:  "unknown",
		ProductID: "unknown",
		Revision:  "unknown",
		DeviceID:  "unknown",

		raw: s,
	}

	m := root.FindStringSubmatch(p.raw)
	if m != nil {
		p.Root = m[root.SubexpIndex("ROOT")]
		p.Type = m[root.SubexpIndex("TYPE")]
	}

	switch p.Type {
	case "usb":
		p.extract(usb)
	case "pci":
		p.extract(pci)
	case "sw":
	case "unknown":
	}

	return p
}

func (p *Path) extract(r regexPath) {
	m := r.vid.FindStringSubmatch(p.raw)
	if m != nil {
		p.VendorID = m[r.vid.SubexpIndex("VID")]
	}

	m = r.pid.FindStringSubmatch(p.raw)
	if m != nil {
		p.ProductID = m[r.pid.SubexpIndex("PID")]
	}

	m = r.rev.FindStringSubmatch(p.raw)
	if m != nil {
		p.Revision = m[r.rev.SubexpIndex("REV")]
	}

	m = r.dev.FindStringSubmatch(p.raw)
	if m != nil {
		p.DeviceID = m[r.dev.SubexpIndex("DEV")]
	}
}
