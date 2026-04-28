package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
)

type Config struct {
	Root          string // repo root absolute path
	RepoSlug      string // "owner/repo" (e.g. idtazkia/aplikasi-surat-kecamatan)
	CommitSHA     string
	ConceptsDir   string // <root>/docs/concepts/src
	IgnoreDirs    []string
	SourceExts    []string // file extensions to scan for markers
	Markdown      []string // markdown extensions for concept pages
}

func loadConfig(root string) (*Config, error) {
	root, err := filepath.Abs(root)
	if err != nil {
		return nil, err
	}

	slug, err := repoSlug(root)
	if err != nil {
		return nil, err
	}

	sha, err := commitSHA(root)
	if err != nil {
		return nil, err
	}

	conceptsDir := filepath.Join(root, "docs", "concepts", "src")
	if _, err := os.Stat(conceptsDir); err != nil {
		return nil, fmt.Errorf("concepts dir tidak ditemukan: %s", conceptsDir)
	}

	return &Config{
		Root:        root,
		RepoSlug:    slug,
		CommitSHA:   sha,
		ConceptsDir: conceptsDir,
		IgnoreDirs:  []string{".git", "node_modules", "vendor", "book", "dist", "bin"},
		SourceExts:  []string{".go", ".sql", ".vue", ".ts", ".tsx", ".js", ".py"},
		Markdown:    []string{".md"},
	}, nil
}

// repoSlug extract "owner/repo" dari git remote 'origin'. Override via env.
func repoSlug(root string) (string, error) {
	if env := os.Getenv("CONCEPT_LINK_REPO"); env != "" {
		return env, nil
	}

	cmd := exec.Command("git", "config", "--get", "remote.origin.url")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git remote: %w", err)
	}
	url := strings.TrimSpace(string(out))

	// Support format: git@github.com:owner/repo.git OR https://github.com/owner/repo.git
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`git@github\.com:([^/]+/[^.]+?)(\.git)?$`),
		regexp.MustCompile(`https://github\.com/([^/]+/[^.]+?)(\.git)?$`),
	}
	for _, p := range patterns {
		if m := p.FindStringSubmatch(url); m != nil {
			return m[1], nil
		}
	}
	return "", fmt.Errorf("tidak bisa extract repo slug dari %q (set CONCEPT_LINK_REPO)", url)
}

func commitSHA(root string) (string, error) {
	if env := os.Getenv("CONCEPT_LINK_SHA"); env != "" {
		return env, nil
	}
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = root
	out, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("git rev-parse: %w", err)
	}
	return strings.TrimSpace(string(out)), nil
}

func (c *Config) permalink(relPath string, startLine, endLine int) string {
	if startLine == endLine {
		return fmt.Sprintf("https://github.com/%s/blob/%s/%s#L%d",
			c.RepoSlug, c.CommitSHA, relPath, startLine)
	}
	return fmt.Sprintf("https://github.com/%s/blob/%s/%s#L%d-L%d",
		c.RepoSlug, c.CommitSHA, relPath, startLine, endLine)
}

func (c *Config) shouldSkipDir(name string) bool {
	for _, d := range c.IgnoreDirs {
		if name == d {
			return true
		}
	}
	return false
}

func (c *Config) isSourceFile(name string) bool {
	ext := filepath.Ext(name)
	for _, e := range c.SourceExts {
		if e == ext {
			return true
		}
	}
	return false
}

func (c *Config) isMarkdownFile(name string) bool {
	ext := filepath.Ext(name)
	for _, e := range c.Markdown {
		if e == ext {
			return true
		}
	}
	return false
}
