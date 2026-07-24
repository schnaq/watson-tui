package watson

import (
	"path/filepath"
	"testing"
)

func TestDirOverrideWins(t *testing.T) {
	t.Setenv("WATSON_DIR", "/env/watson")
	got, err := Dir("/explicit")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/explicit" {
		t.Errorf("got %q, want /explicit", got)
	}
}

func TestDirEnvVar(t *testing.T) {
	t.Setenv("WATSON_DIR", "/env/watson")
	got, err := Dir("")
	if err != nil {
		t.Fatal(err)
	}
	if got != "/env/watson" {
		t.Errorf("got %q, want /env/watson", got)
	}
}

func TestDirDefaultEndsWithWatson(t *testing.T) {
	t.Setenv("WATSON_DIR", "")
	got, err := Dir("")
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(got) != "watson" {
		t.Errorf("got %q, want path ending in /watson", got)
	}
}
