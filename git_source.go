package main

import (
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

type gitSource struct {
	CloneURL string
	Ref      string
	Subdir   string
}

func parseGitSource(ctx context.Context, rawURL string) (gitSource, error) {
	if strings.HasPrefix(rawURL, "git@") || strings.HasPrefix(rawURL, "ssh://") {
		return gitSource{CloneURL: rawURL}, nil
	}
	if !isSupportedRepoArtifact(rawURL) {
		return gitSource{}, fmt.Errorf("unsupported source (expected git repository URL): %s", rawURL)
	}

	parsed, err := url.Parse(rawURL)
	if err != nil {
		return gitSource{}, fmt.Errorf("invalid URL: %s", rawURL)
	}

	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return gitSource{}, fmt.Errorf("unsupported source (expected git repository URL): %s", rawURL)
	}

	pathParts := splitURLPath(parsed.Path)
	if len(pathParts) < 2 {
		return gitSource{}, fmt.Errorf("unsupported source (expected git repository URL): %s", rawURL)
	}

	if parsed.Host == "github.com" {
		return parseGitHubSource(ctx, parsed, pathParts)
	}

	return gitSource{CloneURL: rawURL}, nil
}

func parseGitHubSource(ctx context.Context, parsed *url.URL, pathParts []string) (gitSource, error) {
	if len(pathParts) < 2 {
		return gitSource{}, fmt.Errorf("unsupported source (expected git repository URL): %s", parsed.String())
	}

	repoPath := strings.Join(pathParts[:2], "/")
	cloneURL := parsed.Scheme + "://" + parsed.Host + "/" + repoPath
	if len(pathParts) == 2 {
		return gitSource{CloneURL: cloneURL}, nil
	}
	if len(pathParts) >= 4 && pathParts[2] == "tree" {
		refAndSubdir := pathParts[3:]
		ref, subdir, err := resolveGitHubTreeRef(ctx, cloneURL, refAndSubdir)
		if err != nil {
			return gitSource{}, err
		}
		if subdir == "" {
			return gitSource{}, fmt.Errorf("directory path is required for tree URL: %s", parsed.String())
		}

		return gitSource{CloneURL: cloneURL, Ref: ref, Subdir: subdir}, nil
	}

	if strings.HasSuffix(repoPath, ".git") {
		return gitSource{CloneURL: cloneURL}, nil
	}

	return gitSource{}, fmt.Errorf("unsupported source (expected git repository URL): %s", parsed.String())
}

func resolveGitHubTreeRef(ctx context.Context, cloneURL string, refAndSubdir []string) (string, string, error) {
	if len(refAndSubdir) < 2 {
		return "", "", fmt.Errorf("directory path is required for tree URL: %s/tree/%s", cloneURL, strings.Join(refAndSubdir, "/"))
	}

	refs, err := listRemoteRefs(ctx, cloneURL)
	if err != nil {
		return "", "", err
	}

	for candidateLength := len(refAndSubdir) - 1; candidateLength >= 1; candidateLength-- {
		candidateRef := strings.Join(refAndSubdir[:candidateLength], "/")
		if _, ok := refs[candidateRef]; !ok {
			continue
		}

		return candidateRef, strings.Join(refAndSubdir[candidateLength:], "/"), nil
	}

	return "", "", fmt.Errorf("failed to resolve branch or tag from tree URL: %s/tree/%s", cloneURL, strings.Join(refAndSubdir, "/"))
}

func listRemoteRefs(ctx context.Context, cloneURL string) (map[string]struct{}, error) {
	output, err := gitCommandOutput(ctx, "ls-remote", "--heads", "--tags", cloneURL)
	if err != nil {
		return nil, fmt.Errorf("inspect git repository %q: %w", cloneURL, err)
	}

	refs := map[string]struct{}{}
	for _, line := range strings.Split(string(output), "\n") {
		fields := strings.Fields(line)
		if len(fields) != 2 {
			continue
		}

		refName := strings.TrimPrefix(fields[1], "refs/heads/")
		refName = strings.TrimPrefix(refName, "refs/tags/")
		if refName != fields[1] {
			refs[refName] = struct{}{}
		}
	}

	return refs, nil
}

func splitURLPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}

	return strings.Split(trimmed, "/")
}

func isSupportedRepoArtifact(rawURL string) bool {
	if strings.Contains(rawURL, ".zip") {
		return false
	}
	if strings.Contains(rawURL, "SKILL.md") {
		return false
	}

	return !strings.HasSuffix(rawURL, ".md")
}

func runGitCommand(ctx context.Context, args ...string) error {
	_, err := gitCommandOutput(ctx, args...)
	return err
}

func gitCommandOutput(ctx context.Context, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return nil, formatGitCommandError(args, output, err)
	}

	return output, nil
}

func formatGitCommandError(args []string, output []byte, err error) error {
	trimmedOutput := strings.TrimSpace(string(output))
	if trimmedOutput == "" {
		return fmt.Errorf("git %s: %w", strings.Join(args, " "), err)
	}

	return fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, trimmedOutput)
}
