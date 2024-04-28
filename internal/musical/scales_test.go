package musical

import (
	"strings"
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

func TestScaleCreation(t *testing.T) {
	tests := []struct {
		name     string
		expected string
		scale    Scale
	}{
		{
			name:     "C major",
			expected: "C - D - E - F - G - A - B - C",
			scale:    Scale{Tonic: C, Degrees: MajorScaleDegrees},
		},
		{
			name:     "A major",
			expected: "A - B - C# - D - E - F# - G# - A",
			scale:    Scale{Tonic: A, Degrees: MajorScaleDegrees},
		},
		{
			name:     "C minor",
			expected: "C - D - D# - F - G - G# - A# - C",
			scale:    Scale{Tonic: C, Degrees: MinorScaleDegrees},
		},
		{
			name:     "C lydian",
			expected: "C - D - E - F# - G - A - B - C",
			scale:    Scale{Tonic: C, Degrees: LydianScaleDegrees},
		},
		{
			name:     "B hirajōshi",
			expected: "B - C# - D - F# - G",
			scale:    Scale{Tonic: B, Degrees: HirajōshiScaleDegrees},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AssertEquals(t, tt.name, tt.scale.Name(), "scale.Name()")
			AssertEquals(t, tt.expected, tt.scale.String(), "scale.Notes()")
		})
	}
}

func TestScaleChords(t *testing.T) {
	tests := []struct {
		scale    Scale
		expected string
	}{
		{
			scale:    Scale{Tonic: C, Degrees: MajorScaleDegrees},
			expected: "C major - D minor - E minor - F major - G major - A minor - B diminished",
		},
		{
			scale:    Scale{Tonic: A, Degrees: MajorScaleDegrees},
			expected: "A major - B minor - C# minor - D major - E major - F# minor - G# diminished",
		},
		{
			scale:    Scale{Tonic: C, Degrees: MinorScaleDegrees},
			expected: "C minor - D diminished - D# major - F minor - G minor - G# major - A# major",
		},
	}

	for _, tt := range tests {
		t.Run(tt.scale.Name(), func(t *testing.T) {
			builder := strings.Builder{}
			chords := tt.scale.Chords()

			builder.WriteString(chords[0].Name())
			for i := 1; i < len(chords); i++ {
				builder.WriteString(" - ")
				builder.WriteString(chords[i].Name())
			}

			AssertEquals(t, tt.expected, builder.String(), "scale.Chords()")
		})
	}
}
