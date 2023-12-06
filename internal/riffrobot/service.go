package riffrobot

import "github.com/eldelto/core/internal/musical"

func RandomScale(seed string) *musical.Scale {
	scale := musical.Scale{Tonic: musical.C, Steps: musical.MajorSteps}
	return &scale
}
