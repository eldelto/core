package musical

import "strings"

// 1 = a semitone
type ScaleDegrees struct {
	name      string
	intervals []int
}

func (s *ScaleDegrees) Shift(name string, offset int) ScaleDegrees {
	// Decrease the offset so it is consistent with music literature to start
	// on the n-th scale degree.
	offset--
	intervalLen := len(s.intervals)
	newIntervals := make([]int, intervalLen)

	for i := 0; i < intervalLen; i++ {
		newIntervals[i] = s.intervals[(offset+i)%intervalLen]
	}

	return ScaleDegrees{
		name:      name,
		intervals: newIntervals,
	}
}

var (
	MajorScaleDegrees = ScaleDegrees{
		name:      "major",
		intervals: []int{2, 2, 1, 2, 2, 2, 1},
	}
	MinorScaleDegrees = MajorScaleDegrees.Shift("minor", 6)
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
	Tonic   Note
	Degrees ScaleDegrees
}

func (s *Scale) Name() string {
	return s.Tonic.ShortName() + " " + s.Degrees.name
}

func (s *Scale) Notes() []Note {
	return notesFromIntervals(s.Tonic, s.Degrees.intervals)
}

func (s *Scale) Chords() []Chord {
	extendedIntervals := make([]int, len(s.Degrees.intervals))
	copy(extendedIntervals, s.Degrees.intervals)
	extendedIntervals = append(extendedIntervals, s.Degrees.intervals...)

	chords := make([]Chord, len(s.Degrees.intervals))
	notes := s.Notes()
	for i, root := range notes[:len(notes)-1] {
		second := root.TransposeSemitone(extendedIntervals[i] + extendedIntervals[i+1])
		third := second.TransposeSemitone(extendedIntervals[i+2] + extendedIntervals[i+3])
		chords[i] = NewTriad([3]Note{root, second, third})
	}

	return chords
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
