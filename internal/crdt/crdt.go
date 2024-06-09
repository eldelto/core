package crdt

import (
	"fmt"
)

type Mergeable interface {
	Identifier() string
	//CanBeMerged(other Mergeable) bool
	Merge(other Mergeable) (Mergeable, error)
}

type PSet[M Mergeable] struct {
	LiveSet      map[string]M
	TombstoneSet map[string]M
	identifier   string
}

func NewPSet[M Mergeable](identifier string) PSet[M] {
	return PSet[M]{
		LiveSet:      map[string]M{},
		TombstoneSet: map[string]M{},
		identifier:   identifier,
	}
}

func (p *PSet[M]) Add(item M) error {
	return addToItemMap(p.LiveSet, item)
}

func (p *PSet[M]) Remove(item M) {
	key := item.Identifier()

	if _, ok := p.LiveView()[key]; !ok {
		return
	}

	p.TombstoneSet[key] = item
}

func (p *PSet[M]) LiveView() map[string]M {
	liveView := map[string]M{}

	for key, value := range p.LiveSet {
		if _, ok := p.TombstoneSet[key]; !ok {
			liveView[key] = value
		}
	}

	return liveView
}

func (p *PSet[M]) Identifier() string {
	return p.identifier
}

func (p *PSet[M]) Merge(other Mergeable) (Mergeable, error) {
	// Weird-ass hack to prevent uninitialized nested PSets from
	// blocking the merge.
	if p.Identifier() == "" {
		return other, nil
	}
	if other.Identifier() == "" {
		return p, nil
	}

	if p.Identifier() != other.Identifier() {
		err := NewCannotBeMergedError(p, other)
		return nil, err
	}

	otherPSet, ok := other.(*PSet[M])
	if !ok {
		err := NewTypeMismatchError(p, other)
		return nil, err
	}

	mergedLiveSet, err := mergeItemMaps(p.LiveSet, otherPSet.LiveSet)
	if err != nil {
		return nil, err
	}

	mergedTombstoneSet, err := mergeItemMaps(p.TombstoneSet, otherPSet.TombstoneSet)
	if err != nil {
		return nil, err
	}

	mergedPSet := PSet[M]{
		LiveSet:      mergedLiveSet,
		TombstoneSet: mergedTombstoneSet,
		identifier:   p.identifier,
	}
	return &mergedPSet, nil
}

func mergeItemMaps[M Mergeable](this, other map[string]M) (map[string]M, error) {
	mergedItemMap := map[string]M{}
	for key, value := range this {
		mergedItemMap[key] = value
	}

	for _, value := range other {
		err := addToItemMap(mergedItemMap, value)
		if err != nil {
			return nil, err
		}
	}

	return mergedItemMap, nil
}

func addToItemMap[M Mergeable](itemMap map[string]M, item M) error {
	key := item.Identifier()

	oldItem, ok := itemMap[key]
	if !ok {
		itemMap[key] = item
		return nil
	}

	mergedItem, err := oldItem.Merge(item)
	if err != nil {
		return err
	}

	itemMap[key] = mergedItem.(M)
	return nil
}

// CannotBeMergedError indicates that two entities cannot be merged
// (e.g. IDs do not match)
type CannotBeMergedError struct {
	this    Mergeable
	other   Mergeable
	message string
}

func NewCannotBeMergedError(this, other Mergeable) *CannotBeMergedError {
	return &CannotBeMergedError{
		this:    this,
		other:   other,
		message: fmt.Sprintf("item with ID '%v' cannot be merged with item with ID '%v'", this.Identifier(), other.Identifier()),
	}
}

func (e *CannotBeMergedError) Error() string {
	return e.message
}

type TypeMisMatchError struct {
	this    Mergeable
	other   Mergeable
	message string
}

func NewTypeMismatchError(this, other Mergeable) *TypeMisMatchError {
	return &TypeMisMatchError{
		this:    this,
		other:   other,
		message: fmt.Sprintf("item with type '%t' cannot be merged with item with type '%t'", this, other),
	}
}

func (e *TypeMisMatchError) Error() string {
	return e.message
}
