package main

import (
	"encoding/json"
	"fmt"
	"os"
)

// LinkEntry adalah satu mapping concept_id → permalink, ditulis ke
// concept-links.json untuk dikonsumsi Vue student drawer.
type LinkEntry struct {
	ID        string `json:"id"`
	Permalink string `json:"permalink"`
	File      string `json:"file"`
	StartLine int    `json:"start_line"`
	EndLine   int    `json:"end_line"`
}

// LinksOutput adalah top-level JSON.
type LinksOutput struct {
	GeneratedFromCommit string      `json:"generated_from_commit"`
	RepoSlug            string      `json:"repo_slug"`
	Links               []LinkEntry `json:"links"`
}

func runEmitJSON(cfg *Config, args []string) error {
	markers, err := scanMarkers(cfg)
	if err != nil {
		return err
	}

	out := LinksOutput{
		GeneratedFromCommit: cfg.CommitSHA,
		RepoSlug:            cfg.RepoSlug,
		Links:               make([]LinkEntry, 0, len(markers)),
	}
	for _, m := range markers {
		out.Links = append(out.Links, LinkEntry{
			ID:        m.ConceptID,
			Permalink: cfg.permalink(m.RelPath, m.StartLine, m.EndLine),
			File:      m.RelPath,
			StartLine: m.StartLine,
			EndLine:   m.EndLine,
		})
	}

	w := os.Stdout
	if len(args) > 0 && args[0] != "-" {
		f, err := os.Create(args[0])
		if err != nil {
			return fmt.Errorf("create output: %w", err)
		}
		defer f.Close()
		w = f
	}

	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	if err := enc.Encode(out); err != nil {
		return fmt.Errorf("encode: %w", err)
	}
	return nil
}
