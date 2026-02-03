package match

// TODO: Add go style comments that reflect the purpose of each type, function, var, and const.

type Result int

func (r Result) String() string {
	switch r {
	case Duplicate:
		return "Duplicate"
	case Invalid:
		return "Invalid"
	case Missed:
		return "Missed"
	case NotFound:
		return "Not Found"
	case Found:
		return "Found"
	}
	return "Unknown"
}
