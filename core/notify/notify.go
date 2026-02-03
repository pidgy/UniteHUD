package notify

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

import (
	"fmt"
	"image"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/pidgy/unitehud/core/rgba/nrgba"
	"github.com/pidgy/unitehud/exe"
	"github.com/pidgy/unitehud/system/ini"
)

type (
	// Post represents a single notification entry.
	Post struct {
		nrgba.NRGBA
		time.Time
		Hidden bool

		msg   string
		orig  string
		count int
	}
)

var (
	// Score/Event captured preview images rendered by the UI.
	OrangeScore image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	PurpleScore image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	SelfScore   image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	Energy      image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	Time        image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))

	preview           image.Image = image.NewRGBA(image.Rect(0, 0, 0, 0))
	previewResolution             = "0x0"

	// Disabled controls which classes of notifications are suppressed.
	Disabled struct {
		Errors      bool
		Warnings    bool
		Info        bool
		System      bool
		Debug       bool
		DebugColors bool
	}

	colorError  = nrgba.Pinkity
	colorWarn   = nrgba.PastelCoral
	colorSystem = nrgba.System
	colorDebug  = nrgba.PastelBlue.Alpha(50)

	feed = &notify{}
)

// SetPreview stores the latest preview image and its resolution.
func SetPreview(img image.Image) {
	preview = img
	previewResolution = fmt.Sprintf("%dx%d", img.Bounds().Dx(), img.Bounds().Dy())
}

// Preview returns the current preview image.
func Preview() image.Image {
	return preview
}

// PreviewResolutionString returns the preview image resolution as "WxH".
func PreviewResolutionString() string {
	return previewResolution
}

// notify holds the in-memory notification feed.
type notify struct {
	logs []Post
}

// Announce logs a system announcement message.
func Announce(format string, a ...any) {
	feed.log(colorSystem, true, false, format, a...)
}

// Append logs a message without a timestamp prefix.
func Append(c nrgba.NRGBA, format string, a ...any) {
	feed.log(c, false, false, format, a...)
}

// Bool logs a boolean-colored message.
func Bool(b bool, format string, a ...any) {
	feed.log(nrgba.Bool(b), false, false, format, a...)
}

// CLS clears all stored feed entries.
func CLS() {
	feed.logs = feed.logs[:0]
}

// Clear resets cached score images.
func Clear() {
	OrangeScore = nil
	PurpleScore = nil
	SelfScore = nil
	Energy = nil
	Time = nil
}

// Debug logs a debug message when debug mode is enabled.
func Debug(format string, a ...any) {
	if !exe.Debug {
		return
	}

	feed.log(colorDebug, true, false, format, a...)
}

// Error logs an error message.
func Error(format string, a ...any) {
	feed.log(colorError, true, false, format, a...)
}

// Feed logs a message using the specified color.
func Feed(color nrgba.NRGBA, format string, a ...any) {
	feed.log(color, true, false, format, a...)
}

// FeedReplace logs a message and hides a recent matching entry.
func FeedReplace(color nrgba.NRGBA, r *regexp.Regexp, format string, a ...any) {
	defer Feed(color, format, a...)

	max := 20
	for i := len(feed.logs) - 1; i >= 0 && max >= 0; i-- {
		if r.MatchString(feed.logs[i].orig) {
			feed.logs[i].Hidden = true
			return
		}
		max--
	}
}

// FeedStrings returns the feed as string messages.
func FeedStrings() (s []string) {
	for _, p := range Feeds() {
		s = append(s, p.String())
	}
	return
}

// FeedUnique logs a unique message without consolidation.
func FeedUnique(color nrgba.NRGBA, format string, a ...any) {
	feed.log(color, true, true, format, a...)
}

// Feeds returns the filtered feed based on current disabled flags.
func Feeds() []Post {
	if !Disabled.Errors && !Disabled.Warnings && !Disabled.Info && !Disabled.System {
		if exe.Debug && !Disabled.Debug {
			return feed.logs
		}
	}

	p := []Post{}
	for _, l := range feed.logs {
		switch {
		case l.NRGBA.Eq(colorError):
			if !Disabled.Errors {
				p = append(p, l)
			}
		case l.NRGBA.Eq(colorWarn):
			if !Disabled.Warnings {
				p = append(p, l)
			}
		case l.NRGBA.Eq(colorSystem):
			if !Disabled.System {
				p = append(p, l)
			}
		case l.NRGBA.Eq(colorDebug):
			if !Disabled.Debug {
				p = append(p, l)
			}
		default:
			if !Disabled.Info {
				p = append(p, l)
			}
		}
	}

	return p
}

// Iter returns the next feed message and updated index.
func Iter(i int) (string, int) {
	if len(feed.logs) > i {
		return feed.logs[i].orig, i + 1
	}
	return "", i
}

// Last returns the most recent feed entry.
func Last() Post {
	if len(feed.logs) == 0 {
		return Post{}
	}
	return feed.logs[len(feed.logs)-1]
}

// LastNStrings returns up to n most recent non-duplicate messages.
func LastNStrings(n int) (s []string) {
	for i := len(feed.logs) - 1; i >= 0 && i > len(feed.logs)-1-n; i-- {
		if feed.logs[i].count > 1 {
			continue
		}
		s = append(s, feed.logs[i].msg)
	}
	return
}

// Missed logs a missed event for a given window.
func Missed(event any, window string) {
	Debug("[UI] Missed %T event (%s)", event, window)
}

// String formats the post message with a count suffix if repeated.
func (p *Post) String() string {
	if p.count > 1 {
		return fmt.Sprintf("%s (x%d)", p.msg, p.count)
	}
	return p.msg
}

// Replace logs a message and removes older entries with the same prefix.
func Replace(prefix string, log func(format string, a ...any), format string, a ...any) {
	log(format, a...)

	n := 0
	for i := len(feed.logs) - 1; i >= 0; i-- {
		p := feed.logs[i]
		if strings.HasPrefix(p.orig, prefix) {
			n++
		}
		if n == 2 {
			feed.logs = append(feed.logs[:i], feed.logs[i+1:]...)
			return
		}
	}
}

// System logs a unique system message.
func System(format string, a ...any) {
	feed.log(colorSystem, true, true, format, a...)
}

// SystemAppend logs a system message without a timestamp prefix.
func SystemAppend(format string, a ...any) {
	feed.log(colorSystem.Alpha(200), false, false, format, a...)
}

// Unique logs a unique message using the provided color.
func Unique(c nrgba.NRGBA, format string, a ...any) {
	feed.log(c, true, true, format, a...)
}

// Warn logs a warning message.
func Warn(format string, a ...any) {
	feed.log(colorWarn, true, false, format, a...)
}

// log formats and stores a feed entry, consolidating repeats when allowed.
func (n *notify) log(r nrgba.NRGBA, clock, unique bool, format string, a ...any) {
	msg := ini.Format(fmt.Sprintf(format, a...))

	p := Post{
		NRGBA: r,
		Time:  time.Now(),

		orig:  msg,
		count: 1,
	}

	if exe.Debug {
		msg := msg
		if !Disabled.DebugColors {
			msg = p.NRGBA.Windows(msg)
		} else {
			msg = msg + "\n"
		}
		fmt.Fprint(os.Stdout, msg)
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
			// if dedup {
			// 	n.logs[i].count = 1
			// } else {
			n.logs[i].count++
			// }

			return
		}
	}

	n.logs = append(n.logs, p)
}
