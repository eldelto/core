package musical

import (
	"fmt"
	"github.com/eldelto/core/internal/util"
	"math"
	"strconv"
)

type Accidental int

const (
	Natural Accidental = 0
	Flat    Accidental = -1
	Sharp   Accidental = 1
)

func (a Accidental) String() string {
	switch a {
	case Natural:
		return ""
	case Flat:
		return "b"
	case Sharp:
		return "#"
	default:
		panic(fmt.Sprintf("invalid accidental '%d'", a))
	}
}

type Note struct {
	value      uint
	octave     uint
	accidental Accidental
	name       string
}

var (
	C Note = Note{value: 0, name: "C"}
	D Note = Note{value: 2, name: "D"}
	E Note = Note{value: 4, name: "E"}
	F Note = Note{value: 5, name: "F"}
	G Note = Note{value: 7, name: "G"}
	A Note = Note{value: 9, name: "A"}
	B Note = Note{value: 11, name: "B"}
)

var baseOctave = []Note{
	C,
	D,
	E,
	F,
	G,
	A,
	B,
}

var baseOctaveReversed = util.ReverseCopy(baseOctave)

func (n Note) totalValue() uint {
	return n.value + uint(n.accidental) + n.octave*(B.value+1)
}

func noteFromValue(baseOctave []Note, value uint, comp func(a, b uint) bool) Note {
	baseValue := value % (B.value + 1)
	octave := value / (B.value + 1)

	newNote := baseOctave[0]
	for _, note := range baseOctave {
		if note.value == baseValue {
			newNote = note
			newNote.octave = octave
			return newNote
		}

		// if note.value > baseValue {
		if comp(note.value, baseValue) {
			break
		}
		newNote = note
	}

	newNote.octave = octave
	newNote.accidental = Accidental(baseValue - newNote.value)
	return newNote
}

func NoteFromValue(value uint) Note {
	return noteFromValue(baseOctave, value, func(a, b uint) bool { return a > b })
}

func (n Note) Accidental(a Accidental) Note {
	n.accidental = a
	return n
}

func (n Note) TransposeSemitone(x int) Note {
	if x >= 0 {
		return n.raiseSemitone(uint(x))
	}

	return n.lowerSemitone(uint(util.AbsI(x)))
}

func AbsI(x int) {
	panic("unimplemented")
}

func (n Note) raiseSemitone(x uint) Note {
	return NoteFromValue(n.totalValue() + x)
}

func (n Note) lowerSemitone(x uint) Note {
	value := n.totalValue() - x
	return noteFromValue(baseOctaveReversed, value, func(a, b uint) bool { return a < b })
}

func (n Note) TransposeOctave(x int) Note {
	o := int(n.octave) + x
	n.octave = uint(util.ClampI(o, 0, math.MaxInt))
	return n
}

func (n Note) Interval(other Note) int {
	return int(other.totalValue()) - int(n.totalValue())
}

func (n Note) ShortName() string {
	return n.name + n.accidental.String()
}

func (n Note) String() string {
	return n.name + n.accidental.String() + strconv.Itoa(int(n.octave))
}
