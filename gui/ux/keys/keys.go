package keys

import (
	"fmt"
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
)

type (
	bind struct {
		set  key.Set
		mods key.Modifiers
	}

	Bind []bind
)

const (
	None  = ""
	NoMod = key.Modifiers(0)
)

var (
	Esc = New().Bind(NoMod, key.NameEscape)
)

func New() Bind {
	return []bind{}
}

func (b Bind) Bind(m key.Modifiers, s ...string) Bind {
	return append(b, bind{set: key.Set(strings.Join(s, "|")), mods: m})
}

func (b Bind) Up(gtx layout.Context, tag any) (name string) {
	if len(b) == 0 {
		return ""
	}

	for _, e := range gtx.Events(tag) {
		event, ok := e.(key.Event)
		if !ok {
			continue
		}
		if event.State != key.Release {
			continue
		}
		for _, bind := range b {
			if bind.set.Contains(event.Name, 0) && bind.mods.Contain(event.Modifiers) {
				name = event.Name
				if bind.mods != NoMod {
					name = bind.mods.String() + "-" + event.Name
				}
				goto push
			}
		}
	}

push:
	set := b[0].set
	for _, bind := range b[1:] {
		set = key.Set(fmt.Sprintf("%s|%s", set, bind.set))
	}

	area := clip.Rect(gtx.Constraints).Push(gtx.Ops)
	key.InputOp{
		Tag:  tag,
		Keys: set,
	}.Add(gtx.Ops)
	area.Pop()

	return name
}

func Ctrl(s string) string {
	return "Ctrl-" + s
}
