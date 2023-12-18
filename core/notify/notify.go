package notify

import (
	"fmt"
	"image"
	"strings"
	"time"

	"github.com/pidgy/unitehud/core/global"
	"github.com/pidgy/unitehud/core/nrgba"
)

type (
	Post struct {
		nrgba.NRGBA
		time.Time

		msg    string
		orig   string
		count  int
		dedup  bool
		unique bool
	}
)

var (
	Preview     image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	OrangeScore image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	PurpleScore image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	SelfScore   image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	Energy      image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	Time        image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
)

type debugger struct {
	fmt,
	ftl func(format string, v ...interface{})
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
	feed.logs = feed.logs[:0]
}

func Debug(format string, a ...interface{}) {
	if !global.DebugMode {
		return
	}

	feed.log(nrgba.PastelBlue.Alpha(50), true, true, false, format, a...)
}

func Debugger(prefix string) *debugger {
	return &debugger{
		fmt: func(format string, v ...interface{}) { Debug(prefix+" "+format, v...) },
		ftl: func(format string, v ...interface{}) { Debug(prefix+" [Fatal] "+format, v...) },
	}
}

func Dedup(r nrgba.NRGBA, format string, a ...interface{}) {
	feed.log(r, true, true, false, format, a...)
}

func Error(format string, a ...interface{}) {
	feed.log(nrgba.DarkRed, true, false, false, format, a...)
}

func Feeds() []Post {
	return feed.logs
}

func Feed(r nrgba.NRGBA, format string, a ...interface{}) {
	feed.log(r, true, false, false, format, a...)
}

func Iter(i int) (string, int) {
	if len(feed.logs) > i {
		return feed.logs[i].orig, i + 1
	}
	return "", i
}

func Last() Post {
	if len(feed.logs) == 0 {
		return Post{}
	}
	return feed.logs[len(feed.logs)-1]
}

func Missed(event interface{}, window string) {
	Debug("UI: Missed %T event (%s)", event, window)
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
	feed.log(nrgba.White, true, false, true, format, a...)
}

func SystemAppend(format string, a ...interface{}) {
	feed.log(nrgba.System, false, false, false, format, a...)
}

func SystemWarn(format string, a ...interface{}) {
	feed.log(nrgba.Pinkity, true, false, false, format, a...)
}

func Unique(c nrgba.NRGBA, format string, a ...interface{}) {
	feed.log(c, true, false, true, format, a...)
}

func Warn(format string, a ...interface{}) {
	feed.log(nrgba.Pinkity, true, false, false, format, a...)
}

func (d *debugger) Fatal(v ...interface{})                 {} //d.ftl("%s", fmt.Sprint(v...)) }
func (d *debugger) Fatalf(format string, v ...interface{}) {} //d.ftl(format, v...) }
func (d *debugger) Print(v ...interface{})                 {} //d.fmt("%s", fmt.Sprint(v...)) }
func (d *debugger) Printf(format string, v ...interface{}) {} //d.fmt(format, v...) }

func (n *notify) log(r nrgba.NRGBA, clock, dedup, unique bool, format string, a ...interface{}) {
	p := Post{
		NRGBA: r,
		Time:  time.Now(),

		orig:   fmt.Sprintf(format, a...),
		count:  1,
		dedup:  dedup,
		unique: unique,
	}

	if global.DebugMode {
		fmt.Printf(format+"\n", a...)
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
}
