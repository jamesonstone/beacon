package integrations

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
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

func readSettings(path string) (map[string]any, bool, error) {
	info, statErr := os.Lstat(path)
	if statErr == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return nil, true, errors.New("decode integration settings: path must be a regular file")
		}
	} else if !errors.Is(statErr, os.ErrNotExist) {
		return nil, false, fmt.Errorf("inspect integration settings: %w", statErr)
	}
	contents, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return map[string]any{}, false, nil
	}
	if err != nil {
		return nil, false, fmt.Errorf("read integration settings: %w", err)
	}
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.UseNumber()
	var document map[string]any
	if err := decoder.Decode(&document); err != nil {
		return nil, true, fmt.Errorf("decode integration settings: %w", err)
	}
	if document == nil {
		return nil, true, errors.New("decode integration settings: top-level value must be an object")
	}
	if err := ensureEOF(decoder); err != nil {
		return nil, true, err
	}
	if err := validateHooks(document); err != nil {
		return nil, true, err
	}
	return document, true, nil
}

func ensureEOF(decoder *json.Decoder) error {
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		if err == nil {
			return errors.New("decode integration settings: multiple JSON values")
		}
		return fmt.Errorf("decode integration settings: %w", err)
	}
	return nil
}

func validateHooks(document map[string]any) error {
	value, ok := document["hooks"]
	if !ok {
		return nil
	}
	hooks, ok := value.(map[string]any)
	if !ok {
		return errors.New("decode integration settings: hooks must be an object")
	}
	for event, rawGroups := range hooks {
		groups, ok := rawGroups.([]any)
		if !ok {
			return fmt.Errorf("decode integration settings: hooks.%s must be an array", event)
		}
		for index, rawGroup := range groups {
			group, ok := rawGroup.(map[string]any)
			if !ok {
				return fmt.Errorf("decode integration settings: hooks.%s[%d] must be an object", event, index)
			}
			rawHandlers, ok := group["hooks"]
			if !ok {
				return fmt.Errorf("decode integration settings: hooks.%s[%d].hooks is required", event, index)
			}
			handlers, ok := rawHandlers.([]any)
			if !ok {
				return fmt.Errorf("decode integration settings: hooks.%s[%d].hooks must be an array", event, index)
			}
			for handlerIndex, rawHandler := range handlers {
				if _, ok := rawHandler.(map[string]any); !ok {
					return fmt.Errorf("decode integration settings: hooks.%s[%d].hooks[%d] must be an object", event, index, handlerIndex)
				}
			}
		}
	}
	return nil
}

func removeMarked(document map[string]any, changes *[]Change) {
	hooks, ok := document["hooks"].(map[string]any)
	if !ok {
		return
	}
	for event, rawGroups := range hooks {
		groups := rawGroups.([]any)
		keptGroups := make([]any, 0, len(groups))
		for _, rawGroup := range groups {
			group := rawGroup.(map[string]any)
			handlers := group["hooks"].([]any)
			keptHandlers := make([]any, 0, len(handlers))
			for _, rawHandler := range handlers {
				handler := rawHandler.(map[string]any)
				command, _ := handler["command"].(string)
				if isMarked(command) {
					if changes != nil {
						*changes = append(*changes, Change{Operation: "remove", Event: event, Matcher: stringValue(group["matcher"]), Command: command})
					}
					continue
				}
				keptHandlers = append(keptHandlers, rawHandler)
			}
			if len(keptHandlers) == 0 {
				continue
			}
			group["hooks"] = keptHandlers
			keptGroups = append(keptGroups, group)
		}
		if len(keptGroups) == 0 {
			delete(hooks, event)
		} else {
			hooks[event] = keptGroups
		}
	}
	if len(hooks) == 0 {
		delete(document, "hooks")
	}
}

