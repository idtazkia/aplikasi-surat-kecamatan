package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

// ConceptPage adalah satu markdown file di docs/concepts/src/.
type ConceptPage struct {
	RelPath  string // relatif ke repo root
	ID       string // dari frontmatter `id:`
	Pending  bool   // dari frontmatter `pending: true` — implementasi belum di-anchor
	Anchors  []string // semua @anchor:<id> yang muncul di body
}

var (
	frontmatterIDRE      = regexp.MustCompile(`(?m)^id:\s*([a-z0-9][a-z0-9-]*)\s*$`)
	frontmatterPendingRE = regexp.MustCompile(`(?m)^pending:\s*true\s*$`)
	anchorRE             = regexp.MustCompile(`@anchor:([a-z0-9][a-z0-9-]*)`)
)

// scanPages baca semua markdown di concepts dir, extract id dan anchors.
func scanPages(cfg *Config) ([]ConceptPage, error) {
	var pages []ConceptPage

	walkErr := filepath.WalkDir(cfg.ConceptsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !cfg.isMarkdownFile(d.Name()) {
			return nil
		}
		// Skip SUMMARY.md dan introduction (tidak punya frontmatter id)
		if d.Name() == "SUMMARY.md" {
			return nil
		}

		rel, err := filepath.Rel(cfg.Root, path)
		if err != nil {
			return err
		}

		page, err := parsePage(path, rel)
		if err != nil {
			return fmt.Errorf("parse %s: %w", rel, err)
		}
		if page.ID == "" {
			// Page tanpa frontmatter id — anggap bukan concept page (mis. introduction.md)
			return nil
		}
		pages = append(pages, *page)
		return nil
	})

	if walkErr != nil {
		return nil, walkErr
	}

	sort.Slice(pages, func(i, j int) bool {
		return pages[i].ID < pages[j].ID
	})

	return pages, nil
}

func parsePage(absPath, relPath string) (*ConceptPage, error) {
	f, err := os.Open(absPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	page := &ConceptPage{RelPath: relPath}

	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	var content strings.Builder
	for scanner.Scan() {
		content.WriteString(scanner.Text())
		content.WriteByte('\n')
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	body := content.String()
	if m := frontmatterIDRE.FindStringSubmatch(body); m != nil {
		page.ID = m[1]
	}
	if frontmatterPendingRE.MatchString(body) {
		page.Pending = true
	}

	for _, m := range anchorRE.FindAllStringSubmatch(body, -1) {
		page.Anchors = append(page.Anchors, m[1])
	}

	return page, nil
}
