package main

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
)

type installOptions struct {
	skipExisting bool
}

func ensureGitAvailable() error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("required command not found: git")
	}

	return nil
}

func installFromGitRepoURL(
	ctx context.Context,
	rawURL string,
	targetRoot string,
	aliasName string,
	version string,
	options installOptions,
) error {
	source, err := parseGitSource(ctx, rawURL)
	if err != nil {
		return err
	}
	if version != "" {
		source.Ref = version
	}

	tmpRoot, err := os.MkdirTemp("", "codex-skill-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpRoot)

	repoDir := filepath.Join(tmpRoot, "repo")
	cloneArgs := []string{"clone", "--depth", "1"}
	if source.Ref != "" && !looksLikeCommitHash(source.Ref) {
		cloneArgs = append(cloneArgs, "--branch", source.Ref)
	}
	cloneArgs = append(cloneArgs, source.CloneURL, repoDir)
	if err := runGitCommand(ctx, cloneArgs...); err != nil {
		return fmt.Errorf("clone git repository %q: %w", source.CloneURL, err)
	}
	if source.Ref != "" && looksLikeCommitHash(source.Ref) {
		if err := checkoutCommit(ctx, repoDir, source.Ref, source.CloneURL); err != nil {
			return err
		}
	}

	if source.Subdir != "" {
		sourceDir := filepath.Join(repoDir, filepath.FromSlash(source.Subdir))
		if _, err := os.Stat(sourceDir); err != nil {
			return fmt.Errorf("directory not found in git repository: %s", rawURL)
		}

		return installDirContentsFromSource(sourceDir, targetRoot, aliasName, false, options)
	}

	skillDirs, err := findSkillDirs(repoDir)
	if err != nil {
		return err
	}
	if len(skillDirs) == 0 {
		return fmt.Errorf("no SKILL.md found in git repository: %s", rawURL)
	}

	currentAlias := aliasName
	for _, skillDir := range skillDirs {
		if err := installDirFromSource(skillDir, targetRoot, currentAlias, options); err != nil {
			return err
		}
		currentAlias = ""
	}

	return nil
}

func looksLikeCommitHash(ref string) bool {
	matched, _ := regexp.MatchString("^[0-9a-fA-F]{40}$", ref)
	return matched
}

func checkoutCommit(ctx context.Context, repoDir string, commit string, cloneURL string) error {
	if err := runGitCommand(
		ctx,
		"-C",
		repoDir,
		"fetch",
		"--depth",
		"1",
		"origin",
		commit,
	); err != nil {
		return fmt.Errorf("fetch commit %q from git repository %q: %w", commit, cloneURL, err)
	}

	if err := runGitCommand(ctx, "-C", repoDir, "checkout", "--detach", "FETCH_HEAD"); err != nil {
		return fmt.Errorf("checkout commit %q from git repository %q: %w", commit, cloneURL, err)
	}

	return nil
}

func findSkillDirs(repoDir string) ([]string, error) {
	var skillDirs []string
	err := filepath.WalkDir(repoDir, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "SKILL.md" {
			return nil
		}

		skillDirs = append(skillDirs, filepath.Dir(path))
		return nil
	})
	if err != nil {
		return nil, err
	}

	return skillDirs, nil
}

func installDirFromSource(sourceDir string, targetRoot string, aliasName string, options installOptions) error {
	return installDirContentsFromSource(sourceDir, targetRoot, aliasName, true, options)
}

func installDirContentsFromSource(sourceDir string, targetRoot string, aliasName string, requireSkill bool, options installOptions) error {
	skillName := aliasName
	if skillName == "" {
		skillName = filepath.Base(sourceDir)
	}

	if requireSkill {
		if _, err := os.Stat(filepath.Join(sourceDir, "SKILL.md")); err != nil {
			fmt.Fprintf(os.Stderr, "Skip: missing SKILL.md in %s\n", sourceDir)
			return nil
		}
	}

	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return err
	}

	destination := filepath.Join(targetRoot, skillName)
	if options.skipExisting {
		if _, err := os.Stat(destination); err == nil {
			fmt.Printf("Skipped existing: %s\n", destination)
			return nil
		} else if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	if err := os.RemoveAll(destination); err != nil {
		return err
	}
	if err := copyDir(sourceDir, destination); err != nil {
		return err
	}

	fmt.Printf("Installed: %s -> %s\n", skillName, destination)
	return nil
}

func copyDir(source string, destination string) error {
	return filepath.WalkDir(source, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		relPath, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}

		targetPath := destination
		if relPath != "." {
			targetPath = filepath.Join(destination, relPath)
		}

		info, err := d.Info()
		if err != nil {
			return err
		}

		if d.IsDir() {
			return os.MkdirAll(targetPath, info.Mode())
		}

		return copyFile(path, targetPath, info.Mode())
	})
}

func copyFile(source string, destination string, mode fs.FileMode) error {
	in, err := os.Open(source)
	if err != nil {
		return err
	}
	defer in.Close()

	if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
		return err
	}

	out, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
	if err != nil {
		return err
	}
	defer out.Close()

	if _, err := io.Copy(out, in); err != nil {
		return err
	}

	return out.Close()
}
