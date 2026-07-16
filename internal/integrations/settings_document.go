package integrations

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"strings"
)

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

func stringValue(value any) string {
	result, _ := value.(string)
	return result
}
