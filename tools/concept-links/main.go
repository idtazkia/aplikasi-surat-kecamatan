// concept-links scan source code untuk marker `concept:<id>:start|end`,
// resolve ke (file, line range, commit SHA), generate GitHub permalink.
//
// Sub-commands:
//
//	scan      — print mapping marker → file:lines (debugging)
//	inject    — replace `@anchor:<id>` di markdown concept page dengan permalink
//	emit-json — emit concept-links.json untuk Vue student drawer
//	lint      — orphan detection (anchor tanpa marker, marker tanpa concept page)
//
// Source-of-truth marker dan concept page wajib match. CI gate via `lint`.
package main

import (
	"fmt"
	"os"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	cmd := os.Args[1]
	args := os.Args[2:]

	root, err := os.Getwd()
	if err != nil {
		fatal(err)
	}

	cfg, err := loadConfig(root)
	if err != nil {
		fatal(err)
	}

	switch cmd {
	case "scan":
		err = runScan(cfg)
	case "inject":
		err = runInject(cfg)
	case "emit-json":
		err = runEmitJSON(cfg, args)
	case "lint":
		err = runLint(cfg)
	case "help", "-h", "--help":
		usage()
		return
	default:
		fmt.Fprintf(os.Stderr, "unknown subcommand: %s\n\n", cmd)
		usage()
		os.Exit(2)
	}

	if err != nil {
		fatal(err)
	}
}

func usage() {
	fmt.Fprintf(os.Stderr, `concept-links — concept catalog tooling

Usage:
  concept-links scan       # debug: list semua marker yang ditemukan
  concept-links inject     # replace @anchor:<id> di markdown concept pages
  concept-links emit-json  # emit concept-links.json ke stdout
  concept-links lint       # orphan detection (CI gate)

Repository config:
  Repo URL diambil dari git remote 'origin' (atau env CONCEPT_LINK_REPO).
  Commit SHA dari git rev-parse HEAD.
  Concept pages di docs/concepts/src/.
  Source markers di-scan recursive (skip .git, node_modules, vendor, book/).
`)
}

func fatal(err error) {
	fmt.Fprintln(os.Stderr, "error:", err)
	os.Exit(1)
}
