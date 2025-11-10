package splash

import (
	"encoding/base64"
	"image/png"
	"os"
	"testing"
)

func TestCache(t *testing.T) {
	b, err := os.ReadFile("default.png")
	if err != nil {
		t.Fatal(err)
	}

	b64 := base64.StdEncoding.EncodeToString(b)

	g, err := os.Create("b64.go")
	if err != nil {
		t.Fatal(err)
	}

	_, err = g.Write([]byte("package splash\n\n"))
	if err != nil {
		t.Fatal(err)
	}

	_, err = g.Write([]byte("var defaultb64 = `" + b64 + "`"))
	if err != nil {
		t.Fatal(err)
	}

}

func TestReadDefaultB64(t *testing.T) {
	f, err := os.Create("default2.png")
	if err != nil {
		t.Fatal(err)
	}

	err = png.Encode(f, defaultPNG)
	if err != nil {
		t.Fatal(err)
	}
}
