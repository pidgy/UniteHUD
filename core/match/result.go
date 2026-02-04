package match

// Result represents the outcome of a match operation.
type Result int

// String returns a human-readable label for the Result.
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
