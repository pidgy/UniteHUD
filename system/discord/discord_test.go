package discord

import (
	"fmt"
	"testing"

	"github.com/pidgy/unitehud/core/server"
	"github.com/pidgy/unitehud/core/state"
	"github.com/pidgy/unitehud/core/team"
	"github.com/pidgy/unitehud/gui/is"
)

func TestStatus(t *testing.T) {
	is.Now = is.MainMenu
	server.SetReady()
	server.SetMatchStarted()

	server.SetTime(9, 45)
	server.SetScore(team.Purple, 45)
	server.SetScore(team.Orange, 120)

	a := Activity()
	for e := state.Nothing; e <= state.RayquazaSecurePurple; e++ {
		state.Add(e, "9:45", 12)
		a = Activity()
		fmt.Printf("(%s) Discord: %s: \"%s\"\n", e, a.Details, a.State)
	}
}
