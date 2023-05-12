package crdt

import (
	"fmt"
)

type Mergeable interface {
	Identifier() interface{}
	//CanBeMerged(other Mergeable) bool
	Merge(other Mergeable) (Mergeable, error)
}

type ItemMap map[interface{}]Mergeable

type PSet struct {
	LiveSet      ItemMap
	TombstoneSet ItemMap
	identifier   string
}

func NewPSet(identifier string) PSet {
	return PSet{
		LiveSet:      ItemMap{},
		TombstoneSet: ItemMap{},
		identifier:   identifier,
	}
}

func (p *PSet) Add(item Mergeable) error {
	return addToItemMap(p.LiveSet, item)
}

func (p *PSet) Remove(item Mergeable) {
	key := item.Identifier()

	if _, ok := p.LiveView()[key]; !ok {
		return
	}

	p.TombstoneSet[key] = item
}

func (p *PSet) LiveView() ItemMap {
	liveView := ItemMap{}

	for key, value := range p.LiveSet {
		if _, ok := p.TombstoneSet[key]; !ok {
			liveView[key] = value
		}
	}

	return liveView
}

func (p *PSet) Identifier() interface{} {
	return p.identifier
}

func (p *PSet) Merge(other Mergeable) (Mergeable, error) {
	if p.Identifier() != other.Identifier() {
		err := NewCannotBeMergedError(p, other)
		return nil, err
	}

	otherPSet, ok := other.(*PSet)
	if !ok {
		err := NewTypeMisMatchError(p, other)
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

	mergedPSet := PSet{
		LiveSet:      mergedLiveSet,
		TombstoneSet: mergedTombstoneSet,
		identifier:   p.identifier,
	}
	return &mergedPSet, nil
}

func mergeItemMaps(this, other ItemMap) (ItemMap, error) {
	mergedItemMap := ItemMap{}
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

func addToItemMap(itemMap ItemMap, item Mergeable) error {
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

	itemMap[key] = mergedItem
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

func NewTypeMisMatchError(this, other Mergeable) *TypeMisMatchError {
	return &TypeMisMatchError{
		this:    this,
		other:   other,
		message: fmt.Sprintf("item with type '%t' cannot be merged with item with type '%t'", this, other),
	}
}

func (e *TypeMisMatchError) Error() string {
	return e.message
}
