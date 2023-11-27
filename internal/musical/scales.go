package musical

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

type Scale struct {
	Tonic Note
	Steps ScaleSteps
}

func (s *Scale) Name() string {
	return s.Tonic.String() + " " + s.Steps.name
}

func (s *Scale) Notes() []Note {
	notes := make([]Note, len(s.Steps.intervals)+1)
	notes[0] = s.Tonic
	intervals := s.Steps.intervals

	for i := range intervals {
		notes[i+1] = notes[i].TransposeSemitone(intervals[i])
	}

	return notes
}
