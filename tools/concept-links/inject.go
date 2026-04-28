package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// runInject scan semua concept page markdown, replace `@anchor:<id>` dengan
// `[file:lines](permalink)` markdown link.
//
// In-place edit. Idempotent: replace pakai regex yang match `@anchor:<id>`,
// dan setelah replace teks `@anchor:<id>` sudah hilang. Re-run aman.
//
// Convention: `@anchor:<id>` boleh muncul di mana saja di body markdown.
// Hasil replace = `[file.go:42-58](https://github.com/...)`.
func runInject(cfg *Config) error {
	markers, err := scanMarkers(cfg)
	if err != nil {
		return err
	}
	markerByID := map[string]Marker{}
	for _, m := range markers {
		markerByID[m.ConceptID] = m
	}

	pages, err := scanPages(cfg)
	if err != nil {
		return err
	}

	injected := 0
	missing := []string{}

	walkErr := filepath.WalkDir(cfg.ConceptsDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() || !cfg.isMarkdownFile(d.Name()) {
			return nil
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		original := string(raw)

		updated := anchorRE.ReplaceAllStringFunc(original, func(match string) string {
			id := strings.TrimPrefix(match, "@anchor:")
			m, ok := markerByID[id]
			if !ok {
				missing = append(missing, fmt.Sprintf("%s -> @anchor:%s (no marker)", d.Name(), id))
				return match // biarkan untuk lint detect
			}
			injected++
			label := fmt.Sprintf("%s:%d-%d", m.RelPath, m.StartLine, m.EndLine)
			return fmt.Sprintf("[%s](%s)", label, cfg.permalink(m.RelPath, m.StartLine, m.EndLine))
		})

		if updated != original {
			if err := os.WriteFile(path, []byte(updated), 0o644); err != nil {
				return err
			}
		}
		return nil
	})

	if walkErr != nil {
		return walkErr
	}

	fmt.Fprintf(os.Stderr, "inject: %d anchor(s) di-resolve, %d page(s) di-scan\n", injected, len(pages))
	if len(missing) > 0 {
		fmt.Fprintf(os.Stderr, "warning: %d anchor tanpa marker (jalankan `lint` untuk detail)\n", len(missing))
	}
	return nil
}
