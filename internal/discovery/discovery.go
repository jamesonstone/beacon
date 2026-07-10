package discovery

import (
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jamesonstone/beacon/internal/command"
	"github.com/jamesonstone/beacon/internal/config"
)

const (
	gitTimeout    = 5 * time.Second
	githubTimeout = 20 * time.Second
)

type Warning struct {
	Path    string `json:"path"`
	Stage   string `json:"stage"`
	Message string `json:"message"`
}

type Result struct {
	Repositories []config.Repository
	Warnings     []Warning
}

type Discoverer struct {
	Runner command.Runner
}

type candidate struct {
	path      string
	commonDir string
	repo      config.Repository
}

func (d Discoverer) Discover(ctx context.Context, sources []config.Source) Result {
	var result Result
	var candidates []candidate
	for _, source := range sources {
		roots, warnings := repositoryRoots(source.Path)
		result.Warnings = append(result.Warnings, warnings...)
		for _, root := range roots {
			item, err := d.inspect(ctx, root)
			if err != nil {
				result.Warnings = append(result.Warnings, Warning{Path: root, Stage: "inspect", Message: err.Error()})
				continue
			}
			candidates = append(candidates, item)
		}
	}

	sort.Slice(candidates, func(i, j int) bool { return candidates[i].path < candidates[j].path })
	seenCommon := make(map[string]struct{})
	seenGitHub := make(map[string]struct{})
	for _, item := range candidates {
		if _, exists := seenCommon[item.commonDir]; exists {
			continue
		}
		if _, exists := seenGitHub[item.repo.GitHub]; exists {
			continue
		}
		seenCommon[item.commonDir] = struct{}{}
		seenGitHub[item.repo.GitHub] = struct{}{}
		result.Repositories = append(result.Repositories, item.repo)
	}
	sort.Slice(result.Warnings, func(i, j int) bool {
		if result.Warnings[i].Path != result.Warnings[j].Path {
			return result.Warnings[i].Path < result.Warnings[j].Path
		}
		return result.Warnings[i].Stage < result.Warnings[j].Stage
	})
	return result
}

func IsRepositoryRoot(path string) bool {
	root, err := os.Lstat(path)
	if err != nil || root.Mode()&os.ModeSymlink != 0 {
		return false
	}
	info, err := os.Lstat(filepath.Join(path, ".git"))
	if err != nil {
		return false
	}
	return info.IsDir() || info.Mode().IsRegular()
}

func repositoryRoots(source string) ([]string, []Warning) {
	info, err := os.Lstat(source)
	if err != nil {
		return nil, []Warning{{Path: source, Stage: "walk", Message: err.Error()}}
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return nil, []Warning{{Path: source, Stage: "walk", Message: "source is a symbolic link and was not followed"}}
	}
	if IsRepositoryRoot(source) {
		return []string{source}, nil
	}
	var roots []string
	var warnings []Warning
	err = filepath.WalkDir(source, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			warnings = append(warnings, Warning{Path: path, Stage: "walk", Message: walkErr.Error()})
			if entry != nil && entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if path != source && entry.Type()&os.ModeSymlink != 0 {
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if !entry.IsDir() {
			return nil
		}
		if IsRepositoryRoot(path) {
			roots = append(roots, path)
			return filepath.SkipDir
		}
		return nil
	})
	if err != nil {
		warnings = append(warnings, Warning{Path: source, Stage: "walk", Message: err.Error()})
	}
	sort.Strings(roots)
	return roots, warnings
}

func (d Discoverer) inspect(ctx context.Context, path string) (candidate, error) {
	commonDir, err := d.CommonDirectory(ctx, path)
	if err != nil {
		return candidate{}, err
	}

	remote, github, err := d.githubRemote(ctx, path)
	if err != nil {
		return candidate{}, err
	}
	metadata, err := d.githubMetadata(ctx, github)
	if err != nil {
		return candidate{}, err
	}
	canonicalPath, err := filepath.Abs(path)
	if err != nil {
		return candidate{}, fmt.Errorf("canonicalize repository path: %w", err)
	}
	name := strings.TrimPrefix(metadata.NameWithOwner, strings.Split(metadata.NameWithOwner, "/")[0]+"/")
	if name == "" || name == metadata.NameWithOwner {
		name = filepath.Base(canonicalPath)
	}
	base := metadata.DefaultBranchRef.Name
	if base == "" {
		base = "main"
	}
	return candidate{
		path: canonicalPath, commonDir: commonDir,
		repo: config.Repository{
			Name: name, Path: filepath.Clean(canonicalPath), GitHub: metadata.NameWithOwner,
			Base: base, Remote: remote, CommonDir: commonDir,
		},
	}, nil
}

