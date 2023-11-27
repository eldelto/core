package musical

import (
	"strings"
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

func TestNotesManipulation(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		f        func(n Note) Note
	}{
		{
			name:     "raise semitone",
			expected: "C2 - C#2 - D2 - D#2 - E2 - F2 - F#2 - G2 - G#2 - A2 - A#2 - B2 - C3 - C#3 - D3 - D#3",
			f:        func(n Note) Note { return n.TransposeSemitone(1) },
		},
		{
			name:     "lower semitone",
			expected: "C2 - B1 - Bb1 - A1 - Ab1 - G1 - Gb1 - F1 - E1 - Eb1 - D1 - Db1 - C1 - B0 - Bb0 - A0",
			f:        func(n Note) Note { return n.TransposeSemitone(-1) },
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			note := C.TransposeOctave(2)
			builder := strings.Builder{}

			builder.WriteString(note.String())
			for i := 0; i < 15; i++ {
				note = tt.f(note)
				builder.WriteString(" - ")
				builder.WriteString(note.String())
			}

			AssertEquals(t, tt.expected, builder.String(), "notes")
		})
	}
}
