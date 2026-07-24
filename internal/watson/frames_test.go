package watson

import (
	"testing"
	"time"
)

const goldenFrames = `[
 [
  1658000000,
  1658003600,
  "watson-tui",
  "9f3a2cf607e34b1c8f0e0c1234567890",
  [
   "cli",
   "docs"
  ],
  1658003600
 ],
 [
  1658007200,
  1658010800,
  "büro & R&D",
  "0aa2b3c4d5e6f708192a3b4c5d6e7f80",
  [],
  1658010800
 ]
]`

func TestParseFramesGolden(t *testing.T) {
	frames, err := ParseFrames([]byte(goldenFrames))
	if err != nil {
		t.Fatal(err)
	}
	if len(frames) != 2 {
		t.Fatalf("got %d frames, want 2", len(frames))
	}
	f := frames[0]
	if !f.Start.Equal(time.Unix(1658000000, 0)) {
		t.Errorf("start = %v", f.Start)
	}
	if !f.Stop.Equal(time.Unix(1658003600, 0)) {
		t.Errorf("stop = %v", f.Stop)
	}
	if f.Project != "watson-tui" {
		t.Errorf("project = %q", f.Project)
	}
	if f.ID != "9f3a2cf607e34b1c8f0e0c1234567890" {
		t.Errorf("id = %q", f.ID)
	}
	if len(f.Tags) != 2 || f.Tags[0] != "cli" || f.Tags[1] != "docs" {
		t.Errorf("tags = %v", f.Tags)
	}
	if frames[1].Project != "büro & R&D" {
		t.Errorf("unicode/html project = %q", frames[1].Project)
	}
	if len(frames[1].Tags) != 0 {
		t.Errorf("empty tags = %v", frames[1].Tags)
	}
}

func TestMarshalFramesRoundTripBytes(t *testing.T) {
	frames, err := ParseFrames([]byte(goldenFrames))
	if err != nil {
		t.Fatal(err)
	}
	out, err := MarshalFrames(frames)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != goldenFrames {
		t.Errorf("round trip mismatch:\ngot:\n%s\nwant:\n%s", out, goldenFrames)
	}
}

func TestMarshalFramesEmpty(t *testing.T) {
	out, err := MarshalFrames(nil)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != "[]" {
		t.Errorf("got %q, want []", out)
	}
}

func TestParseFramesErrors(t *testing.T) {
	if _, err := ParseFrames([]byte("{ nope")); err == nil {
		t.Error("want error for corrupt json")
	}
	if _, err := ParseFrames([]byte(`[[1, 2, "p"]]`)); err == nil {
		t.Error("want error for short row")
	}
}

func TestFindByIDPrefix(t *testing.T) {
	frames := []Frame{
		{ID: "9f3a2cf607e34b1c8f0e0c1234567890"},
		{ID: "9f9999f607e34b1c8f0e0c1234567890"},
		{ID: "0aa2b3c4d5e6f708192a3b4c5d6e7f80"},
	}
	if i, err := FindByIDPrefix(frames, "0aa2b3c"); err != nil || i != 2 {
		t.Errorf("got (%d, %v), want (2, nil)", i, err)
	}
	if _, err := FindByIDPrefix(frames, "9f"); err != ErrAmbiguous {
		t.Errorf("got %v, want ErrAmbiguous", err)
	}
	if _, err := FindByIDPrefix(frames, "ffff"); err != ErrNotFound {
		t.Errorf("got %v, want ErrNotFound", err)
	}
}

func TestSortFrames(t *testing.T) {
	frames := []Frame{
		{ID: "b", Start: time.Unix(200, 0)},
		{ID: "a", Start: time.Unix(100, 0)},
	}
	SortFrames(frames)
	if frames[0].ID != "a" {
		t.Errorf("not sorted by start: %v", frames)
	}
}

func TestShortID(t *testing.T) {
	f := Frame{ID: "9f3a2cf607e34b1c8f0e0c1234567890"}
	if f.ShortID() != "9f3a2cf" {
		t.Errorf("got %q", f.ShortID())
	}
}
