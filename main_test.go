package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestInstallDirContentsFromSourceSkipsExistingDestination(t *testing.T) {
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "skill.txt"), []byte("new content"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	targetRoot := t.TempDir()
	destination := filepath.Join(targetRoot, "existing-skill")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatalf("create destination: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destination, "skill.txt"), []byte("existing content"), 0o644); err != nil {
		t.Fatalf("write destination file: %v", err)
	}

	err := installDirContentsFromSource(sourceDir, targetRoot, "existing-skill", false, installOptions{skipExisting: true})
	if err != nil {
		t.Fatalf("installDirContentsFromSource returned error: %v", err)
	}

	contents, err := os.ReadFile(filepath.Join(destination, "skill.txt"))
	if err != nil {
		t.Fatalf("read destination file: %v", err)
	}
	if string(contents) != "existing content" {
		t.Fatalf("destination was overwritten: got %q", string(contents))
	}

	if _, err := os.Stat(filepath.Join(destination, "SKILL.md")); !os.IsNotExist(err) {
		t.Fatalf("unexpected files copied while skipping: %v", err)
	}
}

func TestInstallDirContentsFromSourceReplacesExistingDestinationByDefault(t *testing.T) {
	sourceDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(sourceDir, "skill.txt"), []byte("new content"), 0o644); err != nil {
		t.Fatalf("write source file: %v", err)
	}

	targetRoot := t.TempDir()
	destination := filepath.Join(targetRoot, "existing-skill")
	if err := os.MkdirAll(destination, 0o755); err != nil {
		t.Fatalf("create destination: %v", err)
	}
	if err := os.WriteFile(filepath.Join(destination, "skill.txt"), []byte("existing content"), 0o644); err != nil {
		t.Fatalf("write destination file: %v", err)
	}

	err := installDirContentsFromSource(sourceDir, targetRoot, "existing-skill", false, installOptions{})
	if err != nil {
		t.Fatalf("installDirContentsFromSource returned error: %v", err)
	}

	contents, err := os.ReadFile(filepath.Join(destination, "skill.txt"))
	if err != nil {
		t.Fatalf("read destination file: %v", err)
	}
	if string(contents) != "new content" {
		t.Fatalf("destination was not replaced: got %q", string(contents))
	}
}

func TestParseGitSourceRejectsNonRepoArtifacts(t *testing.T) {
	t.Parallel()

	testCases := []string{
		"https://example.com/archive.zip",
		"https://example.com/skills/SKILL.md",
		"https://example.com/docs/readme.md",
	}

	for _, rawURL := range testCases {
		rawURL := rawURL
		t.Run(rawURL, func(t *testing.T) {
			t.Parallel()

			if _, err := parseGitSource(context.Background(), rawURL); err == nil {
				t.Fatalf("parseGitSource(%q) succeeded unexpectedly", rawURL)
			}
		})
	}
}

func TestParseGitSourceAcceptsSSHCloneURL(t *testing.T) {
	t.Parallel()

	source, err := parseGitSource(context.Background(), "git@github.com:owner/repo.git")
	if err != nil {
		t.Fatalf("parseGitSource returned error: %v", err)
	}

	if source.CloneURL != "git@github.com:owner/repo.git" {
		t.Fatalf("unexpected clone URL: %q", source.CloneURL)
	}
}
