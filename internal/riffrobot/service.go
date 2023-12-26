package riffrobot

import (
	"crypto/sha1"
	"fmt"
	"math/rand"

	"github.com/eldelto/core/internal/musical"
)

var (
	degrees = []musical.ScaleDegrees{
		musical.MajorScaleDegrees,
	}
	tonics = []musical.Note{
		musical.A,
		musical.B,
		musical.C,
		musical.D,
		musical.E,
		musical.F,
		musical.G,
	}
)

func RandomScale(seed string) (musical.Scale, error) {
	hash, err := sha1.New().Write([]byte(seed))
	if err != nil {
		return musical.Scale{}, fmt.Errorf("failed to hash seed %q: %w", seed, err)
	}

	r := rand.New(rand.NewSource(int64(hash)))
	randValue := r.Int()

	tonic := tonics[len(tonics)%randValue-1]
	// TODO: Add random accidental.
	degree := degrees[len(degrees)%randValue-1]
	scale := musical.Scale{Tonic: tonic, Degrees: degree}

	return scale, nil
}
