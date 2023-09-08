package is

type Is int

const (
	Closing Is = iota
	Loading
	MainMenu
	Projecting
	Display
)

var Now Is = Loading

func (i Is) String() string {
	switch i {
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
