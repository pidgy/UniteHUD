package window

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"strings"
	"time"

	"github.com/pidgy/unitehud/team"
	"gocv.io/x/gocv"
)

type message struct {
	rgba color.RGBA
	txt  string
}

type window struct {
	*gocv.Window
	messageq chan message
	messages []message
	bg       gocv.Mat
}

var (
	Default = color.RGBA{255, 255, 255, 255}
)

var (
	win *window

	point  = image.Pt(10, 84)
	lines  = 19
	purple = ""
	orange = ""
	clock  = ""
	title  = "Pokemon Unite HUD Server"
)

func Init() error {
	f, err := os.Open("img/bg.png")
	if err != nil {
		return nil
	}

	img, err := png.Decode(f)
	if err != nil {
		return nil
	}

	bg, err := gocv.ImageToMatRGBA(img)
	if err != nil {
		return nil
	}

	win = &window{
		Window:   gocv.NewWindow("Pokemon Unite HUD Server"),
		messageq: make(chan message),
		messages: []message{},
		bg:       bg,
	}

	return nil
}

func Close() {
	if win != nil && win.IsOpen() {
		win.Close()
	}
}

func Open(msg string) {
	mat := win.bg.Clone()
	win.ResizeWindow(mat.Cols(), mat.Rows())
	win.IMShow(mat)

	Write(Default, msg)

	for {
		if !win.visible() {
			break
		}

		select {
		case msg := <-win.messageq:
			mat.Close()
			mat = redraw(msg)
			win.IMShow(mat)
		default:
			win.WaitKey(1)
		}
	}

	if win.visible() {
		win.Close()
	}
}

func Score(p, o, s int) {
	purple = fmt.Sprintf("%d/%d", s, p)
	orange = fmt.Sprintf("%d", o)
}

func Time(seconds int) {
	clock = fmt.Sprintf("[%02d:%02d]", seconds/60, seconds%60)
}

func Write(rgba color.RGBA, txt ...string) {
	go func() {
		win.messageq <- message{rgba, "[" + time.Now().Format(time.Kitchen) + "] " + strings.Join(txt, " ")}
	}()
}

func (w *window) chunk(m message) {
	str := ""
	for i := 0; i < len(m.txt); i++ {
		str += string(m.txt[i])

		if i != 0 && i%win.bg.Cols() == 0 {
			win.messages = append(win.messages, message{m.rgba, str})
			str = ""
		}
	}

	if str != "" {
		win.messages = append(win.messages, message{m.rgba, str})
	}
}

func (w *window) line() string {
	line := ""
	for i := 0; i < w.bg.Cols(); i++ {
		line += "-"
	}
	return line
}

func redraw(m message) gocv.Mat {
	win.chunk(m)

	mat := win.bg.Clone()

	gocv.PutTextWithParams(&mat, title, image.Pt(point.X, 21), gocv.FontHersheyPlain, 1, Default, 1, gocv.Filled, false)
	gocv.PutTextWithParams(&mat, purple, image.Pt(point.X, 42), gocv.FontHersheyPlain, 1, team.Purple.RGBA, 1, gocv.Filled, false)
	gocv.PutTextWithParams(&mat, orange, image.Pt(win.bg.Cols()-75, 42), gocv.FontHersheyPlain, 1, team.Orange.RGBA, 1, gocv.Filled, false)
	gocv.PutTextWithParams(&mat, clock, image.Pt(win.bg.Cols()-75, 21), gocv.FontHersheyPlain, 1, Default, 1, gocv.Filled, false)

	gocv.PutTextWithParams(&mat, win.line(), image.Pt(0, 63), gocv.FontHersheyPlain, 1, Default, 1, gocv.Filled, false)

	msgs := win.messages
	if len(win.messages) > lines {
		msgs = win.messages[len(win.messages)-lines:]
	}

	for i, msg := range msgs {
		gocv.PutTextWithParams(&mat, msg.txt, image.Pt(point.X, point.Y+(21*i)), gocv.FontHersheyPlain, 1, msg.rgba, 1, gocv.Filled, false)
	}

	return mat
}

func (w *window) visible() bool {
	return win.GetWindowProperty(gocv.WindowPropertyVisible) != 0
}
