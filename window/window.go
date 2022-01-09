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

	purple string
	orange string
	clock  string
}

var (
	Default = color.RGBA{255, 255, 255, 255}
)

var (
	win = &window{
		Window:   gocv.NewWindow("Pokemon Unite HUD Server"),
		messageq: make(chan message),
		messages: []message{},
	}

	point = image.Pt(10, 84)
	lines = 19
	title = "Pokemon Unite HUD Server"
)

func Close() {
	if win != nil && win.IsOpen() {
		win.Close()
	}
}

func Init() error {
	f, err := os.Open("img/bg.png")
	if err != nil {
		return nil
	}

	img, err := png.Decode(f)
	if err != nil {
		return nil
	}

	win.bg, err = gocv.ImageToMatRGBA(img)
	if err != nil {
		return nil
	}

	Score(0, 0, 0)
	Time(0)

	return nil
}

func Open() {
	mat := win.bg.Clone()
	win.ResizeWindow(mat.Cols(), mat.Rows())
	win.IMShow(mat)

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
	win.purple = fmt.Sprintf("%d/%d", s, p)
	win.orange = fmt.Sprintf("%d", o)
}

func Time(seconds int) {
	win.clock = fmt.Sprintf("[%02d:%02d]", seconds/60, seconds%60)
}

func Write(rgba color.RGBA, txt ...string) {
	if len(txt) == 0 {
		return
	}

	go func() {
		win.messageq <- message{rgba: rgba, txt: "[" + time.Now().Format(time.Kitchen) + "] " + strings.Join(txt, " ")}
	}()
}

func (w *window) chunk(m message) {
	txt := ""

	for i := 0; i < len(m.txt); i++ {
		txt += string(m.txt[i])

		if i != 0 && i%win.bg.Cols() == 0 {
			win.messages = append(win.messages, message{rgba: m.rgba, txt: txt})
			txt = ""
		}
	}

	if txt != "" {
		win.messages = append(win.messages, message{rgba: m.rgba, txt: txt})
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

	size := gocv.GetTextSize(" ", gocv.FontHersheyPlain, 1, 1).X

	gocv.PutTextWithParams(&mat, title, image.Pt(point.X, 21), gocv.FontHersheyPlain, 1, Default, 1, gocv.Filled, false)
	gocv.PutTextWithParams(&mat, win.clock, image.Pt(win.bg.Cols()-75, 21), gocv.FontHersheyPlain, 1, Default, 1, gocv.Filled, false)
	points := strings.Split(win.purple, "/")
	gocv.PutTextWithParams(&mat, points[0], image.Pt(point.X, 42), gocv.FontHersheyPlain, 1, color.RGBA{0, 255, 0, 255}, 1, gocv.Filled, false)
	gocv.PutTextWithParams(&mat, "/", image.Pt(point.X+(len(points[0])*size), 42), gocv.FontHersheyPlain, 1, Default, 1, gocv.Filled, false)
	gocv.PutTextWithParams(&mat, points[1], image.Pt(point.X+((len(points[0])+1)*size), 42), gocv.FontHersheyPlain, 1, team.Purple.RGBA, 1, gocv.Filled, false)
	gocv.PutTextWithParams(&mat, win.orange, image.Pt(win.bg.Cols()-75, 42), gocv.FontHersheyPlain, 1, team.Orange.RGBA, 1, gocv.Filled, false)

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
