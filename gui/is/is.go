package is

type Is int

const (
	Closing Is = iota
	Loading
	MainMenu
	Projecting
	TabMenu
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
	}
	return "Unknown"
}
