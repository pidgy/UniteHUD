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
	Msg   string
	Color color.RGBA
}

var feed = &notify{}

func Feeds() []Post {
	return feed.logs
}

func Feed(c color.RGBA, format string, a ...interface{}) {
	feed.log(c, format, a...)
}

func (n *notify) log(c color.RGBA, format string, a ...interface{}) {
	txt := fmt.Sprintf(format, a...)

	n.logs = append(n.logs, Post{
		Msg:   fmt.Sprintf("[%s] %s", time.Now().Format(time.Kitchen), txt),
		Color: c,
	})
	if len(n.logs) > 37 {
		n.logs = n.logs[len(n.logs)-38:]
	}
}
