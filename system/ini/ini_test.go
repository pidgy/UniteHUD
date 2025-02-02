package ini

import (
	"testing"

	"gopkg.in/ini.v1"
)

func TestFormat(t *testing.T) {
	i, err := ini.Load(`C:\Users\trash\Documents\dev\go\src\github.com\pidgy\unitehud\assets\ini\en-US.ini`)
	if err != nil {
		t.Fatal(err)
	}
	file = i

	type test struct {
		format,
		want string
	}

	tests := []test{
		{
			format: "[UniteHUD] <ini:error:failed_to_load> %s <ini:toast:connect_discord_remember> (%v)",
			want:   "[UniteHUD] Failed to load %s Don't ask me again (%v)",
		},
		{
			format: "[UniteHUD] Failed to load %s Don't ask me again (%v)",
			want:   "[UniteHUD] Failed to load %s Don't ask me again (%v)",
		},
		{
			format: "<ini: blah",
			want:   "<ini: blah",
		},
		{
			format: "<ini:unk:unk> blah",
			want:   "unk-unk blah",
		},
	}

	for _, test := range tests {
		got := Format(test.format)
		if got != test.want {
			t.Fatalf("want: %s, got: %s", test.want, got)
		}
	}
}

func TestOpen(t *testing.T) {
	err := Open("en-US")
	if err != nil {
		t.Fatal(err)
	}
}
