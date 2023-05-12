package plantguild

import (
	_ "embed"
	"fmt"

	"gopkg.in/yaml.v3"
)

type Info struct {
	Name           string   `yaml:"name"`
	Height         uint     `yaml:"height"`
	Width          uint     `yaml:"width"`
	Family         string   `yaml:"family"`
	FeederType     string   `yaml:"feederType"`
	RootType       string   `yaml:"rootType"`
	GoodCompanions []string `yaml:"goodCompanions"`
	BadCompanions  []string `yaml:"badCompanions"`
}

type LexiconListing struct {
	Entries []Info `yaml:"plants"`
}

func (l *LexiconListing) Lexicon() *Lexicon {
	lexicon := Lexicon{
		Entries: map[string]Info{},
	}
	for _, entry := range l.Entries {
		lexicon.Entries[entry.Name] = entry
	}

	return &lexicon
}

type Lexicon struct {
	Entries map[string]Info
}

func (l *Lexicon) Listing() *LexiconListing {
	listing := LexiconListing{
		Entries: make([]Info, len(l.Entries)),
	}
	i := 0
	for _, entry := range l.Entries {
		listing.Entries[i] = entry
		i++
	}

	return &listing
}

//go:embed plants.yml
var rawPlantInfo string
var EmbeddedLexicon *Lexicon = &Lexicon{}

func init() {
	listing := LexiconListing{}
	if err := yaml.Unmarshal([]byte(rawPlantInfo), &listing); err != nil {
		panic(fmt.Errorf("failed to parse plant infos: %w", err))
	}
	EmbeddedLexicon = listing.Lexicon()

	// Make the raw data reclaimable by the GC.
	rawPlantInfo = ""
}
