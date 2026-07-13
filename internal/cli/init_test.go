package cli

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/discovery"
)

func TestInitNonInteractiveMergesSourcesAndExistingConfig(t *testing.T) {
	root := t.TempDir()
	existingRepo := filepath.Join(root, "existing")
	source := filepath.Join(root, "source")
	for _, path := range []string{existingRepo, source} {
		if err := os.Mkdir(path, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	configPath := filepath.Join(root, "config.yaml")
	writeTestConfig(t, configPath, `version: 1
repositories:
  - name: existing
    path: `+existingRepo+`
    github: owner/existing
`)
	writer := &recordingConfigWriter{}
	service := newTestInitService(configPath, writer)
	service.options = initOptions{sources: []string{source}, githubScope: "all", yes: true}
	canonicalSource := canonicalTestSource(t, source)
	service.discoverer = fakeDiscoverer{results: map[string]discovery.Result{
		canonicalSource: {Repositories: []config.Repository{{Name: "found", Path: filepath.Join(source, "found"), GitHub: "owner/found"}}},
	}}

	if err := service.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if writer.calls != 1 || writer.path != configPath {
		t.Fatalf("writer = %#v", writer)
	}
	if writer.cfg.Version != config.Version || writer.cfg.Settings.GitHubScope != config.GitHubScopeAll {
		t.Fatalf("config = %#v", writer.cfg)
	}
	if len(writer.cfg.Sources) != 1 || writer.cfg.Sources[0].Path != canonicalSource {
		t.Fatalf("sources = %#v", writer.cfg.Sources)
	}
	if len(writer.cfg.Repositories) != 1 || writer.cfg.Repositories[0].Name != "existing" {
		t.Fatalf("repositories = %#v", writer.cfg.Repositories)
	}
}

func TestInitRepositorySourceBecomesExplicitRepository(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	repository := config.Repository{Name: "repo", Path: root, GitHub: "owner/repo", Base: "main", Remote: "origin"}
	writer := &recordingConfigWriter{}
	service := newTestInitService(filepath.Join(t.TempDir(), "config.yaml"), writer)
	service.options = initOptions{sources: []string{root}, yes: true}
	service.discoverer = fakeDiscoverer{results: map[string]discovery.Result{canonicalTestSource(t, root): {Repositories: []config.Repository{repository}}}}

	if err := service.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(writer.cfg.Sources) != 0 || len(writer.cfg.Repositories) != 1 || writer.cfg.Repositories[0] != repository {
		t.Fatalf("config = %#v", writer.cfg)
	}
}

func TestInitInteractiveSelectWritesOnlySelectedRepositories(t *testing.T) {
	root := t.TempDir()
	repositories := []config.Repository{
		{Name: "one", Path: filepath.Join(root, "one"), GitHub: "owner/one"},
		{Name: "two", Path: filepath.Join(root, "two"), GitHub: "owner/two"},
	}
	writer := &recordingConfigWriter{}
	prompter := &fakeInitPrompter{mode: initModeSelect, directory: root, selected: repositories[1:], confirmations: []bool{true}}
	service := newTestInitService(filepath.Join(t.TempDir(), "config.yaml"), writer)
	service.isTTY = func() bool { return true }
	service.prompter = prompter
	service.discoverer = fakeDiscoverer{results: map[string]discovery.Result{canonicalTestSource(t, root): {Repositories: repositories}}}

	if err := service.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(writer.cfg.Sources) != 0 || len(writer.cfg.Repositories) != 1 || writer.cfg.Repositories[0].GitHub != "owner/two" {
		t.Fatalf("config = %#v", writer.cfg)
	}
}

func TestInitInteractiveOffersBackgroundAgentAfterWritingConfig(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	repository := config.Repository{Name: "repo", Path: root, GitHub: "owner/repo", Base: "main", Remote: "origin"}
	writer := &recordingConfigWriter{}
	prompter := &fakeInitPrompter{confirmations: []bool{true, true}}
	service := newTestInitService(filepath.Join(t.TempDir(), "config.yaml"), writer)
	service.isTTY = func() bool { return true }
	service.prompter = prompter
	service.options = initOptions{sources: []string{root}}
	service.discoverer = fakeDiscoverer{results: map[string]discovery.Result{
		canonicalTestSource(t, root): {Repositories: []config.Repository{repository}},
	}}
	installed := false
	service.agentInstall = func(context.Context) error {
		installed = true
		return nil
	}
	if err := service.run(context.Background()); err != nil {
		t.Fatal(err)
	}
	if writer.calls != 1 || prompter.confirmCalls != 2 || !installed {
		t.Fatalf("writes=%d confirmations=%d installed=%t", writer.calls, prompter.confirmCalls, installed)
	}
}

func TestInitNonInteractiveRequirementsDoNotWrite(t *testing.T) {
	tests := []struct {
		name    string
		options initOptions
		want    string
	}{
		{name: "source required", options: initOptions{yes: true}, want: "requires --source"},
		{name: "confirmation required", options: initOptions{sources: []string{t.TempDir()}}, want: "requires --yes"},
		{name: "scope", options: initOptions{sources: []string{t.TempDir()}, githubScope: "team", yes: true}, want: "mine or all"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			writer := &recordingConfigWriter{}
			service := newTestInitService(filepath.Join(t.TempDir(), "config.yaml"), writer)
			service.options = test.options
			err := service.run(context.Background())
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v", err)
			}
			if writer.calls != 0 {
				t.Fatalf("writer calls = %d", writer.calls)
			}
		})
	}
}

