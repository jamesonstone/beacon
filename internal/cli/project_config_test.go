package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/charmbracelet/bubbles/list"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
)

func TestConfiguredProjectSelectorUsesDedicatedProjectSelection(t *testing.T) {
	root := canonicalTestDirectory(t)
	owner := filepath.Join(root, "owner")
	first := makeRepositoryDirectory(t, owner, "first")
	second := makeRepositoryDirectory(t, owner, "second")
	outside := makeRepositoryDirectory(t, canonicalTestDirectory(t), "outside")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath, `version: 2
projects:
  - path: `+outside+`
sources:
  - path: `+owner+`
repositories:
  - name: first
    path: `+first+`
    github: owner/first
    base: trunk
    remote: upstream
`)
	prompter := &scriptedConfiguredProjectPrompter{actions: []projectBrowserAction{
		{Kind: projectBrowserEnter, Path: owner},
		{Kind: projectBrowserToggle, Path: second},
		{Kind: projectBrowserSave},
	}}
	output, starter, err := executeConfiguredProjects(t, configPath, root, prompter)
	if err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	selected := map[string]bool{}
	for _, project := range loaded.Projects {
		selected[project.Path] = true
	}
	if len(loaded.Projects) != 2 || !selected[second] || !selected[outside] ||
		len(loaded.Sources) != 1 || loaded.Sources[0].Path != owner ||
		len(loaded.Repositories) != 1 || loaded.Repositories[0].Base != "trunk" {
		t.Fatalf("config = %#v", loaded)
	}
	if starter.calls != 0 {
		t.Fatalf("agent starts = %d", starter.calls)
	}
	if !strings.Contains(output, "2 projects in ") {
		t.Fatalf("output = %q", output)
	}
	if len(prompter.views) != 3 ||
		prompter.views[0].SelectedOutside != 1 ||
		prompter.views[1].Current != owner ||
		prompter.views[2].Selected != 2 ||
		prompter.views[2].FocusPath != second {
		t.Fatalf("views = %#v", prompter.views)
	}
}

func TestConfiguredProjectSelectorStartsLegacyInventoryUnselected(t *testing.T) {
	root := canonicalTestDirectory(t)
	repository := makeRepositoryDirectory(t, root, "repo")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath, `version: 2
sources:
  - path: `+root+`
repositories:
  - name: repo
    path: `+repository+`
    github: owner/repo
`)
	prompter := &scriptedConfiguredProjectPrompter{
		actions: []projectBrowserAction{{Kind: projectBrowserSave}},
	}
	if _, _, err := executeConfiguredProjects(t, configPath, root, prompter); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(prompter.views) != 1 || prompter.views[0].Selected != 0 ||
		len(loaded.Projects) != 0 || len(loaded.Sources) != 1 ||
		len(loaded.Repositories) != 1 {
		t.Fatalf("views = %#v, config = %#v", prompter.views, loaded)
	}
}

func TestConfiguredProjectSelectorCreatesConfigAndCanSaveEmptySelection(t *testing.T) {
	root := canonicalTestDirectory(t)
	owner := filepath.Join(root, "owner")
	repository := makeRepositoryDirectory(t, owner, "repo")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	prompter := &scriptedConfiguredProjectPrompter{actions: []projectBrowserAction{
		{Kind: projectBrowserEnter, Path: owner},
		{Kind: projectBrowserToggle, Path: repository},
		{Kind: projectBrowserSave},
	}}
	if _, _, err := executeConfiguredProjects(t, configPath, root, prompter); err != nil {
		t.Fatal(err)
	}
	loaded, err := config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Projects) != 1 || loaded.Projects[0].Path != repository {
		t.Fatalf("config = %#v", loaded)
	}

	prompter = &scriptedConfiguredProjectPrompter{actions: []projectBrowserAction{
		{Kind: projectBrowserEnter, Path: owner},
		{Kind: projectBrowserToggle, Path: repository},
		{Kind: projectBrowserSave},
	}}
	if _, _, err := executeConfiguredProjects(t, configPath, root, prompter); err != nil {
		t.Fatal(err)
	}
	loaded, err = config.Load(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if len(loaded.Projects) != 0 {
		t.Fatalf("empty config = %#v", loaded)
	}
}

