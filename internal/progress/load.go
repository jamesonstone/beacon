package progress

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// Load reads optional Kit progress evidence from repoRoot. A repository without
// a project progress summary returns an empty result without diagnostics.
func Load(repoRoot string) Result {
	root, err := filepath.Abs(repoRoot)
	if err != nil {
		return Result{Diagnostics: []Diagnostic{errorAt(repoRoot, fmt.Sprintf("resolve repository root: %v", err))}}
	}
	summaryFile := filepath.Join(root, filepath.FromSlash(SummaryPath))
	contents, err := os.ReadFile(summaryFile)
	if errors.Is(err, os.ErrNotExist) {
		return Result{}
	}
	if err != nil {
		return Result{Diagnostics: []Diagnostic{errorAt(SummaryPath, fmt.Sprintf("read progress summary: %v", err))}}
	}

	features, diagnostics := parseSummary(contents)
	if len(features) == 0 {
		return Result{Diagnostics: diagnostics}
	}
	features, specDiagnostics := mergeSpecs(root, features)
	diagnostics = append(diagnostics, specDiagnostics...)
	sortFeatures(features)

	result := Result{Features: features, Diagnostics: diagnostics}
	result.Selected = selectFeature(features)
	result.IssueLinks = buildIssueLinks(features)
	return result
}

func mergeSpecs(root string, features []Feature) ([]Feature, []Diagnostic) {
	byPath := make(map[string]int, len(features))
	var diagnostics []Diagnostic
	for index := range features {
		relative, valid := safeRelativePath(features[index].Path)
		if !valid {
			diagnostics = append(diagnostics, warningAt(SummaryPath, fmt.Sprintf("feature %q has unsafe path %q", features[index].Slug, features[index].Path)))
			continue
		}
		features[index].Path = relative
		if prior, exists := byPath[relative]; exists {
			diagnostics = append(diagnostics, warningAt(SummaryPath, fmt.Sprintf("features %q and %q use the same path %q", features[prior].Slug, features[index].Slug, relative)))
			continue
		}
		byPath[relative] = index
	}

	pattern := filepath.Join(root, "docs", "specs", "*", "SPEC.md")
	paths, err := filepath.Glob(pattern)
	if err != nil {
		return features, append(diagnostics, errorAt("docs/specs", fmt.Sprintf("discover feature specs: %v", err)))
	}
	sort.Strings(paths)
	seenSpecs := make(map[string]struct{}, len(paths))
	for _, absolute := range paths {
		relativeFile, err := filepath.Rel(root, absolute)
		if err != nil {
			continue
		}
		relativeFile = filepath.ToSlash(relativeFile)
		relativeDir := filepath.ToSlash(filepath.Dir(relativeFile))
		contents, err := os.ReadFile(absolute)
		if err != nil {
			diagnostics = append(diagnostics, warningAt(relativeFile, fmt.Sprintf("read feature spec: %v", err)))
			continue
		}
		info, specDiagnostics := parseSpec(contents, relativeFile)
		diagnostics = append(diagnostics, specDiagnostics...)
		if info.ID == "" {
			continue
		}
		if info.Dir != "" && info.Dir != filepath.Base(relativeDir) {
			diagnostics = append(diagnostics, warningAt(relativeFile, fmt.Sprintf("SPEC feature.dir %q does not match directory %q", info.Dir, filepath.Base(relativeDir))))
		}
		seenSpecs[relativeDir] = struct{}{}
		if index, exists := byPath[relativeDir]; exists {
			features[index], specDiagnostics = mergeSpec(features[index], info, relativeFile)
			diagnostics = append(diagnostics, specDiagnostics...)
			continue
		}
		features = append(features, Feature{
			ID: info.ID, Slug: info.Slug, Path: relativeDir, Phase: info.Phase,
			SpecPath: relativeFile, IssueURLs: info.IssueURLs,
		})
		diagnostics = append(diagnostics, warningAt(relativeFile, "feature SPEC is not listed in project progress summary"))
	}

	for _, feature := range features {
		if !feature.Listed {
			continue
		}
		if _, exists := seenSpecs[feature.Path]; !exists {
			diagnostics = append(diagnostics, warningAt(SummaryPath, fmt.Sprintf("feature %q references missing or malformed %s/SPEC.md", feature.Slug, feature.Path)))
		}
	}
	return deduplicateFeatures(features, &diagnostics), diagnostics
}

