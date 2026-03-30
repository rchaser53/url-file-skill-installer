package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

type config struct {
	OutputDir string   `yaml:"outputDir"`
	Targets   []target `yaml:"targets"`
}

type target struct {
	URL     string `yaml:"url"`
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
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
		trimmedVersion := strings.TrimSpace(item.Version)
		if trimmedVersion == "" {
			trimmedVersion = "main"
		}
		if trimmedURL == "" {
			return nil, fmt.Errorf("targets[%d].url is required", index)
		}
		if trimmedName == "" {
			return nil, fmt.Errorf("targets[%d].name is required", index)
		}

		result = append(result, target{URL: trimmedURL, Name: trimmedName, Version: trimmedVersion})
	}

	return result, nil
}
