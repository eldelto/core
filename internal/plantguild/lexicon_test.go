package plantguild_test

import (
	"os"
	"sort"
	"testing"

	"github.com/eldelto/core/internal/plantguild"
	"gopkg.in/yaml.v3"
)

func TestLexiconConsistency(t *testing.T) {
	if sanitizeLexicon(plantguild.EmbeddedLexicon) {
		file, err := os.Create("plants-corrected.yml")
		if err != nil {
			t.Fatalf("failed to create plants-corrected.yml: %s", err)
		}
		defer file.Close()

		listing := plantguild.EmbeddedLexicon.Listing()
		entries := listing.Entries
		sort.Slice(entries, func(i, j int) bool {
			return entries[i].Name < entries[j].Name
		})
		if err := yaml.NewEncoder(file).Encode(listing); err != nil {
			t.Fatalf("failed to serialize to plants-corrected.yml: %s", err)
		}
		t.Fatal("lexicon is not consistent - compare content of plants-corrected.yml with original content")
	}
}

func hasEntry(l *plantguild.Lexicon, plantName string) bool {
	_, ok := l.Entries[plantName]
	return ok
}

func addMissingEntries(l *plantguild.Lexicon, plantNames []string) (modified bool) {
	for _, plantName := range plantNames {
		if !hasEntry(l, plantName) {
			l.Entries[plantName] = plantguild.Info{Name: plantName}
			modified = true
		}
	}

	return modified
}

func sanitizeEntries(l *plantguild.Lexicon) (modified bool) {
	for _, entry := range l.Entries {
		modified = addMissingEntries(l, entry.GoodCompanions) || modified
		modified = addMissingEntries(l, entry.BadCompanions) || modified
	}

	return modified
}

func appendCompanion(companions *[]string, plantName string) bool {
	for _, companion := range *companions {
		if companion == plantName {
			return false
		}
	}
	*companions = append(*companions, plantName)
	return true
}

func addMissingCompanions(l *plantguild.Lexicon, entry *plantguild.Info) (modified bool) {
	for _, companion := range entry.GoodCompanions {
		companionEntry := l.Entries[companion]
		modified = appendCompanion(&companionEntry.GoodCompanions, entry.Name) || modified
		l.Entries[companion] = companionEntry
	}

	for _, companion := range entry.BadCompanions {
		companionEntry := l.Entries[companion]
		modified = appendCompanion(&companionEntry.BadCompanions, entry.Name) || modified
		l.Entries[companion] = companionEntry
	}

	return modified
}

func sanitizeCompanions(l *plantguild.Lexicon) (modified bool) {
	for _, entry := range l.Entries {
		modified = addMissingCompanions(l, &entry) || modified
	}

	return modified
}

func sanitizeLexicon(l *plantguild.Lexicon) (modified bool) {
	modified = sanitizeEntries(l) || modified
	modified = sanitizeCompanions(l) || modified

	return modified
}
