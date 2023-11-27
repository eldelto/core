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
			name:     "C0 major",
			expected: "C0 - D0 - E0 - F0 - G0 - A0 - B0 - C1",
			scale:    Scale{Tonic: C, Steps: MajorSteps},
		},
		{
			name:     "A0 major",
			expected: "A0 - B0 - C#1 - D1 - E1 - F#1 - G#1 - A1",
			scale:    Scale{Tonic: A, Steps: MajorSteps},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AssertEquals(t, tt.name, tt.scale.Name(), "scale.Name()")

			builder := strings.Builder{}
			for i, note := range tt.scale.Notes() {
				if i != 0 {
					builder.WriteString(" - ")
				}
				builder.WriteString(note.String())
			}

			AssertEquals(t, tt.expected, builder.String(), "scale.Notes()")
		})
	}
}
