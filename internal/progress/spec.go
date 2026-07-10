package progress

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

	"go.yaml.in/yaml/v3"
)

var githubIssueURLPattern = regexp.MustCompile(`^https://github\.com/[^/\s?#]+/[^/\s?#]+/issues/[1-9][0-9]*$`)

type specInfo struct {
	ID        string
	Slug      string
	Dir       string
	Phase     string
	IssueURLs []string
}

type specFrontMatter struct {
	Artifact string `yaml:"artifact"`
	Phase    string `yaml:"phase"`
	Feature  struct {
		ID   yaml.Node `yaml:"id"`
		Slug string    `yaml:"slug"`
		Dir  string    `yaml:"dir"`
	} `yaml:"feature"`
	References []yaml.Node `yaml:"references"`
}

func parseSpec(contents []byte, path string) (specInfo, []Diagnostic) {
	frontMatter, err := extractFrontMatter(contents)
	if err != nil {
		return specInfo{}, []Diagnostic{warningAt(path, err.Error())}
	}
	var raw specFrontMatter
	if err := yaml.Unmarshal(frontMatter, &raw); err != nil {
		return specInfo{}, []Diagnostic{warningAt(path, fmt.Sprintf("decode YAML front matter: %v", err))}
	}
	info := specInfo{
		ID: strings.TrimSpace(raw.Feature.ID.Value), Slug: strings.TrimSpace(raw.Feature.Slug),
		Dir: strings.TrimSpace(raw.Feature.Dir), Phase: strings.TrimSpace(raw.Phase),
	}
	if !strings.EqualFold(strings.TrimSpace(raw.Artifact), "spec") {
		return specInfo{}, []Diagnostic{warningAt(path, "front matter artifact must be spec")}
	}
	if !featureIDPattern.MatchString(info.ID) || info.Slug == "" {
		return specInfo{}, []Diagnostic{warningAt(path, "front matter requires a numeric feature.id and feature.slug")}
	}

	seen := make(map[string]struct{})
	var diagnostics []Diagnostic
	for _, reference := range raw.References {
		referenceType, target, ok := referenceValues(reference)
		if !ok || !strings.EqualFold(referenceType, "github-issue") {
			continue
		}
		if !githubIssueURLPattern.MatchString(target) {
			diagnostics = append(diagnostics, warningAt(path, fmt.Sprintf("ignore malformed GitHub issue URL %q", target)))
			continue
		}
		if _, exists := seen[target]; exists {
			continue
		}
		seen[target] = struct{}{}
		info.IssueURLs = append(info.IssueURLs, target)
	}
	sort.Strings(info.IssueURLs)
	return info, diagnostics
}

func referenceValues(reference yaml.Node) (string, string, bool) {
	if reference.Kind != yaml.MappingNode {
		return "", "", false
	}
	values := make(map[string]string, len(reference.Content)/2)
	for index := 0; index+1 < len(reference.Content); index += 2 {
		values[strings.TrimSpace(reference.Content[index].Value)] = strings.TrimSpace(reference.Content[index+1].Value)
	}
	return values["type"], values["target"], true
}

func extractFrontMatter(contents []byte) ([]byte, error) {
	text := strings.ReplaceAll(string(contents), "\r\n", "\n")
	lines := strings.Split(text, "\n")
	if len(lines) == 0 || strings.TrimSpace(lines[0]) != "---" {
		return nil, fmt.Errorf("missing YAML front matter")
	}
	for index := 1; index < len(lines); index++ {
		if strings.TrimSpace(lines[index]) == "---" {
			return []byte(strings.Join(lines[1:index], "\n")), nil
		}
	}
	return nil, fmt.Errorf("unterminated YAML front matter")
}