func mergeSpec(feature Feature, info specInfo, specPath string) (Feature, []Diagnostic) {
	var diagnostics []Diagnostic
	if feature.ID != info.ID {
		diagnostics = append(diagnostics, warningAt(specPath, fmt.Sprintf("SPEC feature ID %q does not match summary ID %q; using SPEC", info.ID, feature.ID)))
		feature.ID = info.ID
	}
	if feature.Slug != info.Slug {
		diagnostics = append(diagnostics, warningAt(specPath, fmt.Sprintf("SPEC feature slug %q does not match summary slug %q; using SPEC", info.Slug, feature.Slug)))
		feature.Slug = info.Slug
	}
	if feature.Phase != info.Phase {
		diagnostics = append(diagnostics, warningAt(specPath, fmt.Sprintf("SPEC phase %q does not match summary phase %q; using SPEC", info.Phase, feature.Phase)))
		feature.Phase = info.Phase
	}
	feature.SpecPath = specPath
	feature.IssueURLs = info.IssueURLs
	return feature, diagnostics
}

func deduplicateFeatures(features []Feature, diagnostics *[]Diagnostic) []Feature {
	sortFeatures(features)
	seen := make(map[string]struct{}, len(features))
	result := make([]Feature, 0, len(features))
	for _, feature := range features {
		if _, exists := seen[feature.ID]; exists {
			*diagnostics = append(*diagnostics, warningAt(feature.SpecPath, fmt.Sprintf("duplicate feature ID %s (%s) ignored", feature.ID, feature.Slug)))
			continue
		}
		seen[feature.ID] = struct{}{}
		result = append(result, feature)
	}
	return result
}

func safeRelativePath(path string) (string, bool) {
	cleaned := filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(path))))
	if cleaned == "." || filepath.IsAbs(cleaned) || cleaned == ".." || strings.HasPrefix(cleaned, "../") {
		return "", false
	}
	return cleaned, true
}

func selectFeature(features []Feature) *Feature {
	for index := len(features) - 1; index >= 0; index-- {
		if !features[index].Paused && !isDelivered(features[index].Phase) {
			selected := features[index]
			return &selected
		}
	}
	for index := len(features) - 1; index >= 0; index-- {
		if isDelivered(features[index].Phase) {
			selected := features[index]
			return &selected
		}
	}
	return nil
}

func buildIssueLinks(features []Feature) []IssueLink {
	featureIDs := make(map[string][]string)
	for _, feature := range features {
		for _, issueURL := range feature.IssueURLs {
			featureIDs[issueURL] = append(featureIDs[issueURL], feature.ID)
		}
	}
	urls := make([]string, 0, len(featureIDs))
	for issueURL := range featureIDs {
		urls = append(urls, issueURL)
	}
	sort.Strings(urls)
	links := make([]IssueLink, 0, len(urls))
	for _, issueURL := range urls {
		ids := featureIDs[issueURL]
		sort.SliceStable(ids, func(left, right int) bool { return featureNumber(ids[left]) < featureNumber(ids[right]) })
		links = append(links, IssueLink{URL: issueURL, FeatureIDs: ids})
	}
	return links
}

func warningAt(path, message string) Diagnostic {
	return Diagnostic{Severity: SeverityWarning, Path: path, Message: message}
}

func errorAt(path, message string) Diagnostic {
	return Diagnostic{Severity: SeverityError, Path: path, Message: message}
}
