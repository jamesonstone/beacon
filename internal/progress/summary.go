package progress

import (
	"fmt"
	"regexp"
	"sort"
	"strconv"
	"strings"
)

var featureIDPattern = regexp.MustCompile(`^[0-9]+$`)

type featureDetails struct {
	status    string
	paused    string
	intent    string
	approach  string
	openItems string
}

func parseSummary(contents []byte) ([]Feature, []Diagnostic) {
	lines := strings.Split(strings.ReplaceAll(string(contents), "\r\n", "\n"), "\n")
	tableStart := findHeading(lines, "## FEATURE PROGRESS TABLE")
	if tableStart < 0 {
		return nil, []Diagnostic{errorAt(SummaryPath, "missing FEATURE PROGRESS TABLE section")}
	}

	features, diagnostics := parseFeatureTable(lines, tableStart+1)
	if len(features) == 0 {
		return nil, diagnostics
	}

	details, detailDiagnostics := parseFeatureDetails(lines)
	diagnostics = append(diagnostics, detailDiagnostics...)
	for index := range features {
		detail, ok := details[features[index].Slug]
		if !ok {
			diagnostics = append(diagnostics, warningAt(SummaryPath, fmt.Sprintf("feature %q has no FEATURE SUMMARIES entry", features[index].Slug)))
			continue
		}
		features[index].Intent = detail.intent
		features[index].Approach = detail.approach
		features[index].OpenItems = detail.openItems
		if detail.status != "" && !equivalentPhaseStatus(features[index].Phase, detail.status) {
			diagnostics = append(diagnostics, warningAt(SummaryPath, fmt.Sprintf("feature %q table phase %q does not match summary status %q", features[index].Slug, features[index].Phase, detail.status)))
		}
		if detail.paused != "" {
			paused, valid := parseYesNo(detail.paused)
			if !valid {
				diagnostics = append(diagnostics, warningAt(SummaryPath, fmt.Sprintf("feature %q has invalid summary PAUSED value %q", features[index].Slug, detail.paused)))
			} else if paused != features[index].Paused {
				diagnostics = append(diagnostics, warningAt(SummaryPath, fmt.Sprintf("feature %q table and summary PAUSED values disagree", features[index].Slug)))
			}
		}
	}
	return features, diagnostics
}

func equivalentPhaseStatus(phase, status string) bool {
	phase = strings.ToLower(strings.TrimSpace(phase))
	status = strings.ToLower(strings.TrimSpace(status))
	return phase == status || strings.HasPrefix(status, phase+";")
}

func parseFeatureTable(lines []string, start int) ([]Feature, []Diagnostic) {
	var features []Feature
	var diagnostics []Diagnostic
	seen := make(map[string]struct{})
	foundHeader := false
	for index := start; index < len(lines); index++ {
		line := strings.TrimSpace(lines[index])
		if strings.HasPrefix(line, "## ") {
			break
		}
		if !strings.HasPrefix(line, "|") {
			continue
		}
		cells := splitTableRow(line)
		if len(cells) != 7 {
			diagnostics = append(diagnostics, warningAt(SummaryPath, fmt.Sprintf("line %d: expected 7 feature table columns", index+1)))
			continue
		}
		if strings.EqualFold(cells[0], "ID") {
			foundHeader = true
			continue
		}
		if isAlignmentRow(cells) {
			continue
		}
		if !foundHeader {
			continue
		}
		feature, err := featureFromCells(cells)
		if err != nil {
			diagnostics = append(diagnostics, warningAt(SummaryPath, fmt.Sprintf("line %d: %v", index+1, err)))
			continue
		}
		if _, exists := seen[feature.ID]; exists {
			diagnostics = append(diagnostics, warningAt(SummaryPath, fmt.Sprintf("line %d: duplicate feature ID %q", index+1, feature.ID)))
			continue
		}
		seen[feature.ID] = struct{}{}
		features = append(features, feature)
	}
	if !foundHeader {
		diagnostics = append(diagnostics, errorAt(SummaryPath, "feature progress table header is missing"))
	}
	return features, diagnostics
}

