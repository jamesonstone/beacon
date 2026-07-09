package output

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/jamesonstone/beacon/internal/model"
)

func TestJSONEmitsVersionedSnapshot(t *testing.T) {
	snapshot := model.Snapshot{SchemaVersion: 1, GeneratedAt: time.Now(), Groups: model.Groups{}, Lanes: []model.Lane{}, Errors: []model.ScanError{}}
	var buffer bytes.Buffer
	if err := JSON(&buffer, snapshot); err != nil {
		t.Fatal(err)
	}
	var decoded map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &decoded); err != nil {
		t.Fatal(err)
	}
	if decoded["schema_version"] != float64(1) {
		t.Fatalf("JSON = %s", buffer.String())
	}
}

func TestJSONEmitsEmptyCollectionsAsArrays(t *testing.T) {
	snapshot := model.Snapshot{
		SchemaVersion: 1,
		Refresh:       []model.Refresh{},
		Groups:        model.Groups{Ready: []string{}, Action: []string{}, Waiting: []string{}, Idle: []string{}},
		Lanes:         []model.Lane{},
		Errors:        []model.ScanError{},
	}
	var buffer bytes.Buffer
	if err := JSON(&buffer, snapshot); err != nil {
		t.Fatal(err)
	}
	for _, expected := range []string{`"refresh": []`, `"ready": []`, `"lanes": []`, `"errors": []`} {
		if !strings.Contains(buffer.String(), expected) {
			t.Fatalf("JSON missing %s: %s", expected, buffer.String())
		}
	}
}

func TestTerminalGroupsLanes(t *testing.T) {
	snapshot := model.Snapshot{
		GeneratedAt: time.Now(), Groups: model.Groups{Ready: []string{"lane"}},
		Lanes: []model.Lane{{ID: "lane", Repository: "example", Branch: "feature", ReviewReady: true, NextAction: model.ActionReviewPR}},
	}
	var buffer bytes.Buffer
	if err := Terminal(&buffer, snapshot); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buffer.String(), "Ready for Review") || !strings.Contains(buffer.String(), "review pull request") {
		t.Fatalf("terminal output = %q", buffer.String())
	}
}
