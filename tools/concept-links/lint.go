package main

import (
	"fmt"
	"os"
	"sort"
)

// runLint detect:
//
//   - @anchor:<id> di markdown tapi tidak ada marker concept:<id>:start|end
//   - marker concept:<id> di source tapi tidak ada concept page dengan id:<id>
//   - concept page dengan id:<id> tapi tidak ada marker
//
// Exit non-zero kalau ada problem (CI gate).
func runLint(cfg *Config) error {
	markers, err := scanMarkers(cfg)
	if err != nil {
		return err
	}
	pages, err := scanPages(cfg)
	if err != nil {
		return err
	}

	markerIDs := map[string]Marker{}
	for _, m := range markers {
		markerIDs[m.ConceptID] = m
	}
	pageIDs := map[string]ConceptPage{}
	for _, p := range pages {
		pageIDs[p.ID] = p
	}

	var issues []string

	// Marker tanpa concept page
	for id, m := range markerIDs {
		if _, ok := pageIDs[id]; !ok {
			issues = append(issues, fmt.Sprintf(
				"[orphan-marker]   concept:%s di %s:%d-%d tapi tidak ada concept page",
				id, m.RelPath, m.StartLine, m.EndLine))
		}
	}

	// Concept page tanpa marker (kecuali pending: true di frontmatter — overview
	// page yang implementasinya akan di-anchor di fase berikutnya).
	for id, p := range pageIDs {
		if p.Pending {
			continue
		}
		if _, ok := markerIDs[id]; !ok {
			issues = append(issues, fmt.Sprintf(
				"[orphan-page]     concept page id:%s (%s) tapi tidak ada marker di source",
				id, p.RelPath))
		}
	}

	// Anchor tanpa marker (skip kalau page pending — masih placeholder)
	for _, p := range pages {
		if p.Pending {
			continue
		}
		for _, a := range p.Anchors {
			if _, ok := markerIDs[a]; !ok {
				issues = append(issues, fmt.Sprintf(
					"[orphan-anchor]   @anchor:%s di %s tapi tidak ada marker di source",
					a, p.RelPath))
			}
		}
	}

	sort.Strings(issues)
	for _, msg := range issues {
		fmt.Fprintln(os.Stderr, msg)
	}

	fmt.Fprintf(os.Stderr, "\nstats: %d marker, %d page, %d issue\n", len(markers), len(pages), len(issues))

	if len(issues) > 0 {
		return fmt.Errorf("%d concept-link issue(s) — fix sebelum merge", len(issues))
	}
	return nil
}
