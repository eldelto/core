package worklog

import "time"

type StubSink struct{}

func (s *StubSink) Name() string {
	return "Stub-Sink"
}

func (s StubSink) ValidIdentifier(identifier string) bool {
	return identifier == "stub"
}

func (s *StubSink) FetchEntries(start, end time.Time) ([]Entry, error) {
	return []Entry{}, nil
}

func (s *StubSink) IsApplicable(e Entry) bool {
	return true
}

func (s *StubSink) ProcessActions(actions []Action, localEntries []Entry) error {
	return nil
}
