package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
)

// Marker mendeskripsikan satu konsep yang di-tag di source code.
type Marker struct {
	ConceptID string
	RelPath   string // path relatif ke repo root
	StartLine int    // line number marker `start` (1-indexed)
	EndLine   int    // line number marker `end`
}

var (
	startRE = regexp.MustCompile(`concept:([a-z0-9][a-z0-9-]*):start\b`)
	endRE   = regexp.MustCompile(`concept:([a-z0-9][a-z0-9-]*):end\b`)
)

// scanMarkers walk repo source files, extract semua marker, return sorted by ID.
//
// Error kalau:
//   - start tanpa pasangan end (atau sebaliknya)
//   - ID yang sama muncul lebih dari satu kali (start atau end ganda)
//   - end muncul sebelum start
func scanMarkers(cfg *Config) ([]Marker, error) {
	starts := map[string]struct {
		path string
		line int
	}{}
	ends := map[string]struct {
		path string
		line int
	}{}

	walkErr := filepath.WalkDir(cfg.Root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if cfg.shouldSkipDir(d.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		if !cfg.isSourceFile(d.Name()) {
			return nil
		}

		rel, err := filepath.Rel(cfg.Root, path)
		if err != nil {
			return err
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		scanner := bufio.NewScanner(f)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		lineNo := 0
		for scanner.Scan() {
			lineNo++
			line := scanner.Text()
			if m := startRE.FindStringSubmatch(line); m != nil {
				id := m[1]
				if existing, ok := starts[id]; ok {
					return fmt.Errorf("duplicate concept:%s:start di %s:%d (sebelumnya %s:%d)",
						id, rel, lineNo, existing.path, existing.line)
				}
				starts[id] = struct {
					path string
					line int
				}{rel, lineNo}
			}
			if m := endRE.FindStringSubmatch(line); m != nil {
				id := m[1]
				if existing, ok := ends[id]; ok {
					return fmt.Errorf("duplicate concept:%s:end di %s:%d (sebelumnya %s:%d)",
						id, rel, lineNo, existing.path, existing.line)
				}
				ends[id] = struct {
					path string
					line int
				}{rel, lineNo}
			}
		}
		return scanner.Err()
	})

	if walkErr != nil {
		return nil, walkErr
	}

	var markers []Marker
	for id, s := range starts {
		e, ok := ends[id]
		if !ok {
			return nil, fmt.Errorf("concept:%s:start tanpa pasangan end (di %s:%d)", id, s.path, s.line)
		}
		if e.path != s.path {
			return nil, fmt.Errorf("concept:%s start di %s:%d, end di %s:%d (harus file yang sama)",
				id, s.path, s.line, e.path, e.line)
		}
		if e.line <= s.line {
			return nil, fmt.Errorf("concept:%s:end (line %d) harus setelah :start (line %d) di %s",
				id, e.line, s.line, s.path)
		}
		markers = append(markers, Marker{
			ConceptID: id,
			RelPath:   s.path,
			StartLine: s.line,
			EndLine:   e.line,
		})
	}

	for id, e := range ends {
		if _, ok := starts[id]; !ok {
			return nil, fmt.Errorf("concept:%s:end tanpa pasangan start (di %s:%d)", id, e.path, e.line)
		}
	}

	sort.Slice(markers, func(i, j int) bool {
		return markers[i].ConceptID < markers[j].ConceptID
	})

	return markers, nil
}

func runScan(cfg *Config) error {
	markers, err := scanMarkers(cfg)
	if err != nil {
		return err
	}
	for _, m := range markers {
		fmt.Printf("%-40s  %s:%d-%d\n", m.ConceptID, m.RelPath, m.StartLine, m.EndLine)
	}
	fmt.Printf("\n%d marker(s) ditemukan\n", len(markers))
	return nil
}
