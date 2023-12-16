package musical

import "fmt"

type Quality uint

const (
	UnknownQuality = Quality(iota)
	Major
	Minor
	Diminished
	Augmented
)

func (q Quality) String() string {
	switch q {
	case UnknownQuality:
		return "unknown"
	case Major:
		return "major"
	case Minor:
		return "minor"
	case Diminished:
		return "diminished"
	case Augmented:
		return "augmented"
	default:
		panic(fmt.Sprintf("unhandled Quality with value '%d'", q))
	}
}

type Chord struct {
	quality Quality
	notes   []Note
}

func NewTriad(notes [3]Note) Chord {
	return Chord{
		quality: triadQuality(notes),
		notes:   notes[:],
	}
}

func triadQuality(notes [3]Note) Quality {
	firstInterval := notes[0].Interval(notes[1])
	secondInterval := notes[1].Interval(notes[2])

	switch {
	case firstInterval == 4 && secondInterval == 3:
		return Major
	case firstInterval == 3 && secondInterval == 4:
		return Minor
	case firstInterval == 3 && secondInterval == 3:
		return Diminished
	case firstInterval == 4 && secondInterval == 4:
		return Augmented
	}

	return UnknownQuality
}

func (c *Chord) Name() string {
	return c.notes[0].ShortName() + " " + c.quality.String()
}
