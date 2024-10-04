package worklog

import (
	"fmt"
	"slices"
	"time"

	"github.com/eldelto/core/internal/cli"
)

type Entry struct {
	Ticket     string
	ExternalID string
	From       time.Time
	To         time.Time
}

func (e *Entry) Duration() time.Duration {
	return e.To.Sub(e.From)
}

func groupByDay(actions []Action) map[time.Time][]Action {
	groups := map[time.Time][]Action{}
	for _, a := range actions {
		y, m, d := a.Entry.From.Date()
		day := time.Date(y, m, d, 0, 0, 0, 0, time.Local)
		groups[day] = append(groups[day], a)
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

func generateActions(local, remote []Entry) []Action {
	actions := []Action{}
	for _, l := range local {
		if !slices.ContainsFunc(remote, entryEqual(l)) {
			actions = append(actions, Action{Entry: l, Operation: Add})
		}
	}
	for _, r := range remote {
		if !slices.ContainsFunc(local, entryEqual(r)) {
			actions = append(actions, Action{Entry: r, Operation: Remove})
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
	Handle(a Action) error
}

func DryRun(source Source, sinks []Sink, start, end time.Time) error {
	localEntries, err := source.FetchEntries(start, end)
	if err != nil {
		fmt.Errorf("fetch entries from source %q: %w", source.Name(), err)
	}

	for _, sink := range sinks {
		fmt.Println(cli.Brown("Syncing " + sink.Name()))

		remoteEntries, err := sink.FetchEntries(start, end)
		if err != nil {
			fmt.Errorf("fetch entries from sink %q: %w", source.Name(), err)
		}

		actions := generateActions(localEntries, remoteEntries)
		PrettyPrintActions(groupByDay(actions))

		dryRun := true
		if !dryRun {
			for _, a := range actions {
				if err := sink.Handle(a); err != nil {
					fmt.Errorf("handle action via sink %q: %w", sink.Name(), err)
				}
			}
		}
	}

	return nil
}
