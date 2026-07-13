package cli

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"testing"

	"github.com/jamesonstone/beacon/internal/config"
	"github.com/jamesonstone/beacon/internal/discovery"
)

func newTestInitService(path string, writer config.Writer) initService {
	return initService{
		runner:     &authRunner{},
		discoverer: fakeDiscoverer{results: map[string]discovery.Result{}},
		writer:     writer,
		prompter:   &fakeInitPrompter{confirmations: []bool{true}},
		authRunner: &recordingInheritedRunner{},
		lookup:     func(name string) (string, error) { return "/usr/bin/" + name, nil },
		isTTY:      func() bool { return false },
		out:        &bytes.Buffer{},
		errOut:     &bytes.Buffer{},
		configPath: path,
	}
}

type fakeDiscoverer struct {
	results map[string]discovery.Result
}

func (d fakeDiscoverer) Discover(_ context.Context, sources []config.Source) discovery.Result {
	var result discovery.Result
	for _, source := range sources {
		item := d.results[source.Path]
		result.Repositories = append(result.Repositories, item.Repositories...)
		result.Warnings = append(result.Warnings, item.Warnings...)
	}
	return result
}

type recordingConfigWriter struct {
	path  string
	cfg   config.Config
	calls int
}

func (w *recordingConfigWriter) Write(path string, cfg config.Config) error {
	w.path = path
	w.cfg = cfg
	w.calls++
	return nil
}

type authRunner struct {
	failures int
	calls    int
}

func (r *authRunner) Run(_ context.Context, _ string, name string, args ...string) ([]byte, error) {
	r.calls++
	if name != "gh" || fmt.Sprint(args) != "[auth status]" {
		return nil, fmt.Errorf("unexpected command: %s %v", name, args)
	}
	if r.failures > 0 {
		r.failures--
		return nil, errors.New("not authenticated")
	}
	return nil, nil
}

type recordingInheritedRunner struct{ calls int }

func (r *recordingInheritedRunner) Run(_ context.Context, name string, args ...string) error {
	r.calls++
	if name != "gh" || fmt.Sprint(args) != "[auth login]" {
		return fmt.Errorf("unexpected command: %s %v", name, args)
	}
	return nil
}

type fakeInitPrompter struct {
	mode          initMode
	directory     string
	selected      []config.Repository
	confirmations []bool
	confirmCalls  int
}

func (p *fakeInitPrompter) ChooseMode(context.Context) (initMode, error) { return p.mode, nil }
func (p *fakeInitPrompter) Directory(context.Context) (string, error)    { return p.directory, nil }
func (p *fakeInitPrompter) SelectRepositories(_ context.Context, _ []config.Repository) ([]config.Repository, error) {
	return p.selected, nil
}
func (p *fakeInitPrompter) Confirm(context.Context, string) (bool, error) {
	if p.confirmCalls >= len(p.confirmations) {
		return false, errors.New("unexpected confirmation")
	}
	value := p.confirmations[p.confirmCalls]
	p.confirmCalls++
	return value, nil
}

func writeTestConfig(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o644); err != nil {
		t.Fatal(err)
	}
}

func canonicalTestSource(t *testing.T, path string) string {
	t.Helper()
	resolved, err := config.CanonicalizeSourcePath(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}
