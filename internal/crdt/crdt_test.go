package crdt

import (
	"testing"

	. "github.com/eldelto/core/internal/testutils"
)

const psetID0 = "pset0"
const psetID1 = "pset1"

const mergeableID0 = "mergeable0"
const mergeableID1 = "mergeable1"

var mergeable0 = testMergeable{
	value:      1,
	identifier: mergeableID0,
}

var mergeable1 = testMergeable{
	value:      2,
	identifier: mergeableID1,
}

var invalidMergeable = testMergeable{
	value:      13,
	identifier: mergeableID0,
}

func TestAdd(t *testing.T) {
	pset := NewPSet[*testMergeable](psetID0)

	err := pset.Add(&mergeable0)
	AssertEquals(t, nil, err, "pset.Add error")

	pset.Add(&mergeable0)
	pset.Add(&mergeable1)

	mergedMergeable := mergeable0
	mergedMergeable.value = 2
	expected := map[string]*testMergeable{
		mergeable0.Identifier(): &mergedMergeable,
		mergeable1.Identifier(): &mergeable1,
	}
	AssertEquals(t, expected, pset.LiveSet, "pset.LiveSet")
}

func TestInvalidAdd(t *testing.T) {
	pset := NewPSet[*testMergeable](psetID0)

	err := pset.Add(&mergeable0)
	AssertEquals(t, nil, err, "pset.Add error")

	err = pset.Add(&invalidMergeable)
	expectedErr := NewCannotBeMergedError(&mergeable0, &invalidMergeable)
	AssertEquals(t, expectedErr, err, "pset.Add error")

	expected := map[string]*testMergeable{
		mergeable0.Identifier(): &mergeable0,
	}
	AssertEquals(t, expected, pset.LiveSet, "pset.LiveSet")
}

func TestRemove(t *testing.T) {
	pset := NewPSet[*testMergeable](psetID0)

	pset.Add(&mergeable0)
	pset.Remove(&mergeable0)
	pset.Remove(&mergeable1)

	expected := map[string]*testMergeable{
		mergeable0.Identifier(): &mergeable0,
	}
	AssertEquals(t, expected, pset.TombstoneSet, "pset.TombstoneSet")
}

func TestLiveView(t *testing.T) {
	pset := NewPSet[*testMergeable](psetID0)

	pset.Add(&mergeable0)
	pset.Add(&mergeable1)
	pset.Remove(&mergeable0)

	expected := map[string]*testMergeable{
		mergeable1.Identifier(): &mergeable1,
	}
	AssertEquals(t, expected, pset.LiveView(), "pset.LiveView")
}

func TestIdentifier(t *testing.T) {
	pset := NewPSet[*testMergeable](psetID0)

	AssertEquals(t, psetID0, pset.Identifier(), "pset.Identifier")
}

func TestMerge(t *testing.T) {
	pset0 := NewPSet[*testMergeable](psetID0)
	pset0.Add(&mergeable0)
	pset0.Add(&mergeable1)

	pset1 := NewPSet[*testMergeable](psetID0)
	pset1.Add(&mergeable0)
	pset1.Remove(&mergeable0)

	mergedPSet, err := pset0.Merge(&pset1)
	AssertEquals(t, nil, err, "pset0.Merge error")

	expected := map[string]*testMergeable{
		mergeable1.Identifier(): &mergeable1,
	}
	AssertEquals(t, expected, mergedPSet.(*PSet[*testMergeable]).LiveView(), "pset.LiveView")
}

func TestInvalidIdentifierMerge(t *testing.T) {
	pset0 := NewPSet[*testMergeable](psetID0)
	pset1 := NewPSet[*testMergeable](psetID1)

	_, err := pset0.Merge(&pset1)

	expected := NewCannotBeMergedError(&pset0, &pset1)
	AssertEquals(t, expected, err, "pset0.Merge error")
}

func TestMergeError(t *testing.T) {
	pset0 := NewPSet[*testMergeable](psetID0)
	pset0.Add(&mergeable0)

	pset1 := NewPSet[*testMergeable](psetID0)
	pset1.Add(&invalidMergeable)

	_, err := pset0.Merge(&pset1)

	expected := NewCannotBeMergedError(&mergeable0, &invalidMergeable)
	AssertEquals(t, expected, err, "pset0.Merge error")
}

type testMergeable struct {
	value      int
	identifier string
}

func (t *testMergeable) Identifier() string {
	return t.identifier
}

func (t *testMergeable) Merge(other Mergeable) (Mergeable, error) {
	if t.Identifier() != other.Identifier() {
		err := NewCannotBeMergedError(t, other)
		return nil, err
	}

	// Produce an error on a specific value for testing
	if t.value == 13 || other.(*testMergeable).value == 13 {
		err := NewCannotBeMergedError(t, other)
		return nil, err
	}

	merged := testMergeable{
		value:      t.value + other.(*testMergeable).value,
		identifier: t.identifier,
	}

	return &merged, nil
}
