package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

type config struct {
	OutputDir string   `yaml:"outputDir"`
	Targets   []target `yaml:"targets"`
}

type target struct {
	URL  string `yaml:"url"`
	Name string `yaml:"name"`
}

var (
	githubRepoURLPattern    = regexp.MustCompile(`^https?://github\.com/[^/]+/[^/]+/?(\.git)?$`)
	gitlabRepoURLPattern    = regexp.MustCompile(`^https?://gitlab\.com/[^/]+/[^/]+/?(\.git)?$`)
	bitbucketRepoURLPattern = regexp.MustCompile(`^https?://bitbucket\.org/[^/]+/[^/]+/?(\.git)?$`)
	genericGitURLPattern    = regexp.MustCompile(`^https?://.+\.git/?$`)
)

func main() {
	if err := run(os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(args []string) error {
	flagSet := flag.NewFlagSet("codex-skills-installer", flag.ContinueOnError)
	flagSet.SetOutput(os.Stdout)
	flagSet.Usage = func() {
		fmt.Fprintln(flagSet.Output(), "Usage:")
		fmt.Fprintln(flagSet.Output(), "  go run . [url_list_file] [install_dir]")
		fmt.Fprintln(flagSet.Output())
		fmt.Fprintln(flagSet.Output(), "Arguments:")
		fmt.Fprintln(flagSet.Output(), "  url_list_file  Optional. Defaults to ./codex-skills.yml (fallback: ./codex-skils.yml)")
		fmt.Fprintln(flagSet.Output(), "  install_dir    Optional target directory. Overrides outputDir in YAML when provided")
		fmt.Fprintln(flagSet.Output())
		fmt.Fprintln(flagSet.Output(), "URL list format:")
		fmt.Fprintln(flagSet.Output(), "  - outputDir: string")
		fmt.Fprintln(flagSet.Output(), "  - targets: array of objects with url and name")
		fmt.Fprintln(flagSet.Output(), "  - Supported sources:")
		fmt.Fprintln(flagSet.Output(), "      * Git repository URLs (e.g. https://github.com/org/repo.git)")
		fmt.Fprintln(flagSet.Output(), "      * GitHub repository URLs (e.g. https://github.com/org/repo)")
	}

	if err := flagSet.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	remaining := flagSet.Args()
	if len(remaining) > 2 {
		flagSet.Usage()
		return errors.New("too many arguments")
	}

	urlListFile, err := resolveURLListFile(remaining)
	if err != nil {
		return err
	}

	cfg, err := parseConfig(urlListFile)
	if err != nil {
		return err
	}

	installDir, err := resolveInstallDir(remaining, cfg.OutputDir)
	if err != nil {
		return err
	}

	targets, err := normalizeTargets(cfg.Targets)
	if err != nil {
		return err
	}
	if len(targets) == 0 {
		return fmt.Errorf("no targets found in %s", urlListFile)
	}

	if err := ensureGitAvailable(); err != nil {
		return err
	}
	if err := os.MkdirAll(installDir, 0o755); err != nil {
		return err
	}

	for index, target := range targets {
		fmt.Printf("[%d] Processing: %s\n", index+1, target.URL)
		if !looksLikeGitRepoURL(target.URL) {
			return fmt.Errorf("unsupported source (expected git repository URL): %s", target.URL)
		}

		if err := installFromGitRepoURL(target.URL, installDir, target.Name); err != nil {
			return err
		}
	}

	fmt.Printf("Done. Installed skills are in: %s\n", installDir)
	return nil
}

func resolveURLListFile(args []string) (string, error) {
	if len(args) > 0 && strings.TrimSpace(args[0]) != "" {
		if _, err := os.Stat(args[0]); err != nil {
			return "", fmt.Errorf("file not found: %s", args[0])
		}
		return args[0], nil
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	primary := filepath.Join(workingDir, "codex-skills.yml")
	if _, err := os.Stat(primary); err == nil {
		return primary, nil
	}

	fallback := filepath.Join(workingDir, "codex-skils.yml")
	if _, err := os.Stat(fallback); err == nil {
		return fallback, nil
	}

	return "", fmt.Errorf("file not found: %s", primary)
}

func resolveInstallDir(args []string, configuredOutputDir string) (string, error) {
	if len(args) >= 2 && strings.TrimSpace(args[1]) != "" {
		return args[1], nil
	}

	if strings.TrimSpace(configuredOutputDir) != "" {
		return configuredOutputDir, nil
	}

	workingDir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	return filepath.Join(workingDir, ".codex", "skills"), nil
}

func parseConfig(path string) (config, error) {
	contents, err := os.ReadFile(path)
	if err != nil {
		return config{}, err
	}

	var cfg config
	if err := yaml.Unmarshal(contents, &cfg); err != nil {
		return config{}, fmt.Errorf("failed to parse YAML: %w", err)
	}

	return cfg, nil
}

func normalizeTargets(items []target) ([]target, error) {
	result := make([]target, 0, len(items))
	for index, item := range items {
		trimmedURL := strings.TrimSpace(item.URL)
		trimmedName := strings.TrimSpace(item.Name)
		if trimmedURL == "" {
			return nil, fmt.Errorf("targets[%d].url is required", index)
		}
		if trimmedName == "" {
			return nil, fmt.Errorf("targets[%d].name is required", index)
		}
		result = append(result, target{URL: trimmedURL, Name: trimmedName})
	}
	return result, nil
}

func ensureGitAvailable() error {
	if _, err := exec.LookPath("git"); err != nil {
		return errors.New("required command not found: git")
	}
	return nil
}

func looksLikeGitRepoURL(url string) bool {
	if strings.Contains(url, ".zip") || strings.Contains(url, "SKILL.md") || strings.HasSuffix(url, ".md") {
		return false
	}
	if strings.HasPrefix(url, "git@") && strings.Contains(url, ":") {
		return true
	}
	if strings.HasPrefix(url, "ssh://") {
		return true
	}
	if githubRepoURLPattern.MatchString(url) {
		return true
	}
	if gitlabRepoURLPattern.MatchString(url) {
		return true
	}
	if bitbucketRepoURLPattern.MatchString(url) {
		return true
	}
	return genericGitURLPattern.MatchString(url)
}

func installFromGitRepoURL(url string, targetRoot string, aliasName string) error {
	tmpRoot, err := os.MkdirTemp("", "codex-skill-install-*")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tmpRoot)

	repoDir := filepath.Join(tmpRoot, "repo")
	cloneCmd := exec.Command("git", "clone", "--depth", "1", url, repoDir)
	cloneCmd.Stdout = io.Discard
	cloneCmd.Stderr = io.Discard
	if err := cloneCmd.Run(); err != nil {
		return fmt.Errorf("failed to clone git repository: %s", url)
	}

	skillDirs, err := findSkillDirs(repoDir)
	if err != nil {
		return err
	}
	if len(skillDirs) == 0 {
		return fmt.Errorf("no SKILL.md found in git repository: %s", url)
	}

	currentAlias := aliasName
	for _, skillDir := range skillDirs {
		if err := installDirFromSource(skillDir, targetRoot, currentAlias); err != nil {
			return err
		}
		currentAlias = ""
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

func installDirFromSource(sourceDir string, targetRoot string, aliasName string) error {
	skillName := aliasName
	if skillName == "" {
		skillName = filepath.Base(sourceDir)
	}

	if _, err := os.Stat(filepath.Join(sourceDir, "SKILL.md")); err != nil {
		fmt.Fprintf(os.Stderr, "Skip: missing SKILL.md in %s\n", sourceDir)
		return nil
	}

	if err := os.MkdirAll(targetRoot, 0o755); err != nil {
		return err
	}

	destination := filepath.Join(targetRoot, skillName)
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
