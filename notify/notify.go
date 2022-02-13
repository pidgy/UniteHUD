package notify

import (
	"fmt"
	"image/color"
	"time"
)

type notify struct {
	logs []Post
}

type Post struct {
	color.RGBA
	msg   string
	orig  string
	count int
}

var feed = &notify{}

func Feeds() []Post {
	return feed.logs
}

func Feed(c color.RGBA, format string, a ...interface{}) {
	feed.log(c, format, a...)
}

func (n *notify) log(c color.RGBA, format string, a ...interface{}) {
	p := Post{
		RGBA:  c,
		orig:  fmt.Sprintf(format, a...),
		count: 1,
	}

	p.msg = fmt.Sprintf("[%s] %s", time.Now().Format(time.Kitchen), p.orig)

	if len(n.logs) > 0 {
		if p.orig == n.logs[len(n.logs)-1].orig {
			n.logs[len(n.logs)-1].count++
			return
		}
	}

	n.logs = append(n.logs, p)
	if len(n.logs) > 37 {
		n.logs = n.logs[len(n.logs)-38:]
	}
}

func (p Post) String() string {
	if p.count > 1 {
		return fmt.Sprintf("%s (x%d)", p.msg, p.count)
	}

	return p.msg
}
