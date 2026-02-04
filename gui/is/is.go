package is

type what int

const (
	// Closing indicates the UI is shutting down.
	Closing what = iota
	// Loading indicates the UI is in its startup/loading state.
	Loading
	// MainMenu indicates the UI is showing the main menu.
	MainMenu
	// Configuring indicates the UI is displaying configuration views.
	Configuring
)

var now what = Loading

// Currently reports whether the UI is in the given state.
func Currently(w what) bool {
	return now == w
}

// Next sets the current UI state.
func Next(w what) {
	now = w
}

// String formats the state for display.
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
