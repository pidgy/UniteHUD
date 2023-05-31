package notify

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/pidgy/unitehud/nrgba"
)

var (
	Preview     image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	OrangeScore image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	PurpleScore image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	SelfScore   image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	Energy      image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	Time        image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
)

type Post struct {
	nrgba.NRGBA
	time.Time

	msg    string
	orig   string
	count  int
	dedup  bool
	unique bool
}

type notify struct {
	logs []Post
}

var feed = &notify{}

func Announce(format string, a ...interface{}) {
	feed.log(nrgba.Announce, true, false, false, format, a...)
}

func Append(c nrgba.NRGBA, format string, a ...interface{}) {
	feed.log(c, false, false, false, format, a...)
}

func Bool(b bool, format string, a ...interface{}) {
	feed.log(nrgba.Bool(b), false, false, false, format, a...)
}

func Clear() {
	OrangeScore = nil
	PurpleScore = nil
	SelfScore = nil
	Energy = nil
	Time = nil
}

func CLS() {
	feed.logs = []Post{}
}

func Dedup(r nrgba.NRGBA, format string, a ...interface{}) {
	feed.log(r, true, true, false, format, a...)
}

func Denounce(format string, a ...interface{}) {
	feed.log(nrgba.Denounce, true, false, false, fmt.Sprintf("%s", format), a...)
}

func Error(format string, a ...interface{}) {
	feed.log(nrgba.DarkRed, true, false, false, fmt.Sprintf("%s", format), a...)
}

func Feeds() []Post {
	return feed.logs
}

func Feed(r nrgba.NRGBA, format string, a ...interface{}) {
	feed.log(r, true, false, false, format, a...)
}

func LastSystem() string {
	for i := len(feed.logs) - 1; i >= 0; i-- {
		if feed.logs[i].NRGBA == nrgba.System {
			return feed.logs[i].orig
		}
	}
	return "..."
}

func (p *Post) String() string {
	if p.count > 1 {
		return fmt.Sprintf("%s (x%d)", p.msg, p.count)
	}

	return p.msg
}

func Remove(r string) {
	logs := []Post{}
	for _, post := range feed.logs {
		if strings.Contains(post.orig, r) {
			continue
		}

		logs = append(logs, post)
	}
	feed.logs = logs
}

func System(format string, a ...interface{}) {
	feed.log(nrgba.System, true, false, true, fmt.Sprintf("%s", format), a...)
}

func SystemAppend(format string, a ...interface{}) {
	feed.log(nrgba.System, false, false, false, format, a...)
}

func SystemWarn(format string, a ...interface{}) {
	feed.log(nrgba.Pinkity, true, false, false, fmt.Sprintf("%s", format), a...)
}

func Unique(c nrgba.NRGBA, format string, a ...interface{}) {
	feed.log(c, true, false, true, fmt.Sprintf("%s", format), a...)
}

func Warn(format string, a ...interface{}) {
	feed.log(nrgba.Pinkity, true, false, false, format, a...)
}

func (n *notify) log(r nrgba.NRGBA, clock, dedup, unique bool, format string, a ...interface{}) {
	p := Post{
		NRGBA: r,
		Time:  time.Now(),

		orig:   fmt.Sprintf(format, a...),
		count:  1,
		dedup:  dedup,
		unique: unique,
	}

	if clock {
		h, m, s := p.Time.Clock()
		p.msg = fmt.Sprintf("[%02d:%02d:%02d] %s", h, m, s, p.orig)
	} else {
		p.msg = p.orig
	}

	walked := 0
	for i := len(n.logs) - 1; i >= 0; i-- {
		if walked > 3 {
			break
		}
		walked++

		// Dont consolidate score updates.
		if strings.Contains(p.msg, "+") || unique {
			break
		}

		p1s := strings.SplitAfter(p.orig, "]")
		p2s := strings.SplitAfter(n.logs[i].orig, "]")
		p1 := p1s[len(p1s)-1]
		p2 := p2s[len(p2s)-1]
		if p1 == p2 {
			if dedup {
				n.logs[i].count = 1
			} else {
				n.logs[i].count++
			}

			return
		}
	}

	n.logs = append(n.logs, p)
	if len(n.logs) > 10000 {
		n.logs = n.logs[1:]
	}
}
