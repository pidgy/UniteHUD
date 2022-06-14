package notify

import (
	"fmt"
	"image"
	"image/color"
	"time"

	"github.com/pidgy/unitehud/config"
	"github.com/pidgy/unitehud/dev"
)

var (
	Preview     image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	OrangeScore image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	PurpleScore image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	SelfScore   image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	Balls       image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	Time        image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
)

type Post struct {
	color.RGBA
	msg   string
	orig  string
	count int
}

type notify struct {
	logs []Post
}

var feed = &notify{}

func Append(c color.RGBA, format string, a ...interface{}) {
	feed.log(c, false, format, a...)
}

func Clear() {
	OrangeScore = nil
	PurpleScore = nil
	SelfScore = nil
	Balls = nil
	Time = nil
}

func Feeds() []Post {
	return feed.logs
}

func Feed(c color.RGBA, format string, a ...interface{}) {
	feed.log(c, true, format, a...)
}

func (n *notify) log(c color.RGBA, clock bool, format string, a ...interface{}) {
	p := Post{
		RGBA:  c,
		orig:  fmt.Sprintf(format, a...),
		count: 1,
	}

	if clock {
		p.msg = fmt.Sprintf("[%s] %s", time.Now().Format(time.Kitchen), p.orig)
	} else {
		p.msg = p.orig
	}

	if len(n.logs) > 0 {
		if p.orig == n.logs[len(n.logs)-1].orig {
			n.logs[len(n.logs)-1].count++
			return
		}
	}

	n.logs = append(n.logs, p)

	if config.Current.Record {
		dev.Log(format, a...)
	}
}

func (p Post) String() string {
	if p.count > 1 {
		return fmt.Sprintf("%s (%d)", p.msg, p.count)
	}

	return p.msg
}
