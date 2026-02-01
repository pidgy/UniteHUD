package is

type what int

const (
	Closing what = iota
	Loading
	MainMenu
	Configuring
)

var now what = Loading

func Currently(w what) bool {
	return now == w
}

func Set(w what) {
	now = w
}

func (w what) String() string {
	switch w {
	case Closing:
		return "Closing"
	case Loading:
		return "Loading"
	case MainMenu:
		return "Main Menu"
	case Configuring:
		return "Configuring"
	}
	return "Unknown State"
}
