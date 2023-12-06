package musical

import (
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
			scale:    Scale{Tonic: C, Steps: MajorSteps},
		},
		{
			name:     "A major",
			expected: "A - B - C# - D - E - F# - G# - A",
			scale:    Scale{Tonic: A, Steps: MajorSteps},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			AssertEquals(t, tt.name, tt.scale.Name(), "scale.Name()")
			AssertEquals(t, tt.expected, tt.scale.String(), "scale.Notes()")
		})
	}
}
