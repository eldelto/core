package musical

import (
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

func TestChordName(t *testing.T) {
	tests := []struct {
		chord        Chord
		expectedName string
	}{
		{NewTriad([3]Note{C, E, G}), "C major"},
		{NewTriad([3]Note{D, F, A}), "D minor"},
		{NewTriad([3]Note{G, B.Accidental(Flat), D.TransposeOctave(1).Accidental(Flat)}), "G diminished"},
		{NewTriad([3]Note{F, A, C.TransposeOctave(1).Accidental(Sharp)}), "F augmented"},
	}

	for _, tt := range tests {
		t.Run(tt.expectedName, func(t *testing.T) {
			AssertEquals(t, tt.expectedName, tt.chord.Name(), "chord.Name()")
		})
	}
}
