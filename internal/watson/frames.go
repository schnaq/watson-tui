package watson

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
)

var (
	ErrNotFound  = errors.New("frame not found")
	ErrAmbiguous = errors.New("frame id prefix is ambiguous")
)

// Frame is one completed tracking entry. Times are UTC.
type Frame struct {
	Start     time.Time
	Stop      time.Time
	Project   string
	ID        string // 32-char lowercase hex UUID v4, no dashes
	Tags      []string
	UpdatedAt time.Time
}

func (f Frame) Duration() time.Duration { return f.Stop.Sub(f.Start) }

func (f Frame) ShortID() string {
	if len(f.ID) < 7 {
		return f.ID
	}
	return f.ID[:7]
}

// marshalNoEscape is json.Marshal without HTML escaping and trailing newline.
func marshalNoEscape(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// MarshalJSON emits Watson's array form [start, stop, project, id, tags, updated_at].
func (f Frame) MarshalJSON() ([]byte, error) {
	tags := f.Tags
	if tags == nil {
		tags = []string{}
	}
	return marshalNoEscape([]any{
		f.Start.Unix(),
		f.Stop.Unix(),
		f.Project,
		f.ID,
		tags,
		f.UpdatedAt.Unix(),
	})
}

// UnmarshalJSON reads Watson's array form. stop may be null (defensive).
func (f *Frame) UnmarshalJSON(data []byte) error {
	var row []json.RawMessage
	if err := json.Unmarshal(data, &row); err != nil {
		return err
	}
	if len(row) != 6 {
		return fmt.Errorf("frame row has %d fields, want 6", len(row))
	}
	var start, updated int64
	var stop *int64
	fields := []struct {
		dst  any
		name string
	}{
		{&start, "start"}, {&stop, "stop"}, {&f.Project, "project"},
		{&f.ID, "id"}, {&f.Tags, "tags"}, {&updated, "updated_at"},
	}
	for i, fd := range fields {
		if err := json.Unmarshal(row[i], fd.dst); err != nil {
			return fmt.Errorf("frame field %s: %w", fd.name, err)
		}
	}
	f.Start = time.Unix(start, 0).UTC()
	if stop != nil {
		f.Stop = time.Unix(*stop, 0).UTC()
	} else {
		f.Stop = time.Time{}
	}
	f.UpdatedAt = time.Unix(updated, 0).UTC()
	if f.Tags == nil {
		f.Tags = []string{}
	}
	return nil
}

// ParseFrames decodes the frames file content.
func ParseFrames(data []byte) ([]Frame, error) {
	var frames []Frame
	if err := json.Unmarshal(data, &frames); err != nil {
		return nil, err
	}
	if frames == nil {
		frames = []Frame{}
	}
	return frames, nil
}

// MarshalFrames encodes frames in Watson's on-disk format:
// indent=1, no HTML escaping, no trailing newline (Python json.dumps equivalent).
func MarshalFrames(frames []Frame) ([]byte, error) {
	if len(frames) == 0 {
		return []byte("[]"), nil
	}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", " ")
	if err := enc.Encode(frames); err != nil {
		return nil, err
	}
	return bytes.TrimRight(buf.Bytes(), "\n"), nil
}

// SortFrames sorts by start time ascending (stable).
func SortFrames(frames []Frame) {
	sort.SliceStable(frames, func(i, j int) bool { return frames[i].Start.Before(frames[j].Start) })
}

// FindByIDPrefix returns the index of the frame whose ID starts with prefix.
func FindByIDPrefix(frames []Frame, prefix string) (int, error) {
	idx := -1
	for i, fr := range frames {
		if strings.HasPrefix(fr.ID, prefix) {
			if idx != -1 {
				return -1, ErrAmbiguous
			}
			idx = i
		}
	}
	if idx == -1 {
		return -1, ErrNotFound
	}
	return idx, nil
}
