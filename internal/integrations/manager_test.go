package integrations

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestInstallStatusObserveAndUninstallPreserveUnrelatedHooks(t *testing.T) {
	home := t.TempDir()
	executable := filepath.Join(home, "bin", "beacon-cli")
	if err := os.MkdirAll(filepath.Dir(executable), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(executable, []byte("#!/bin/sh\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	healthPath := filepath.Join(home, ".cache", "beacon", "integration-health.json")
	now := time.Date(2026, 7, 16, 12, 0, 0, 123, time.UTC)
	manager := Manager{Home: home, Executable: executable, Health: HealthStore{Path: healthPath}, Now: func() time.Time { return now }}
	settings, _ := manager.SettingsPath(ProviderClaudeCode)
	if err := os.MkdirAll(filepath.Dir(settings), 0o700); err != nil {
		t.Fatal(err)
	}
	unrelated := `{"theme":"dark","hooks":{"Stop":[{"matcher":"custom","hooks":[{"type":"command","command":"echo unrelated","timeout":9}]}]}}`
	if err := os.WriteFile(settings, []byte(unrelated), 0o644); err != nil {
		t.Fatal(err)
	}
	plan, err := manager.PlanInstall(ProviderClaudeCode)
	if err != nil {
		t.Fatal(err)
	}
	if !plan.Changed || len(plan.Changes) != 6 || !strings.HasSuffix(plan.BackupPath, ".beacon-backup-20260716T120000.000000123Z") {
		t.Fatalf("install plan = %#v", plan)
	}
	for _, change := range plan.Changes {
		if !strings.Contains(change.Command, ">/dev/null 2>&1 || true # "+Marker) {
			t.Fatalf("command lacks fail-open guard: %s", change.Command)
		}
	}
	if err := manager.Apply(plan); err != nil {
		t.Fatal(err)
	}
	for _, path := range []string{settings, plan.BackupPath, healthPath} {
		info, err := os.Stat(path)
		if err != nil || info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode = %v, %v", path, info.Mode().Perm(), err)
		}
	}
	contents, _ := os.ReadFile(settings)
	if !bytes.Contains(contents, []byte("echo unrelated")) || !bytes.Contains(contents, []byte("agent_needs_input")) {
		t.Fatalf("installed settings = %s", contents)
	}
	if status := manager.Status(ProviderClaudeCode); status.State != StateInstalled {
		t.Fatalf("installed status = %#v", status)
	}
	if err := manager.Observe(ProviderClaudeCode); err != nil {
		t.Fatal(err)
	}
	if status := manager.Status(ProviderClaudeCode); status.State != StateActive {
		t.Fatalf("active status = %#v", status)
	}
	document, _, err := readSettings(settings)
	if err != nil {
		t.Fatal(err)
	}
	hooks := document["hooks"].(map[string]any)
	hooks["SessionEnd"] = append(hooks["SessionEnd"].([]any), map[string]any{
		"hooks": []any{map[string]any{"type": "command", "command": "echo added-later", "timeout": json.Number("2")}},
	})
	updated, _ := marshalSettings(document)
	if err := os.WriteFile(settings, updated, 0o600); err != nil {
		t.Fatal(err)
	}
	if status := manager.Status(ProviderClaudeCode); status.State != StateActive {
		t.Fatalf("status after unrelated hook = %#v", status)
	}
	idempotent, err := manager.PlanInstall(ProviderClaudeCode)
	if err != nil || idempotent.Changed || len(idempotent.Changes) != 0 {
		t.Fatalf("idempotent plan = %#v, %v", idempotent, err)
	}
	uninstall, err := manager.PlanUninstall(ProviderClaudeCode)
	if err != nil || !uninstall.Changed || len(uninstall.Changes) != 6 {
		t.Fatalf("uninstall plan = %#v, %v", uninstall, err)
	}
	if uninstall.BackupPath == plan.BackupPath {
		t.Fatalf("backup path reused: %s", uninstall.BackupPath)
	}
	if err := manager.Apply(uninstall); err != nil {
		t.Fatal(err)
	}
	contents, _ = os.ReadFile(settings)
	if !bytes.Contains(contents, []byte("echo unrelated")) || !bytes.Contains(contents, []byte("echo added-later")) || bytes.Contains(contents, []byte(Marker)) {
		t.Fatalf("uninstalled settings = %s", contents)
	}
	if status := manager.Status(ProviderClaudeCode); status.State != StateNotInstalled {
		t.Fatalf("uninstalled status = %#v", status)
	}
}

func TestApplyRefusesSettingsChangedAfterPreview(t *testing.T) {
	home := t.TempDir()
	manager := Manager{
		Home: home, Executable: executableFile(t, home),
		Health: HealthStore{Path: filepath.Join(home, "health.json")},
	}
	settings, _ := manager.SettingsPath(ProviderCodex)
	if err := os.MkdirAll(filepath.Dir(settings), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(settings, []byte(`{"theme":"dark"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	plan, err := manager.PlanInstall(ProviderCodex)
	if err != nil {
		t.Fatal(err)
	}
	changed := []byte(`{"theme":"light"}`)
	if err := os.WriteFile(settings, changed, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := manager.Apply(plan); err == nil || !strings.Contains(err.Error(), "changed after preview") {
		t.Fatalf("apply error = %v", err)
	}
	contents, _ := os.ReadFile(settings)
	if !bytes.Equal(contents, changed) {
		t.Fatalf("concurrent settings overwritten: %s", contents)
	}
	if _, err := os.Stat(plan.BackupPath); !os.IsNotExist(err) {
		t.Fatalf("backup created after stale preview: %v", err)
	}
}

func TestSettingsSymlinkFailsClosed(t *testing.T) {
	home := t.TempDir()
	manager := Manager{
		Home: home, Executable: executableFile(t, home),
		Health: HealthStore{Path: filepath.Join(home, "health.json")},
	}
	settings, _ := manager.SettingsPath(ProviderCodex)
	if err := os.MkdirAll(filepath.Dir(settings), 0o700); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(home, "target.json")
	if err := os.WriteFile(target, []byte(`{}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, settings); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.PlanInstall(ProviderCodex); err == nil {
		t.Fatal("settings symlink accepted")
	}
	contents, _ := os.ReadFile(target)
	if string(contents) != "{}" {
		t.Fatalf("symlink target mutated: %s", contents)
	}
}

func TestMalformedSettingsFailClosedWithoutBackupOrMutation(t *testing.T) {
	home := t.TempDir()
	executable := executableFile(t, home)
	manager := Manager{Home: home, Executable: executable, Health: HealthStore{Path: filepath.Join(home, "health.json")}}
	settings, _ := manager.SettingsPath(ProviderCodex)
	if err := os.MkdirAll(filepath.Dir(settings), 0o700); err != nil {
		t.Fatal(err)
	}
	malformed := []byte(`{"hooks":{"Stop":"not-an-array"}}`)
	if err := os.WriteFile(settings, malformed, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := manager.PlanInstall(ProviderCodex); err == nil {
		t.Fatal("malformed settings accepted")
	}
	contents, _ := os.ReadFile(settings)
	if !bytes.Equal(contents, malformed) {
		t.Fatalf("malformed settings mutated: %s", contents)
	}
	backups, _ := filepath.Glob(settings + ".beacon-backup-*")
	if len(backups) != 0 {
		t.Fatalf("unexpected backups: %v", backups)
	}
	if status := manager.Status(ProviderCodex); status.State != StateError {
		t.Fatalf("malformed status = %#v", status)
	}
}

func TestStatusDetectsStaleCommandAndExecutable(t *testing.T) {
	home := t.TempDir()
	executable := executableFile(t, home)
	manager := Manager{Home: home, Executable: executable, Health: HealthStore{Path: filepath.Join(home, "health.json")}}
	plan, err := manager.PlanInstall(ProviderCodex)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Apply(plan); err != nil {
		t.Fatal(err)
	}
	settings, _ := manager.SettingsPath(ProviderCodex)
	contents, _ := os.ReadFile(settings)
	contents = bytes.Replace(contents, []byte("--provider codex"), []byte("--provider claude-code"), 1)
	if err := os.WriteFile(settings, contents, 0o600); err != nil {
		t.Fatal(err)
	}
	if status := manager.Status(ProviderCodex); status.State != StateStale {
		t.Fatalf("modified command status = %#v", status)
	}
	repair, err := manager.PlanInstall(ProviderCodex)
	if err != nil {
		t.Fatal(err)
	}
	if err := manager.Apply(repair); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(executable); err != nil {
		t.Fatal(err)
	}
	if status := manager.Status(ProviderCodex); status.State != StateStale {
		t.Fatalf("missing executable status = %#v", status)
	}
}

func TestShellGuardSucceedsQuicklyWhenExecutableIsMissing(t *testing.T) {
	command := Command(ProviderCodex, filepath.Join(t.TempDir(), "moved-beacon"))
	started := time.Now()
	result := exec.Command("/bin/sh", "-c", command)
	if output, err := result.CombinedOutput(); err != nil || len(output) != 0 {
		t.Fatalf("guard output=%q err=%v", output, err)
	}
	if elapsed := time.Since(started); elapsed > 500*time.Millisecond {
		t.Fatalf("guard took %s", elapsed)
	}
}

func TestHealthFileContainsOnlyFingerprintAndObservedFlag(t *testing.T) {
	path := filepath.Join(t.TempDir(), "health.json")
	store := HealthStore{Path: path}
	if err := store.MarkObserved(ProviderCodex, "fingerprint"); err != nil {
		t.Fatal(err)
	}
	contents, _ := os.ReadFile(path)
	for _, forbidden := range []string{"session", "activity", "prompt", "transcript"} {
		if bytes.Contains(contents, []byte(forbidden)) {
			t.Fatalf("health file contains %q: %s", forbidden, contents)
		}
	}
}

func executableFile(t *testing.T, home string) string {
	t.Helper()
	path := filepath.Join(home, "beacon-cli")
	if err := os.WriteFile(path, []byte("#!/bin/sh\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	return path
}
