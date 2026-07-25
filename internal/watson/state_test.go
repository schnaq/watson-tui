package watson

import (
	"strings"
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

// A well-formed JSON object with an empty or missing project is treated as idle
// (nil, nil), not an error -- defensive, matching the `{}` / empty-file semantics.
func TestParseStateEmptyProject(t *testing.T) {
	for _, in := range []string{`{"start": 1658000000}`, `{"project": ""}`, `{"project": "", "start": 1658000000, "tags": ["cli"]}`} {
		s, err := ParseState([]byte(in))
		if err != nil {
			t.Errorf("%q: got err %v, want nil", in, err)
		}
		if s != nil {
			t.Errorf("%q: got %+v, want nil", in, s)
		}
	}
}

// ParseState must anchor Start to UTC, not merely produce an instant that is
// Equal to the expected time while carrying some other location.
func TestParseStateUTC(t *testing.T) {
	s, err := ParseState([]byte(goldenState))
	if err != nil {
		t.Fatal(err)
	}
	if s == nil {
		t.Fatal("got nil, want running state")
	}
	if !s.Start.Equal(time.Unix(1658000000, 0)) {
		t.Errorf("start = %v, want %v", s.Start, time.Unix(1658000000, 0))
	}
	if s.Start.Location() != time.UTC {
		t.Errorf("start location = %v, want UTC", s.Start.Location())
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

// Regression: the encoder must keep HTML-significant characters raw
// (SetEscapeHTML(false)). If escaping were re-enabled, "&" would be emitted as a
// u0026-style escape and "<b>" as a u003c.../u003e pair, breaking round-trips
// with the real watson, which writes these characters verbatim.
func TestMarshalStateHTMLEscaping(t *testing.T) {
	s := &State{Project: "a&b", Start: time.Unix(1658000000, 0).UTC(), Tags: []string{"<b>"}}
	out, err := MarshalState(s)
	if err != nil {
		t.Fatal(err)
	}
	got := string(out)
	// Positive checks discriminate the regression: the raw "&" and "<b>" only
	// survive verbatim when SetEscapeHTML(false) is in effect; with escaping on
	// they become uXXXX escapes and these substrings vanish.
	if !strings.Contains(got, "a&b") {
		t.Errorf("output should contain raw \"a&b\", got:\n%s", got)
	}
	if !strings.Contains(got, "<b>") {
		t.Errorf("output should contain raw \"<b>\", got:\n%s", got)
	}
	// Negative checks with teeth: the escaped forms contain the substrings
	// "u0026" / "u003c", which never appear in the output when escaping is off.
	if strings.Contains(got, "u0026") || strings.Contains(got, "u003c") {
		t.Errorf("output should not contain uXXXX escapes, got:\n%s", got)
	}
}