func featureFromCells(cells []string) (Feature, error) {
	id := strings.TrimSpace(cells[0])
	if !featureIDPattern.MatchString(id) {
		return Feature{}, fmt.Errorf("invalid feature ID %q", id)
	}
	slug := strings.TrimSpace(cells[1])
	path := unquoteCode(cells[2])
	phase := strings.TrimSpace(cells[3])
	paused, valid := parseYesNo(cells[4])
	if slug == "" || path == "" || phase == "" {
		return Feature{}, fmt.Errorf("feature, path, and phase are required")
	}
	if !valid {
		return Feature{}, fmt.Errorf("PAUSED must be yes or no, got %q", cells[4])
	}
	return Feature{
		ID: id, Slug: slug, Path: path, Phase: phase, Paused: paused,
		Created: strings.TrimSpace(cells[5]), Summary: strings.TrimSpace(cells[6]), Listed: true,
	}, nil
}

func parseFeatureDetails(lines []string) (map[string]featureDetails, []Diagnostic) {
	start := findHeading(lines, "## FEATURE SUMMARIES")
	if start < 0 {
		return nil, []Diagnostic{warningAt(SummaryPath, "missing FEATURE SUMMARIES section")}
	}
	details := make(map[string]featureDetails)
	var current string
	for index := start + 1; index < len(lines); index++ {
		line := strings.TrimSpace(lines[index])
		if strings.HasPrefix(line, "## ") && !strings.HasPrefix(line, "### ") {
			break
		}
		if strings.HasPrefix(line, "### ") {
			current = strings.TrimSpace(strings.TrimPrefix(line, "### "))
			if _, exists := details[current]; exists {
				return details, []Diagnostic{warningAt(SummaryPath, fmt.Sprintf("line %d: duplicate feature summary %q", index+1, current))}
			}
			details[current] = featureDetails{}
			continue
		}
		if current == "" || !strings.HasPrefix(line, "- **") {
			continue
		}
		key, value, ok := parseDetailLine(line)
		if !ok {
			continue
		}
		detail := details[current]
		switch key {
		case "STATUS":
			detail.status = value
		case "PAUSED":
			detail.paused = value
		case "INTENT":
			detail.intent = value
		case "APPROACH":
			detail.approach = value
		case "OPEN ITEMS":
			detail.openItems = value
		}
		details[current] = detail
	}
	return details, nil
}

func parseDetailLine(line string) (string, string, bool) {
	end := strings.Index(line, "**:")
	if end < 4 {
		return "", "", false
	}
	key := strings.TrimSpace(strings.TrimPrefix(line[:end], "- **"))
	return key, strings.TrimSpace(line[end+3:]), true
}

func splitTableRow(line string) []string {
	line = strings.TrimSpace(strings.Trim(line, "|"))
	parts := strings.Split(line, "|")
	for index := range parts {
		parts[index] = strings.TrimSpace(parts[index])
	}
	return parts
}

func isAlignmentRow(cells []string) bool {
	for _, cell := range cells {
		trimmed := strings.Trim(cell, " :-")
		if trimmed != "" {
			return false
		}
	}
	return true
}

func parseYesNo(value string) (bool, bool) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "yes":
		return true, true
	case "no":
		return false, true
	default:
		return false, false
	}
}

func findHeading(lines []string, heading string) int {
	for index, line := range lines {
		if strings.EqualFold(strings.TrimSpace(line), heading) {
			return index
		}
	}
	return -1
}

func unquoteCode(value string) string {
	return strings.Trim(strings.TrimSpace(value), "`")
}

func featureNumber(id string) int {
	number, _ := strconv.Atoi(id)
	return number
}

func sortFeatures(features []Feature) {
	sort.SliceStable(features, func(left, right int) bool {
		leftID, rightID := featureNumber(features[left].ID), featureNumber(features[right].ID)
		if leftID != rightID {
			return leftID < rightID
		}
		return features[left].Slug < features[right].Slug
	})
}
