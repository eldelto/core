package musical

import "strings"

// 1 = a semitone
type ScaleSteps struct {
	name      string
	intervals []int
}

var (
	MajorSteps = ScaleSteps{
		name:      "major",
		intervals: []int{2, 2, 1, 2, 2, 2, 1},
	}
)

func notesFromIntervals(tonic Note, intervals []int) []Note {
	notes := make([]Note, len(intervals)+1)
	notes[0] = tonic

	for i := range intervals {
		notes[i+1] = notes[i].TransposeSemitone(intervals[i])
	}

	return notes
}

type Scale struct {
	Tonic Note
	Steps ScaleSteps
}

func (s *Scale) Name() string {
	return s.Tonic.ShortName() + " " + s.Steps.name
}

func (s *Scale) Notes() []Note {
	return notesFromIntervals(s.Tonic, s.Steps.intervals)
}

func (s *Scale) String() string {
	b := strings.Builder{}
	for i, note := range s.Notes() {
		if i != 0 {
			b.WriteString(" - ")
		}
		b.WriteString(note.ShortName())
	}

	return b.String()
}
