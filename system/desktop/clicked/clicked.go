package clicked

import "git.sr.ht/~jackmordaunt/go-toast"

type Action struct {
	then toast.Action
}

var (
	VisitWebsite = Action{
		then: toast.Action{
			Type:      toast.Protocol,
			Content:   "Download",
			Arguments: "https://unitehud.dev",
		},
	}
	OpenUniteHUD = Action{
		then: toast.Action{
			Type:      toast.Foreground,
			Content:   "Open",
			Arguments: "",
		},
	}
	ViewDetails = Action{
		then: toast.Action{
			Type:      toast.Foreground,
			Content:   "View Details",
			Arguments: "UniteHUD.exe",
		},
	}
)

func (a Action) Then() toast.Action {
	return a.then
}
