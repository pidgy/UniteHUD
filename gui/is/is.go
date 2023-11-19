package is

type What int

const (
	Closing What = iota
	Loading
	MainMenu
	Projecting
	Display
)

var Now What = Loading

func (w What) String() string {
	switch w {
	case Closing:
		return "Closing"
	case Loading:
		return "Loading"
	case MainMenu:
		return "Main Menu"
	case Projecting:
		return "Projector Menu"
	}
	return "Unknown"
}
