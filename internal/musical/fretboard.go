package musical

import (
	"slices"
	"strings"
)

func standardTuningFromRoot(note Note) []Note {
	tuning := make([]Note, 6)
	tuning[0] = note
	for i := 1; i < len(tuning); i++ {
		if i == 4 {
			note = note.TransposeSemitone(4)
		} else {
			note = note.TransposeSemitone(5)
		}

		tuning[i] = note
	}

	slices.Reverse(tuning)
	return tuning
}

var (
	TuningEStandard = standardTuningFromRoot(E.TransposeOctave(2))
	TuningDStandard = standardTuningFromRoot(D.TransposeOctave(2))
)

type Fretboard struct {
	Tuning []Note
}

const frets = 12

func (f *Fretboard) String() string {
	b := strings.Builder{}

	for _, openNote := range f.Tuning {
		for fretNumber := 0; fretNumber <= frets; fretNumber++ {
			note := openNote.TransposeSemitone(fretNumber)
			name := note.String()
			if len(name) < 3 {
				name += "-"
			}
			b.WriteString(name)
			b.WriteString("---")
		}
		b.WriteByte('\n')
	}

	return b.String()
}
