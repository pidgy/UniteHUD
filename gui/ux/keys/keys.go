package keys

import (
	"fmt"
	"strings"

	"gioui.org/io/key"
	"gioui.org/layout"
	"gioui.org/op/clip"
)

type (
	// bind pairs a key set with required modifiers.
	bind struct {
		set  key.Set
		mods key.Modifiers
	}

	// Bind is a collection of key bindings.
	Bind []bind
)

const (
	// None represents an empty binding.
	None    = ""
	// NoMod represents no modifier keys.
	NoMod   = key.Modifiers(0)
	// CtrlMod represents the Ctrl modifier.
	CtrlMod = key.ModCtrl

	// HintText is forwarded from gio's key hint text.
	HintText = key.HintText
)

// New creates an empty binding list.
func New() Bind {
	return []bind{}
}

// Bind appends a new binding for the given modifiers and keys.
func (b Bind) Bind(m key.Modifiers, s ...string) Bind {
	return append(b, bind{set: key.Set(strings.Join(s, "|")), mods: m})
}

// Escape reports whether the escape binding fired.
func (b Bind) Escape(gtx layout.Context, tag any) bool {
	return b.Event(gtx, tag) == Escape()
}

// Event processes key events and returns the matched binding name.
func (b Bind) Event(gtx layout.Context, tag any) (name string) {
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

// Escape returns the key name for Escape.
func Escape() string {
	return key.NameEscape
}

// F11 returns the key name for F11.
func F11() string {
	return key.NameF11
}
