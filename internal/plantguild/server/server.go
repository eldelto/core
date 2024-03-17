package server

import (
	"bytes"
	"embed"
	"encoding/json"
	"fmt"

	"github.com/eldelto/core/internal/plantguild"
)

//go:embed assets
var AssetsFS embed.FS

//go:embed templates
var TemplatesFS embed.FS

type TemplateData struct{}

func (td *TemplateData) PlantLexiconJSON() (string, error) {
	buffer := bytes.Buffer{}
	if err := json.NewEncoder(&buffer).Encode(plantguild.EmbeddedLexicon.Entries); err != nil {
		return "", fmt.Errorf("failed to encode plant lexicon: %w", err)
	}

	return buffer.String(), nil
}
