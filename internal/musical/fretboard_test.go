package musical

import (
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

func TestFretboardVisualisation(t *testing.T) {
	tests := []struct {
		name     string
		tuning   Tuning
		expected string
	}{
		{
			name:   "standard E2",
			tuning: TuningEStandard,
			expected: `E4----F4----F#4---G4----G#4---A4----A#4---B4----C5----C#5---D5----D#5---E5----
B3----C4----C#4---D4----D#4---E4----F4----F#4---G4----G#4---A4----A#4---B4----
G3----G#3---A3----A#3---B3----C4----C#4---D4----D#4---E4----F4----F#4---G4----
D3----D#3---E3----F3----F#3---G3----G#3---A3----A#3---B3----C4----C#4---D4----
A2----A#2---B2----C3----C#3---D3----D#3---E3----F3----F#3---G3----G#3---A3----
E2----F2----F#2---G2----G#2---A2----A#2---B2----C3----C#3---D3----D#3---E3----
`,
		},
		{
			name:   "standard D2",
			tuning: TuningDStandard,
			expected: `D4----D#4---E4----F4----F#4---G4----G#4---A4----A#4---B4----C5----C#5---D5----
A3----A#3---B3----C4----C#4---D4----D#4---E4----F4----F#4---G4----G#4---A4----
F3----F#3---G3----G#3---A3----A#3---B3----C4----C#4---D4----D#4---E4----F4----
C3----C#3---D3----D#3---E3----F3----F#3---G3----G#3---A3----A#3---B3----C4----
G2----G#2---A2----A#2---B2----C3----C#3---D3----D#3---E3----F3----F#3---G3----
D2----D#2---E2----F2----F#2---G2----G#2---A2----A#2---B2----C3----C#3---D3----
`,
		},
		{
			name:   "drop D2",
			tuning: TuningDropD,
			expected: `E4----F4----F#4---G4----G#4---A4----A#4---B4----C5----C#5---D5----D#5---E5----
B3----C4----C#4---D4----D#4---E4----F4----F#4---G4----G#4---A4----A#4---B4----
G3----G#3---A3----A#3---B3----C4----C#4---D4----D#4---E4----F4----F#4---G4----
D3----D#3---E3----F3----F#3---G3----G#3---A3----A#3---B3----C4----C#4---D4----
A2----A#2---B2----C3----C#3---D3----D#3---E3----F3----F#3---G3----G#3---A3----
D2----D#2---E2----F2----F#2---G2----G#2---A2----A#2---B2----C3----C#3---D3----
`,
		},
		{
			name:   "drop C2",
			tuning: TuningDropC,
			expected: `D4----D#4---E4----F4----F#4---G4----G#4---A4----A#4---B4----C5----C#5---D5----
A3----A#3---B3----C4----C#4---D4----D#4---E4----F4----F#4---G4----G#4---A4----
F3----F#3---G3----G#3---A3----A#3---B3----C4----C#4---D4----D#4---E4----F4----
C3----C#3---D3----D#3---E3----F3----F#3---G3----G#3---A3----A#3---B3----C4----
G2----G#2---A2----A#2---B2----C3----C#3---D3----D#3---E3----F3----F#3---G3----
C2----C#2---D2----D#2---E2----F2----F#2---G2----G#2---A2----A#2---B2----C3----
`},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AssertEquals(t, tt.name, tt.tuning.Name(), "tuning.Name()")

			fretboard := Fretboard{Tuning: tt.tuning}
			AssertEquals(t, tt.expected, fretboard.String(), "fretboard.String()")
		})
	}
}