func installSpecs(document map[string]any, specs []hookSpec, changes *[]Change) {
	hooks, ok := document["hooks"].(map[string]any)
	if !ok {
		hooks = map[string]any{}
		document["hooks"] = hooks
	}
	for _, spec := range specs {
		group := map[string]any{
			"hooks": []any{map[string]any{"type": "command", "command": spec.Command, "timeout": json.Number("2")}},
		}
		if spec.Matcher != "" {
			group["matcher"] = spec.Matcher
		}
		groups, _ := hooks[spec.Event].([]any)
		hooks[spec.Event] = append(groups, group)
		if changes != nil {
			*changes = append(*changes, Change{Operation: "add", Event: spec.Event, Matcher: spec.Matcher, Command: spec.Command})
		}
	}
}

func markedHandlers(document map[string]any) []Change {
	copy := cloneDocument(document)
	changes := make([]Change, 0)
	removeMarked(copy, &changes)
	return changes
}

func hasExactHandlers(document map[string]any, expected []hookSpec) bool {
	hooks, ok := document["hooks"].(map[string]any)
	if !ok {
		return false
	}
	counts := make(map[string]int, len(expected))
	for _, spec := range expected {
		counts[specKey(spec.Event, spec.Matcher, spec.Command)]++
	}
	found := 0
	for event, rawGroups := range hooks {
		for _, rawGroup := range rawGroups.([]any) {
			group := rawGroup.(map[string]any)
			matcher := stringValue(group["matcher"])
			for _, rawHandler := range group["hooks"].([]any) {
				handler := rawHandler.(map[string]any)
				command, _ := handler["command"].(string)
				if !isMarked(command) {
					continue
				}
				if handler["type"] != "command" || !timeoutIsTwo(handler["timeout"]) {
					return false
				}
				key := specKey(event, matcher, command)
				if counts[key] == 0 {
					return false
				}
				counts[key]--
				found++
			}
		}
	}
	if found != len(expected) {
		return false
	}
	for _, count := range counts {
		if count != 0 {
			return false
		}
	}
	return true
}

func timeoutIsTwo(value any) bool {
	switch number := value.(type) {
	case json.Number:
		parsed, err := number.Float64()
		return err == nil && parsed == 2
	case float64:
		return number == 2
	case int:
		return number == 2
	default:
		return false
	}
}

func specKey(event, matcher, command string) string {
	return event + "\x00" + matcher + "\x00" + command
}

func isMarked(command string) bool {
	return strings.Contains(command, "# "+Marker)
}

func cloneDocument(document map[string]any) map[string]any {
	contents, _ := json.Marshal(document)
	decoder := json.NewDecoder(bytes.NewReader(contents))
	decoder.UseNumber()
	var clone map[string]any
	_ = decoder.Decode(&clone)
	return clone
}

func marshalSettings(document map[string]any) ([]byte, error) {
	contents, err := json.MarshalIndent(document, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("encode integration settings: %w", err)
	}
	return append(contents, '\n'), nil
}

func atomicWrite(path string, contents []byte) error {
	directory := filepath.Dir(path)
	if err := os.MkdirAll(directory, 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(directory, ".beacon-integration-*")
	if err != nil {
		return err
	}
	temporary := file.Name()
	defer os.Remove(temporary)
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(contents); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(temporary, path); err != nil {
		return err
	}
	return os.Chmod(path, 0o600)
}

func writeBackup(path string, contents []byte) error {
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	if err := file.Chmod(0o600); err != nil {
		file.Close()
		return err
	}
	if _, err := file.Write(contents); err != nil {
		file.Close()
		return err
	}
	if err := file.Sync(); err != nil {
		file.Close()
		return err
	}
	return file.Close()
}

func availableBackupPath(path string) (string, error) {
	if _, err := os.Lstat(path); errors.Is(err, os.ErrNotExist) {
		return path, nil
	} else if err != nil {
		return "", fmt.Errorf("inspect integration backup path: %w", err)
	}
	for suffix := 1; ; suffix++ {
		candidate := fmt.Sprintf("%s-%d", path, suffix)
		if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
			return candidate, nil
		} else if err != nil {
			return "", fmt.Errorf("inspect integration backup path: %w", err)
		}
	}
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func isExecutable(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.Mode().IsRegular() && info.Mode().Perm()&0o111 != 0
}

func stringValue(value any) string {
	result, _ := value.(string)
	return result
}
