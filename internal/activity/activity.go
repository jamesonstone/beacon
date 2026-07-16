package activity

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"
)

const (
	ProviderCodex      = "codex"
	ProviderClaudeCode = "claude-code"

	StateWorking        = "working"
	StateNeedsAttention = "needs_attention"
	StateTurnFinished   = "turn_finished"

	MaxHookBytes = 32 << 10
)

var attentionNotifications = map[string]struct{}{
	"permission_prompt":  {},
	"idle_prompt":        {},
	"elicitation_dialog": {},
	"agent_needs_input":  {},
}

type Action string

const (
	ActionNone   Action = ""
	ActionUpsert Action = "upsert"
	ActionRemove Action = "remove"
)

type Event struct {
	Provider   string
	Name       string
	SessionKey string
	CWD        string
	ObservedAt time.Time
	State      string
	Action     Action
	WellFormed bool
}

type hookPayload struct {
	HookEventName    string          `json:"hook_event_name"`
	SessionID        string          `json:"session_id"`
	CWD              string          `json:"cwd"`
	NotificationType string          `json:"notification_type"`
	Timestamp        json.RawMessage `json:"timestamp"`
}

func Decode(provider string, input io.Reader, receivedAt time.Time) (Event, error) {
	if provider != ProviderCodex && provider != ProviderClaudeCode {
		return Event{}, fmt.Errorf("unsupported activity provider %q", provider)
	}
	contents, err := io.ReadAll(io.LimitReader(input, MaxHookBytes+1))
	if err != nil {
		return Event{}, fmt.Errorf("read hook input: %w", err)
	}
	if len(contents) > MaxHookBytes {
		return Event{}, errors.New("hook input exceeds 32 KiB")
	}
	var payload hookPayload
	if err := json.Unmarshal(contents, &payload); err != nil {
		return Event{}, fmt.Errorf("decode hook input: %w", err)
	}
	payload.HookEventName = strings.TrimSpace(payload.HookEventName)
	payload.SessionID = strings.TrimSpace(payload.SessionID)
	payload.CWD = strings.TrimSpace(payload.CWD)
	payload.NotificationType = strings.TrimSpace(payload.NotificationType)
	if payload.HookEventName == "" || payload.SessionID == "" || payload.CWD == "" {
		return Event{}, errors.New("hook input requires hook_event_name, session_id, and cwd")
	}
	if !supportedEvent(provider, payload.HookEventName) {
		return Event{}, fmt.Errorf("unsupported %s hook event %q", provider, payload.HookEventName)
	}
	observedAt, err := decodeTimestamp(payload.Timestamp, receivedAt)
	if err != nil {
		return Event{}, err
	}
	event := Event{
		Provider: provider, Name: payload.HookEventName, SessionKey: HashSession(provider, payload.SessionID),
		CWD: payload.CWD, ObservedAt: observedAt, WellFormed: true,
	}
	switch payload.HookEventName {
	case "UserPromptSubmit":
		event.Action, event.State = ActionUpsert, StateWorking
	case "PermissionRequest":
		event.Action, event.State = ActionUpsert, StateNeedsAttention
	case "Notification":
		if _, ok := attentionNotifications[payload.NotificationType]; ok {
			event.Action, event.State = ActionUpsert, StateNeedsAttention
		}
	case "Stop":
		event.Action, event.State = ActionUpsert, StateTurnFinished
	case "StopFailure", "SessionEnd":
		event.Action = ActionRemove
	}
	return event, nil
}

func HashSession(provider, sessionID string) string {
	sum := sha256.Sum256([]byte(provider + "\x00" + sessionID))
	return hex.EncodeToString(sum[:])
}

func TTL(state string) time.Duration {
	switch state {
	case StateWorking:
		return 2 * time.Hour
	case StateNeedsAttention:
		return 24 * time.Hour
	case StateTurnFinished:
		return time.Hour
	default:
		return 0
	}
}

func supportedEvent(provider, name string) bool {
	switch provider {
	case ProviderCodex:
		return name == "UserPromptSubmit" || name == "PermissionRequest" || name == "Stop"
	case ProviderClaudeCode:
		switch name {
		case "UserPromptSubmit", "PermissionRequest", "Notification", "Stop", "StopFailure", "SessionEnd":
			return true
		}
	}
	return false
}

func decodeTimestamp(raw json.RawMessage, fallback time.Time) (time.Time, error) {
	if len(bytes.TrimSpace(raw)) == 0 || bytes.Equal(bytes.TrimSpace(raw), []byte("null")) {
		return fallback, nil
	}
	var value string
	if err := json.Unmarshal(raw, &value); err == nil {
		parsed, parseErr := time.Parse(time.RFC3339Nano, value)
		if parseErr != nil {
			return time.Time{}, fmt.Errorf("decode hook timestamp: %w", parseErr)
		}
		return parsed, nil
	}
	var number json.Number
	decoder := json.NewDecoder(bytes.NewReader(raw))
	decoder.UseNumber()
	if err := decoder.Decode(&number); err != nil {
		return time.Time{}, errors.New("decode hook timestamp: expected RFC3339 string or Unix time")
	}
	seconds, err := strconv.ParseFloat(number.String(), 64)
	if err != nil {
		return time.Time{}, errors.New("decode hook timestamp: expected RFC3339 string or Unix time")
	}
	whole := int64(seconds)
	nanos := int64((seconds - float64(whole)) * float64(time.Second))
	return time.Unix(whole, nanos), nil
}
