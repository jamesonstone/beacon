package integrations

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	ProviderCodex      = "codex"
	ProviderClaudeCode = "claude-code"

	StateNotInstalled = "not_installed"
	StateInstalled    = "installed"
	StateActive       = "active"
	StateStale        = "stale"
	StateError        = "error"

	Marker = "beacon-activity-v1"
)

type Status struct {
	Provider     string `json:"provider"`
	State        string `json:"state"`
	SettingsPath string `json:"settings_path"`
	Message      string `json:"message,omitempty"`
}

type Change struct {
	Operation string `json:"operation"`
	Event     string `json:"event"`
	Matcher   string `json:"matcher,omitempty"`
	Command   string `json:"command"`
}

type Plan struct {
	Provider     string   `json:"provider"`
	Action       string   `json:"action"`
	SettingsPath string   `json:"settings_path"`
	BackupPath   string   `json:"backup_path,omitempty"`
	Changes      []Change `json:"changes"`
	Changed      bool     `json:"changed"`

	beforeExists bool
	before       []byte
	after        []byte
	fingerprint  string
}

type Manager struct {
	Home       string
	Executable string
	Health     HealthStore
	Now        func() time.Time
}

type hookSpec struct {
	Event   string `json:"event"`
	Matcher string `json:"matcher,omitempty"`
	Command string `json:"command"`
}

func (m Manager) SettingsPath(provider string) (string, error) {
	if err := ValidateProvider(provider); err != nil {
		return "", err
	}
	home := m.Home
	if home == "" {
		var err error
		home, err = os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("resolve home directory: %w", err)
		}
	}
	if provider == ProviderCodex {
		return filepath.Join(home, ".codex", "hooks.json"), nil
	}
	return filepath.Join(home, ".claude", "settings.json"), nil
}

func (m Manager) PlanInstall(provider string) (Plan, error) {
	return m.plan(provider, "install")
}

func (m Manager) PlanUninstall(provider string) (Plan, error) {
	return m.plan(provider, "uninstall")
}

func (m Manager) Apply(plan Plan) error {
	if plan.Action != "install" && plan.Action != "uninstall" {
		return fmt.Errorf("unsupported integration action %q", plan.Action)
	}
	if plan.Changed {
		if plan.beforeExists {
			contents, err := os.ReadFile(plan.SettingsPath)
			if err != nil {
				return fmt.Errorf("read integration settings for backup: %w", err)
			}
			current, _, decodeErr := readSettings(plan.SettingsPath)
			if decodeErr != nil {
				return decodeErr
			}
			canonical, encodeErr := marshalSettings(current)
			if encodeErr != nil {
				return encodeErr
			}
			if !bytes.Equal(canonical, plan.before) {
				return errors.New("integration settings changed after preview; preview again")
			}
			if err := writeBackup(plan.BackupPath, contents); err != nil {
				return fmt.Errorf("back up integration settings: %w", err)
			}
		} else if _, err := os.Lstat(plan.SettingsPath); err == nil {
			return errors.New("integration settings appeared after preview; preview again")
		} else if !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("inspect integration settings after preview: %w", err)
		}
		if err := atomicWrite(plan.SettingsPath, plan.after); err != nil {
			return fmt.Errorf("write integration settings: %w", err)
		}
	}
	if plan.Action == "install" {
		if plan.Changed {
			return m.Health.Reset(plan.Provider, plan.fingerprint)
		}
		return nil
	}
	return m.Health.Remove(plan.Provider)
}

func (m Manager) Status(provider string) Status {
	path, err := m.SettingsPath(provider)
	if err != nil {
		return Status{Provider: provider, State: StateError, Message: err.Error()}
	}
	status := Status{Provider: provider, State: StateNotInstalled, SettingsPath: path}
	document, exists, err := readSettings(path)
	if err != nil {
		status.State, status.Message = StateError, err.Error()
		return status
	}
	if !exists {
		return status
	}
	marked := markedHandlers(document)
	if len(marked) == 0 {
		return status
	}
	executable, err := m.executable()
	if err != nil {
		status.State, status.Message = StateStale, err.Error()
		return status
	}
	if !hasExactHandlers(document, desiredSpecs(provider, executable)) || !isExecutable(executable) {
		status.State = StateStale
		status.Message = "configured Beacon handlers, marker, or executable do not match"
		return status
	}
	status.State = StateInstalled
	fingerprint := Fingerprint(provider, executable)
	entry, healthErr := m.Health.Entry(provider)
	if healthErr != nil {
		status.State, status.Message = StateError, healthErr.Error()
		return status
	}
	if entry.Observed && entry.Fingerprint == fingerprint {
		status.State = StateActive
	}
	return status
}