func (d Discoverer) CommonDirectory(ctx context.Context, path string) (string, error) {
	commonOutput, err := d.run(ctx, gitTimeout, path, "git", "rev-parse", "--git-common-dir")
	if err != nil {
		return "", fmt.Errorf("resolve Git common directory: %w", err)
	}
	commonDir := strings.TrimSpace(string(commonOutput))
	if !filepath.IsAbs(commonDir) {
		commonDir = filepath.Join(path, commonDir)
	}
	commonDir, err = filepath.Abs(commonDir)
	if err != nil {
		return "", fmt.Errorf("canonicalize Git common directory: %w", err)
	}
	commonDir = filepath.Clean(commonDir)
	if resolved, resolveErr := filepath.EvalSymlinks(commonDir); resolveErr == nil {
		commonDir = resolved
	}
	return commonDir, nil
}

func (d Discoverer) githubRemote(ctx context.Context, path string) (string, string, error) {
	output, err := d.run(ctx, gitTimeout, path, "git", "remote")
	if err != nil {
		return "", "", fmt.Errorf("list Git remotes: %w", err)
	}
	remotes := strings.Fields(string(output))
	sort.Strings(remotes)
	if index := indexOf(remotes, "origin"); index > 0 {
		remotes[0], remotes[index] = remotes[index], remotes[0]
	}
	for _, remote := range remotes {
		remoteURL, runErr := d.run(ctx, gitTimeout, path, "git", "remote", "get-url", remote)
		if runErr != nil {
			continue
		}
		if github, ok := ParseGitHubRemote(strings.TrimSpace(string(remoteURL))); ok {
			return remote, github, nil
		}
	}
	return "", "", fmt.Errorf("no GitHub remote found")
}

type repositoryMetadata struct {
	NameWithOwner    string `json:"nameWithOwner"`
	DefaultBranchRef struct {
		Name string `json:"name"`
	} `json:"defaultBranchRef"`
}

func (d Discoverer) githubMetadata(ctx context.Context, github string) (repositoryMetadata, error) {
	output, err := d.run(ctx, githubTimeout, "", "gh", "repo", "view", github, "--json", "nameWithOwner,defaultBranchRef")
	if err != nil {
		return repositoryMetadata{}, fmt.Errorf("access GitHub repository %s: %w", github, err)
	}
	var metadata repositoryMetadata
	if err := json.Unmarshal(output, &metadata); err != nil {
		return repositoryMetadata{}, fmt.Errorf("decode GitHub repository %s: %w", github, err)
	}
	if metadata.NameWithOwner == "" {
		return repositoryMetadata{}, fmt.Errorf("GitHub repository %s returned no nameWithOwner", github)
	}
	return metadata, nil
}

func (d Discoverer) run(ctx context.Context, timeout time.Duration, dir, name string, args ...string) ([]byte, error) {
	commandContext, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return d.Runner.Run(commandContext, dir, name, args...)
}

func ParseGitHubRemote(remote string) (string, bool) {
	if strings.HasPrefix(remote, "git@github.com:") {
		return cleanRepository(strings.TrimPrefix(remote, "git@github.com:"))
	}
	parsed, err := url.Parse(remote)
	if err != nil || !strings.EqualFold(parsed.Hostname(), "github.com") {
		return "", false
	}
	return cleanRepository(parsed.Path)
}

func cleanRepository(path string) (string, bool) {
	path = strings.Trim(strings.TrimSuffix(path, ".git"), "/")
	parts := strings.Split(path, "/")
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", false
	}
	return parts[0] + "/" + parts[1], true
}

func indexOf(values []string, target string) int {
	for index, value := range values {
		if value == target {
			return index
		}
	}
	return -1
}