func TestInitPrerequisiteAndAuthenticationFailures(t *testing.T) {
	t.Run("missing gh", func(t *testing.T) {
		service := newTestInitService("unused", &recordingConfigWriter{})
		service.options = initOptions{sources: []string{t.TempDir()}, yes: true}
		service.lookup = func(name string) (string, error) {
			if name == "gh" {
				return "", os.ErrNotExist
			}
			return "/usr/bin/" + name, nil
		}
		err := service.run(context.Background())
		if err == nil || !strings.Contains(err.Error(), "gh is required") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("non interactive auth", func(t *testing.T) {
		service := newTestInitService("unused", &recordingConfigWriter{})
		service.options = initOptions{sources: []string{t.TempDir()}, yes: true}
		service.runner = &authRunner{failures: 1}
		err := service.run(context.Background())
		if err == nil || err.Error() != "GitHub authentication is required; run: gh auth login" {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestInitInteractiveAuthenticationAndCancellation(t *testing.T) {
	root := t.TempDir()
	writer := &recordingConfigWriter{}
	auth := &authRunner{failures: 1}
	login := &recordingInheritedRunner{}
	prompter := &fakeInitPrompter{confirmations: []bool{true, false}}
	service := newTestInitService(filepath.Join(t.TempDir(), "config.yaml"), writer)
	service.runner = auth
	service.authRunner = login
	service.prompter = prompter
	service.isTTY = func() bool { return true }
	service.options = initOptions{sources: []string{root}}
	service.discoverer = fakeDiscoverer{results: map[string]discovery.Result{canonicalTestSource(t, root): {}}}

	err := service.run(context.Background())
	if err == nil || !strings.Contains(err.Error(), "cancelled") {
		t.Fatalf("error = %v", err)
	}
	if login.calls != 1 || auth.calls != 2 || writer.calls != 0 {
		t.Fatalf("login=%d auth=%d writes=%d", login.calls, auth.calls, writer.calls)
	}
}

func TestInitCommandUsesRepeatableSourceFlagAndConfigAlias(t *testing.T) {
	path := ""
	app := App{Out: &bytes.Buffer{}, Err: &bytes.Buffer{}, Runner: &authRunner{}}
	command := app.initCommand(&path)
	if err := command.Flags().Set("source", "/one"); err != nil {
		t.Fatal(err)
	}
	if err := command.Flags().Set("source", "/two"); err != nil {
		t.Fatal(err)
	}
	values, err := command.Flags().GetStringArray("source")
	if err != nil {
		t.Fatal(err)
	}
	if fmt.Sprint(values) != "[/one /two]" {
		t.Fatalf("sources = %#v", values)
	}
	if app.configCommand(&path).CommandPath() != "config" {
		t.Fatal("config command was not constructed")
	}
	if initAlias, _, err := app.configCommand(&path).Find([]string{"init"}); err != nil || initAlias == nil {
		t.Fatalf("config init alias: command=%v err=%v", initAlias, err)
	}
}