func (m Manager) Observe(provider string) error {
	if err := ValidateProvider(provider); err != nil {
		return err
	}
	executable, err := m.executable()
	if err != nil {
		return err
	}
	return m.Health.MarkObserved(provider, Fingerprint(provider, executable))
}

func ValidateProvider(provider string) error {
	if provider != ProviderCodex && provider != ProviderClaudeCode {
		return fmt.Errorf("unsupported integration %q (want codex or claude-code)", provider)
	}
	return nil
}

func Fingerprint(provider, executable string) string {
	contents, _ := json.Marshal(desiredSpecs(provider, filepath.Clean(executable)))
	sum := sha256.Sum256(contents)
	return hex.EncodeToString(sum[:])
}

func Command(provider, executable string) string {
	return fmt.Sprintf("%s activity ingest --hook --provider %s >/dev/null 2>&1 || true # %s", shellQuote(filepath.Clean(executable)), provider, Marker)
}

func (m Manager) plan(provider, action string) (Plan, error) {
	if err := ValidateProvider(provider); err != nil {
		return Plan{}, err
	}
	path, err := m.SettingsPath(provider)
	if err != nil {
		return Plan{}, err
	}
	document, exists, err := readSettings(path)
	if err != nil {
		return Plan{}, err
	}
	executable, err := m.executable()
	if err != nil {
		return Plan{}, err
	}
	original, err := marshalSettings(document)
	if err != nil {
		return Plan{}, err
	}
	specs := desiredSpecs(provider, executable)
	changes := make([]Change, 0)
	if action != "install" || !hasExactHandlers(document, specs) {
		removeMarked(document, &changes)
		if action == "install" {
			installSpecs(document, specs, &changes)
		}
	}
	after, err := marshalSettings(document)
	if err != nil {
		return Plan{}, err
	}
	changed := !bytes.Equal(original, after)
	if !changed {
		changes = nil
	}
	plan := Plan{
		Provider: provider, Action: action, SettingsPath: path, Changes: changes, Changed: changed,
		beforeExists: exists, before: original, after: after, fingerprint: Fingerprint(provider, executable),
	}
	if changed && exists {
		now := time.Now()
		if m.Now != nil {
			now = m.Now()
		}
		plan.BackupPath, err = availableBackupPath(path + ".beacon-backup-" + now.UTC().Format("20060102T150405.000000000Z"))
		if err != nil {
			return Plan{}, err
		}
	}
	return plan, nil
}

func (m Manager) executable() (string, error) {
	value := m.Executable
	if value == "" {
		var err error
		value, err = os.Executable()
		if err != nil {
			return "", fmt.Errorf("resolve Beacon executable: %w", err)
		}
	}
	absolute, err := filepath.Abs(value)
	if err != nil {
		return "", fmt.Errorf("resolve Beacon executable: %w", err)
	}
	return filepath.Clean(absolute), nil
}

func desiredSpecs(provider, executable string) []hookSpec {
	command := Command(provider, executable)
	if provider == ProviderCodex {
		return []hookSpec{{Event: "UserPromptSubmit", Command: command}, {Event: "PermissionRequest", Command: command}, {Event: "Stop", Command: command}}
	}
	return []hookSpec{
		{Event: "UserPromptSubmit", Command: command},
		{Event: "PermissionRequest", Command: command},
		{Event: "Notification", Matcher: "permission_prompt|idle_prompt|elicitation_dialog|agent_needs_input", Command: command},
		{Event: "Stop", Command: command},
		{Event: "StopFailure", Command: command},
		{Event: "SessionEnd", Command: command},
	}
}