func TestConfiguredProjectSelectorCancelDoesNotWrite(t *testing.T) {
	root := canonicalTestDirectory(t)
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	original := []byte("version: 2\nsources: []\nrepositories: []\n")
	if err := os.WriteFile(configPath, original, 0o644); err != nil {
		t.Fatal(err)
	}
	prompter := &scriptedConfiguredProjectPrompter{
		actions: []projectBrowserAction{{Kind: projectBrowserCancel}},
	}
	output, _, err := executeConfiguredProjects(t, configPath, root, prompter)
	if err != nil {
		t.Fatal(err)
	}
	contents, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(contents, original) || !strings.Contains(output, "unchanged") {
		t.Fatalf("contents = %q, output = %q", contents, output)
	}
}

func TestConfiguredProjectSelectorDefaultsToGoGithubRoot(t *testing.T) {
	home := canonicalTestDirectory(t)
	t.Setenv("HOME", home)
	root := filepath.Join(home, "go", "src", "github.com")
	repository := makeRepositoryDirectory(t, root, "repo")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	prompter := &scriptedConfiguredProjectPrompter{actions: []projectBrowserAction{
		{Kind: projectBrowserToggle, Path: repository},
		{Kind: projectBrowserSave},
	}}
	if _, _, err := executeConfiguredProjects(t, configPath, "", prompter); err != nil {
		t.Fatal(err)
	}
	if len(prompter.views) != 2 || prompter.views[0].Root != root {
		t.Fatalf("views = %#v", prompter.views)
	}
}

