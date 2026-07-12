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
	Bind struct {
		binds []bind
		tag   any
	}
)

const (
	// None represents an empty binding.
	None = ""
	// NoMod represents no modifier keys.
	NoMod = key.Modifiers(0)
	// CtrlMod represents the Ctrl modifier.
	CtrlMod = key.ModCtrl

	// HintText is forwarded from gio's key hint text.
	HintText = key.HintText

	CommandMod = key.ModCommand
)

// New creates an empty binding list.
func New() *Bind {
	return &Bind{binds: []bind{}, tag: new(bool)}
}

// Bind appends a new binding for the given modifiers and keys.
func (b *Bind) Bind(m key.Modifiers, s ...string) *Bind {
	b.binds = append(b.binds, bind{set: key.Set(strings.Join(s, "|")), mods: m})
	return b
}

// Enter reports whether the escape binding fired.
func (b *Bind) Enter(gtx layout.Context) bool {
	return b.Event(gtx) == Enter()
}

// Escape reports whether the escape binding fired.
func (b *Bind) Escape(gtx layout.Context) bool {
	return b.Event(gtx) == Escape()
}

// Event processes key events and returns the matched binding name.
func (b *Bind) Event(gtx layout.Context) (name string) {
	if len(b.binds) == 0 {
		return ""
	}

	for _, e := range gtx.Events(b.tag) {
		event, ok := e.(key.Event)
		if !ok {
			continue
		}
		if event.State != key.Release {
			continue
		}
		for _, bind := range b.binds {
			if bind.set.Contains(event.Name, 0) && event.Modifiers.Contain(bind.mods) {
				name = event.Name
				if bind.mods != NoMod {
					name = bind.mods.String() + "-" + event.Name
				}
				goto push
			}
		}
	}

push:
	set := b.binds[0].set
	for _, bind := range b.binds[1:] {
		set = key.Set(fmt.Sprintf("%s|%s", set, bind.set))
	}

	area := clip.Rect(gtx.Constraints).Push(gtx.Ops)
	key.InputOp{
		Tag:  b.tag,
		Keys: set,
	}.Add(gtx.Ops)
	area.Pop()

	return name
}

func Command(k string) string {
	return fmt.Sprintf("%s-%s", key.NameCommand, k)
}

func Ctrl(k string) string {
	return fmt.Sprintf("%s-%s", key.NameCtrl, k)
}

func Enter() string {
	return key.NameEnter
}

// Escape returns the key name for Escape.
func Escape() string {
	return key.NameEscape
}

// F11 returns the key name for F11.
func F11() string {
	return key.NameF11
}

func Return() string {
	return key.NameReturn
}
