package watson

import (
	"testing"
	"time"
)

const goldenState = `{
 "project": "watson-tui",
 "start": 1658000000,
 "tags": [
  "cli"
 ]
}`

func TestParseStateRunning(t *testing.T) {
	s, err := ParseState([]byte(goldenState))
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("got nil, want running state")
	}
	if s.Project != "watson-tui" || !s.Start.Equal(time.Unix(1658000000, 0)) || len(s.Tags) != 1 {
		t.Errorf("state = %+v", s)
	}
}

func TestParseStateEmpty(t *testing.T) {
	for _, in := range []string{"{}", "", "  \n"} {
		s, err := ParseState([]byte(in))
		if err != nil {
			t.Fatalf("%q: %v", in, err)
		}
		if s != nil {
			t.Errorf("%q: got %+v, want nil", in, s)
		}
	}
}

func TestParseStateCorrupt(t *testing.T) {
	if _, err := ParseState([]byte("{ nope")); err == nil {
		t.Error("want error")
	}
}

func TestMarshalStateRoundTrip(t *testing.T) {
	s, err := ParseState([]byte(goldenState))
	if err != nil {
		t.Fatal(err)
	}
	out, err := MarshalState(s)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != goldenState {
		t.Errorf("got:\n%s\nwant:\n%s", out, goldenState)
	}
}

func TestMarshalStateNil(t *testing.T) {
	out, err := MarshalState(nil)
	if err != nil || string(out) != "{}" {
		t.Errorf("got %q, %v", out, err)
	}
}
