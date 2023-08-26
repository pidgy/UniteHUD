package is

type Is int

const (
	Closing Is = iota
	Loading
	MainMenu
	Projecting
	TabMenu
	Display
)

func (i Is) String() string {
	switch i {
	case Closing:
		return "Closing"
	case Loading:
		return "Loading"
	case MainMenu:
		return "MainMenu"
	case Projecting:
		return "Projecting"
	case TabMenu:
		return "TabMenu"
	case Display:
		return "Display"
	}
	return "Unknown"
}
