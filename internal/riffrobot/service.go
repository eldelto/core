package riffrobot

import (
	"fmt"
	"hash/fnv"
	"math/rand"

	"github.com/eldelto/core/internal/musical"
)

var (
	degrees = []musical.ScaleDegrees{
		musical.MajorScaleDegrees,
		musical.MinorScaleDegrees,
		musical.DorianScaleDegrees,
		musical.PhrygianScaleDegrees,
		musical.LydianScaleDegrees,
		musical.MixolydianScaleDegrees,
		musical.MinorScaleDegrees,
		musical.LocrianScaleDegrees,
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
	hasher := fnv.New32()
	if _, err := hasher.Write([]byte(seed)); err != nil {
		return musical.Scale{}, fmt.Errorf("failed to hash seed %q: %w", seed, err)
	}

	r := rand.New(rand.NewSource(int64(hasher.Sum32())))
	randValue := r.Int()

	tonic := tonics[randValue%len(tonics)]
	// TODO: Add random accidental.
	degree := degrees[randValue%len(degrees)]
	scale := musical.Scale{Tonic: tonic, Degrees: degree}

	return scale, nil
}
