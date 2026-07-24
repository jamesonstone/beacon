package cli

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/jamesonstone/beacon/internal/agent"
	"github.com/jamesonstone/beacon/internal/config"
)

func TestConfiguredProjectSelectorMigratesSourcesAndPersistsSelection(t *testing.T) {
	root := canonicalTestDirectory(t)
	owner := filepath.Join(root, "owner")
	first := makeRepositoryDirectory(t, owner, "first")
	second := makeRepositoryDirectory(t, owner, "second")
	outside := makeRepositoryDirectory(t, canonicalTestDirectory(t), "outside")
	configPath := filepath.Join(t.TempDir(), "config.yaml")
	writeTestConfig(t, configPath, `version: 2
sources:
  - path: `+owner+`
  - path: `+outside+`
repositories:
  - name: first
    path: `+first+`
    github: owner/first
    base: trunk
    remote: upstream
`)
	prompter := &scriptedConfiguredProjectPrompter{actions: []projectBrowserAction{
		{Kind: projectBrowserEnter, Path: owner},
		{Kind: projectBrowserUp},
		{Kind: projectBrowserEnter, Path: owner},
		{Kind: projectBrowserToggle, Path: first},
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
	for _, source := range loaded.Sources {
		selected[source.Path] = true
	}
	if len(loaded.Repositories) != 0 || len(loaded.Sources) != 2 || !selected[second] || !selected[outside] {
		t.Fatalf("config = %#v", loaded)
	}
	if starter.calls != 0 {
		t.Fatalf("agent starts = %d", starter.calls)
	}
	if !strings.Contains(output, "2 projects in ") {
		t.Fatalf("output = %q", output)
	}
	if len(prompter.views) != 5 ||
		prompter.views[0].SelectedOutside != 1 ||
		prompter.views[1].Current != owner ||
		prompter.views[2].Current != root ||
		prompter.views[4].Selected != 2 {
		t.Fatalf("views = %#v", prompter.views)
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
	if len(loaded.Sources) != 1 || loaded.Sources[0].Path != repository {
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
	if len(loaded.Sources) != 0 || len(loaded.Repositories) != 0 {
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
	command := app.Root()
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
