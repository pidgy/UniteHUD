package terminal

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"os"
	"strings"
	"time"

	"gocv.io/x/gocv"

	"github.com/pidgy/unitehud/team"
)

type Terminal struct {
	*gocv.Window
	messageq chan message
	messages []message
	bg       gocv.Mat

	purple string
	orange string
	clock  string
}

type message struct {
	rgba color.RGBA
	txt  string
}

var (
	White = color.RGBA{255, 255, 255, 255}

	term *Terminal

	point = image.Pt(10, 84)
	lines = 19
	title = "Pokemon Unite HUD Server"
)

func Close() {
	if term != nil && term.IsOpen() {
		term.Close()
	}
}

func Init() error {
	term = &Terminal{
		Window:   gocv.NewWindow("Pokemon Unite HUD Server"),
		messageq: make(chan message, math.MaxUint16),
		messages: []message{},
	}

	f, err := os.Open("img/bg.png")
	if err != nil {
		return nil
	}

	img, err := png.Decode(f)
	if err != nil {
		return nil
	}

	term.bg, err = gocv.ImageToMatRGBA(img)
	if err != nil {
		return nil
	}

	Score(0, 0, 0)
	Time(0)

	return nil
}

func Show() {
	mat := term.bg.Clone()
	term.ResizeWindow(mat.Cols(), mat.Rows())
	term.IMShow(mat)

	for {
		if !term.visible() {
			break
		}

		select {
		case msg := <-term.messageq:
			mat.Close()
			mat = redraw(msg)
			term.IMShow(mat)
		default:
			term.WaitKey(1)
		}
	}

	if term.visible() {
		term.Close()
	}
}

func Score(p, o, s int) {
	if term == nil {
		return
	}

	term.purple = fmt.Sprintf("%d/%d", s, p)
	term.orange = fmt.Sprintf("%d", o)
}

func Time(seconds int) {
	if term == nil {
		return
	}

	term.clock = fmt.Sprintf("[%02d:%02d]", seconds/60, seconds%60)
}

func Write(rgba color.RGBA, txt ...string) {
	if term == nil {
		return
	}

	if len(txt) == 0 {
		return
	}

	select {
	case term.messageq <- message{rgba: rgba, txt: "[" + time.Now().Format(time.Kitchen) + "] " + strings.Join(txt, " ")}:
	default:
	}
}

func (w *Terminal) chunk(m message) {
	txt := ""

	for i := 0; i < len(m.txt); i++ {
		txt += string(m.txt[i])

		if i != 0 && i%term.bg.Cols() == 0 {
			term.messages = append(term.messages, message{rgba: m.rgba, txt: txt})
			txt = ""
		}
	}

	if txt != "" {
		term.messages = append(term.messages, message{rgba: m.rgba, txt: txt})
	}
}

func (w *Terminal) line() string {
	line := ""
	for i := 0; i < w.bg.Cols(); i++ {
		line += "-"
	}
	return line
}

func redraw(m message) gocv.Mat {
	term.chunk(m)

	mat := term.bg.Clone()

	size := gocv.GetTextSize(" ", gocv.FontHersheyPlain, 1, 1).X

	gocv.PutTextWithParams(&mat, title, image.Pt(point.X, 21), gocv.FontHersheyPlain, 1, White, 1, gocv.Filled, false)
	gocv.PutTextWithParams(&mat, term.clock, image.Pt(term.bg.Cols()-75, 21), gocv.FontHersheyPlain, 1, White, 1, gocv.Filled, false)
	points := strings.Split(term.purple, "/")
	gocv.PutTextWithParams(&mat, points[0], image.Pt(point.X, 42), gocv.FontHersheyPlain, 1, color.RGBA{0, 255, 0, 255}, 1, gocv.Filled, false)
	gocv.PutTextWithParams(&mat, "/", image.Pt(point.X+(len(points[0])*size), 42), gocv.FontHersheyPlain, 1, White, 1, gocv.Filled, false)
	gocv.PutTextWithParams(&mat, points[1], image.Pt(point.X+((len(points[0])+1)*size), 42), gocv.FontHersheyPlain, 1, team.Purple.RGBA, 1, gocv.Filled, false)
	gocv.PutTextWithParams(&mat, term.orange, image.Pt(term.bg.Cols()-75, 42), gocv.FontHersheyPlain, 1, team.Orange.RGBA, 1, gocv.Filled, false)

	gocv.PutTextWithParams(&mat, term.line(), image.Pt(0, 63), gocv.FontHersheyPlain, 1, White, 1, gocv.Filled, false)

	msgs := term.messages
	if len(term.messages) > lines {
		msgs = term.messages[len(term.messages)-lines:]
	}

	for i, msg := range msgs {
		gocv.PutTextWithParams(&mat, msg.txt, image.Pt(point.X, point.Y+(21*i)), gocv.FontHersheyPlain, 1, msg.rgba, 1, gocv.Filled, false)
	}

	return mat
}

func (w *Terminal) visible() bool {
	return term.GetWindowProperty(gocv.WindowPropertyVisible) != 0
}