func TestProjectBrowserEntriesSkipFilesHiddenDirectoriesAndSymlinks(t *testing.T) {
	root := canonicalTestDirectory(t)
	repository := makeRepositoryDirectory(t, root, "repo")
	if err := os.Mkdir(filepath.Join(root, ".hidden"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "file"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(repository, filepath.Join(root, "linked")); err != nil {
		t.Fatal(err)
	}
	entries, err := projectBrowserEntries(root, map[string]struct{}{repository: {}})
	if err != nil {
		t.Fatal(err)
	}
	want := []projectBrowserEntry{{
		Name: "repo", Path: repository, Repository: true, Selected: true,
	}}
	if !reflect.DeepEqual(entries, want) {
		t.Fatalf("entries = %#v", entries)
	}
}

func TestProjectBrowserModelSpaceActsOnHighlightedEntry(t *testing.T) {
	root := canonicalTestDirectory(t)
	directory := filepath.Join(root, "owner")
	repository := makeRepositoryDirectory(t, root, "repo")
	view := projectBrowserView{
		Root: root, Current: root,
		Entries: []projectBrowserEntry{
			{Name: "owner", Path: directory},
			{Name: "repo", Path: repository, Repository: true},
		},
	}

	model := newProjectBrowserModel(view)
	result, command := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	updated := result.(projectBrowserModel)
	if command == nil || updated.action != (projectBrowserAction{Kind: projectBrowserEnter, Path: directory}) {
		t.Fatalf("directory action = %#v, command = %v", updated.action, command)
	}

	model = newProjectBrowserModel(view)
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(projectBrowserModel)
	result, command = model.Update(tea.KeyMsg{Type: tea.KeySpace})
	updated = result.(projectBrowserModel)
	if command == nil || updated.action != (projectBrowserAction{Kind: projectBrowserToggle, Path: repository}) {
		t.Fatalf("repository action = %#v, command = %v", updated.action, command)
	}
}

func TestProjectBrowserModelDelegatesSpaceAndEnterWhileFiltering(t *testing.T) {
	root := canonicalTestDirectory(t)
	view := projectBrowserView{
		Root: root, Current: root,
		Entries: []projectBrowserEntry{{
			Name: "repository", Path: filepath.Join(root, "repository"), Repository: true,
		}},
	}

	model := newProjectBrowserModel(view)
	model.list.SetFilterState(list.Filtering)
	result, _ := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	updated := result.(projectBrowserModel)
	if updated.action.Kind != "" || updated.list.FilterInput.Value() != " " {
		t.Fatalf("space action = %#v, filter = %q", updated.action, updated.list.FilterInput.Value())
	}

	model = newProjectBrowserModel(view)
	model.list.SetFilterText("repo")
	model.list.SetFilterState(list.Filtering)
	result, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	updated = result.(projectBrowserModel)
	if updated.action.Kind != "" || updated.list.FilterState() != list.FilterApplied {
		t.Fatalf("enter action = %#v, filter state = %s", updated.action, updated.list.FilterState())
	}
}

func TestProjectBrowserModelEnterConfirmsAndTabMovesForward(t *testing.T) {
	root := canonicalTestDirectory(t)
	view := projectBrowserView{
		Root: root, Current: root,
		Entries: []projectBrowserEntry{
			{Name: "first", Path: filepath.Join(root, "first")},
			{Name: "second", Path: filepath.Join(root, "second")},
		},
	}
	model := newProjectBrowserModel(view)

	result, command := model.Update(tea.KeyMsg{Type: tea.KeyTab})
	model = result.(projectBrowserModel)
	if command != nil || model.list.Index() != 1 || model.action.Kind != "" {
		t.Fatalf("tab index = %d, action = %#v, command = %v", model.list.Index(), model.action, command)
	}
	result, command = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model = result.(projectBrowserModel)
	if command == nil || model.action.Kind != projectBrowserSave {
		t.Fatalf("enter action = %#v, command = %v", model.action, command)
	}
}

func TestProjectBrowserModelRestoresFocusAndSpaceMovesUp(t *testing.T) {
	root := canonicalTestDirectory(t)
	current := filepath.Join(root, "owner")
	repository := filepath.Join(root, "repo")
	view := projectBrowserView{
		Root: root, Current: root, FocusPath: repository,
		Entries: []projectBrowserEntry{
			{Name: "owner", Path: current},
			{Name: "repo", Path: repository, Repository: true},
		},
	}
	model := newProjectBrowserModel(view)
	if model.list.Index() != 1 {
		t.Fatalf("focus index = %d", model.list.Index())
	}

	model = newProjectBrowserModel(projectBrowserView{Root: root, Current: current})
	result, command := model.Update(tea.KeyMsg{Type: tea.KeySpace})
	model = result.(projectBrowserModel)
	if command == nil || model.action.Kind != projectBrowserUp {
		t.Fatalf("up action = %#v, command = %v", model.action, command)
	}
}

func executeConfiguredProjects(
	t *testing.T,
	configPath string,
	root string,
	prompter configuredProjectPrompter,
) (string, *recordingAgentStarter, error) {
	t.Helper()
	var output bytes.Buffer
	starter := &recordingAgentStarter{}
	app := App{
		Out: &output, Err: &bytes.Buffer{}, Runner: &recordingRunner{},
		InputIsTTY: func() bool { return true }, OutputIsTTY: func() bool { return false },
		TerminalWidth: func() int { return 120 }, autoStartAgent: true,
		configuredProjectPrompterSource: prompter,
		agentStarterSource:              func(agent.Paths) agentStarter { return starter },
	}
	command := app.BctlRoot()
	args := []string{"--config", configPath, "--color", "never", "projects"}
	if root != "" {
		args = append(args, "--root", root)
	}
	command.SetArgs(args)
	err := command.ExecuteContext(context.Background())
	return output.String(), starter, err
}

func makeRepositoryDirectory(t *testing.T, parent, name string) string {
	t.Helper()
	path := filepath.Join(parent, name)
	if err := os.MkdirAll(filepath.Join(path, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

func canonicalTestDirectory(t *testing.T) string {
	t.Helper()
	path, err := filepath.EvalSymlinks(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	return path
}

type scriptedConfiguredProjectPrompter struct {
	actions []projectBrowserAction
	views   []projectBrowserView
}

func (p *scriptedConfiguredProjectPrompter) ChooseConfiguredProject(
	_ context.Context,
	view projectBrowserView,
) (projectBrowserAction, error) {
	p.views = append(p.views, view)
	action := p.actions[0]
	p.actions = p.actions[1:]
	return action, nil
}
