package musical

import (
	"slices"
	"strconv"
	"strings"
)

type TuningSteps struct {
	name      string
	intervals []int
}

var (
	StandardTuningSteps = TuningSteps{
		name:      "standard",
		intervals: []int{5, 5, 5, 4, 5},
	}
	DropTuningSteps = TuningSteps{
		name:      "drop",
		intervals: []int{7, 5, 5, 4, 5},
	}
)

type Tuning struct {
	Tonic Note
	Steps TuningSteps
}

func (t *Tuning) Name() string {
	return t.Steps.name + " " + t.Tonic.ShortName()
}

func (t *Tuning) Notes() []Note {
	return notesFromIntervals(t.Tonic, t.Steps.intervals)
}

var (
	TuningEStandard = Tuning{Tonic: E.TransposeOctave(2), Steps: StandardTuningSteps}
	TuningDStandard = Tuning{Tonic: D.TransposeOctave(2), Steps: StandardTuningSteps}
	TuningDropD     = Tuning{Tonic: D.TransposeOctave(2), Steps: DropTuningSteps}
	TuningDropC     = Tuning{Tonic: C.TransposeOctave(2), Steps: DropTuningSteps}
)

type Fretboard struct {
	Tuning Tuning
}

const frets = 12

func (f *Fretboard) String() string {
	b := strings.Builder{}
	notes := f.Tuning.Notes()
	slices.Reverse(notes)

	for fretNumber := 0; fretNumber <= frets; fretNumber++ {
		name := strconv.FormatInt(int64(fretNumber), 10)
		if len(name) < 2 {
			name += "-"
		}
		b.WriteString("--")
		b.WriteString(name)
		b.WriteString("--|")
	}

	for _, openNote := range notes {
		b.WriteByte('\n')

		for fretNumber := 0; fretNumber <= frets; fretNumber++ {
			note := openNote.TransposeSemitone(fretNumber)
			name := note.String()
			if len(name) < 3 {
				name += "-"
			}
			b.WriteString("--")
			b.WriteString(name)
			b.WriteString("-|")
		}
	}

	return b.String()
}
