package match

type Result int

func (r Result) String() string {
	switch r {
	case Duplicate:
		return "duplicate"
	case Invalid:
		return "invalid"
	case Missed:
		return "missed"
	case NotFound:
		return "not found"
	case Found:
		return "found"
	}
	return "unknown"
}
