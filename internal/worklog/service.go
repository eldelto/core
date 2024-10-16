package worklog

import (
	"fmt"
	"slices"
	"time"

	"github.com/eldelto/core/internal/cli"
)

type EntryType uint

const (
	EntryTypeWork EntryType = iota
	EntryTypeBreak
)

type Entry struct {
	Ticket     string
	ExternalID string
	Type       EntryType
	From       time.Time
	To         time.Time
}

func (e *Entry) Duration() time.Duration {
	return e.To.Sub(e.From)
}

func groupActionsByDay(actions []Action) map[time.Time][]Action {
	groups := map[time.Time][]Action{}
	for _, a := range actions {
		y, m, d := a.Entry.From.Date()
		day := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
		groups[day] = append(groups[day], a)
	}

	return groups
}

func groupEntriesByDay(entries []Entry) map[time.Time][]Entry {
	groups := map[time.Time][]Entry{}
	for _, e := range entries {
		y, m, d := e.From.Date()
		day := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
		groups[day] = append(groups[day], e)
	}

	return groups
}

type Operation uint

const (
	Add = Operation(iota)
	Remove
)

func (o Operation) String() string {
	switch o {
	case Add:
		return "add"
	case Remove:
		return "remove"
	default:
		return "unknown"
	}
}

type Action struct {
	Entry     Entry
	Operation Operation
}

func entryEqual(this Entry) func(Entry) bool {
	return func(other Entry) bool {
		return this.Ticket == other.Ticket && this.From.Equal(other.From) && this.To.Equal(other.To)
	}
}

func generateActions(local, remote []Entry, sink Sink) []Action {
	actions := []Action{}
	for _, r := range remote {
		if !sink.IsApplicable(r) {
			continue
		}
		if !slices.ContainsFunc(local, entryEqual(r)) {
			actions = append(actions, Action{Entry: r, Operation: Remove})
		}
	}
	for _, l := range local {
		if !sink.IsApplicable(l) {
			continue
		}
		if !slices.ContainsFunc(remote, entryEqual(l)) {
			actions = append(actions, Action{Entry: l, Operation: Add})
		}
	}

	return actions
}

type Source interface {
	Name() string
	FetchEntries(start, end time.Time) ([]Entry, error)
}

type Sink interface {
	Source
	IsApplicable(e Entry) bool
	ProcessActions(actions []Action, localEntries []Entry) error
}

func Sync(source Source, sinks []Sink, start, end time.Time, dryRun bool) error {
	localEntries, err := source.FetchEntries(start, end)
	if err != nil {
		return fmt.Errorf("fetch entries from source %q: %w", source.Name(), err)
	}

	for _, sink := range sinks {
		fmt.Println(cli.Brown("Syncing " + sink.Name()))
		fmt.Println()

		remoteEntries, err := sink.FetchEntries(start, end)
		if err != nil {
			return fmt.Errorf("fetch entries from sink %q: %w", sink.Name(), err)
		}

		actions := generateActions(localEntries, remoteEntries, sink)
		PrettyPrintActions(groupActionsByDay(actions))

		if dryRun || len(actions) < 1 {
			continue
		}

		approveSync, err := cli.ReadYesNo("Continue syncing?")
		if err != nil {
			return fmt.Errorf("approve syncing: %w", err)
		}

		if approveSync {
			if err := sink.ProcessActions(actions, localEntries); err != nil {
				return fmt.Errorf("handle action via sink %q: %w", sink.Name(), err)
			}
		}
	}

	return nil
}
