package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
)

func main() {
	if err := run(context.Background(), os.Args[1:]); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, args []string) error {
	flagSet := flag.NewFlagSet("codex-skills-installer", flag.ContinueOnError)
	flagSet.SetOutput(os.Stdout)
	skipExisting := flagSet.Bool("skip-existing", false, "Skip installing a skill when the destination already exists")
	flagSet.Usage = func() {
		fmt.Fprintln(flagSet.Output(), "Usage:")
		fmt.Fprintln(flagSet.Output(), "  go run . [options] [url_list_file] [install_dir]")
		fmt.Fprintln(flagSet.Output())
		fmt.Fprintln(flagSet.Output(), "Options:")
		fmt.Fprintln(flagSet.Output(), "  --skip-existing  Skip installing a skill when the destination already exists")
		fmt.Fprintln(flagSet.Output())
		fmt.Fprintln(flagSet.Output(), "Arguments:")
		fmt.Fprintln(flagSet.Output(), "  url_list_file  Optional. Defaults to ./codex-skills.yml (fallback: ./codex-skils.yml)")
		fmt.Fprintln(flagSet.Output(), "  install_dir    Optional target directory. Overrides outputDir in YAML when provided")
		fmt.Fprintln(flagSet.Output())
		fmt.Fprintln(flagSet.Output(), "URL list format:")
		fmt.Fprintln(flagSet.Output(), "  - outputDir: string")
		fmt.Fprintln(flagSet.Output(), "  - targets: array of objects with url, name, and optional version")
		fmt.Fprintln(flagSet.Output(), "  - Supported sources:")
		fmt.Fprintln(flagSet.Output(), "      * Git repository URLs (e.g. https://github.com/org/repo.git)")
		fmt.Fprintln(flagSet.Output(), "      * GitHub repository URLs (e.g. https://github.com/org/repo)")
		fmt.Fprintln(flagSet.Output(), "      * GitHub tree URLs (e.g. https://github.com/org/repo/tree/main/path/to/dir)")
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

	options := installOptions{skipExisting: *skipExisting}

	for index, target := range targets {
		fmt.Printf("[%d] Processing: %s\n", index+1, target.URL)
		if err := installFromGitRepoURL(
			ctx,
			target.URL,
			installDir,
			target.Name,
			target.Version,
			options,
		); err != nil {
			return err
		}
	}

	fmt.Printf("Done. Installed skills are in: %s\n", installDir)
	return nil
}
