package agent

import (
	"crypto/sha256"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/jamesonstone/beacon/internal/checkoutwarn"
	"github.com/jamesonstone/beacon/internal/model"
	"github.com/jamesonstone/beacon/internal/scan"
	"github.com/jamesonstone/beacon/internal/tracking"
)

const CacheVersion = 2

type ProjectRecord struct {
	Version               int                         `json:"version"`
	ProjectID             string                      `json:"project_id"`
	Revision              uint64                      `json:"revision"`
	Stage                 string                      `json:"stage"`
	UpdatedAt             time.Time                   `json:"updated_at"`
	LastProbeAt           time.Time                   `json:"last_probe_at,omitempty"`
	CheckoutConfirmations []checkoutwarn.Confirmation `json:"checkout_confirmations,omitempty"`
	Snapshot              model.Snapshot              `json:"snapshot"`
}

type Cache struct {
	Directory string
	Now       func() time.Time
}

func (c Cache) LoadAll() ([]ProjectRecord, []error) {
	entries, err := os.ReadDir(c.Directory)
	if errors.Is(err, os.ErrNotExist) {
		return []ProjectRecord{}, nil
	}
	if err != nil {
		return nil, []error{fmt.Errorf("read project cache: %w", err)}
	}
	records := make([]ProjectRecord, 0, len(entries))
	var failures []error
	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".json" {
			continue
		}
		path := filepath.Join(c.Directory, entry.Name())
		record, err := c.load(path)
		if err != nil {
			failures = append(failures, err)
			if quarantineErr := c.quarantine(path); quarantineErr != nil {
				failures = append(failures, quarantineErr)
			}
			continue
		}
		records = append(records, record)
	}
	sort.Slice(records, func(i, j int) bool { return records[i].ProjectID < records[j].ProjectID })
	return records, failures
}

func (c Cache) Write(record ProjectRecord) error {
	if record.ProjectID == "" {
		return errors.New("cache project_id is required")
	}
	if record.Version == 0 {
		record.Version = CacheVersion
	}
	if record.Version != CacheVersion {
		return fmt.Errorf("cache version must equal %d", CacheVersion)
	}
	if err := os.MkdirAll(c.Directory, 0o700); err != nil {
		return fmt.Errorf("create project cache: %w", err)
	}
	contents, err := json.MarshalIndent(record, "", "  ")
	if err != nil {
		return fmt.Errorf("encode project cache %s: %w", record.ProjectID, err)
	}
	contents = append(contents, '\n')
	path := filepath.Join(c.Directory, ProjectFileName(record.ProjectID))
	file, err := os.CreateTemp(c.Directory, ".beacon-project-*.json")
	if err != nil {
		return fmt.Errorf("create project cache temporary file: %w", err)
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
		return fmt.Errorf("replace project cache %s: %w", path, err)
	}
	return nil
}

func ProjectFileName(projectID string) string {
	return fmt.Sprintf("%x.json", sha256.Sum256([]byte(projectID)))
}

func (c Cache) load(path string) (ProjectRecord, error) {
	file, err := os.Open(path)
	if err != nil {
		return ProjectRecord{}, fmt.Errorf("open project cache %s: %w", path, err)
	}
	defer file.Close()
	decoder := json.NewDecoder(file)
	decoder.DisallowUnknownFields()
	var record ProjectRecord
	if err := decoder.Decode(&record); err != nil {
		return ProjectRecord{}, fmt.Errorf("decode project cache %s: %w", path, err)
	}
	var extra any
	if err := decoder.Decode(&extra); !errors.Is(err, io.EOF) {
		return ProjectRecord{}, fmt.Errorf("decode project cache %s: trailing JSON", path)
	}
	if (record.Version != 1 && record.Version != CacheVersion) || record.ProjectID == "" {
		return ProjectRecord{}, fmt.Errorf("validate project cache %s: unsupported or incomplete record", path)
	}
	record.Version = CacheVersion
	switch record.Snapshot.SchemaVersion {
	case model.SchemaVersion:
	case 2:
		record.Snapshot.SchemaVersion = model.SchemaVersion
		scan.Finalize(&record.Snapshot)
	default:
		return ProjectRecord{}, fmt.Errorf("validate project cache %s: unsupported snapshot schema %d", path, record.Snapshot.SchemaVersion)
	}
	return record, nil
}

func (c Cache) quarantine(path string) error {
	now := time.Now
	if c.Now != nil {
		now = c.Now
	}
	target := fmt.Sprintf("%s.corrupt-%s", path, now().UTC().Format("20060102T150405.000000000Z"))
	if err := os.Rename(path, target); err != nil {
		return fmt.Errorf("quarantine corrupt cache %s: %w", path, err)
	}
	return nil
}

func Assemble(records []ProjectRecord, configPath, trackingPath string, now time.Time) model.Snapshot {
	return AssembleWithRecentWindow(records, configPath, trackingPath, now, 24*time.Hour)
}

func AssembleWithRecentWindow(records []ProjectRecord, configPath, trackingPath string, now time.Time, recentWindow time.Duration) model.Snapshot {
	snapshot := model.Snapshot{
		SchemaVersion: model.SchemaVersion, ConfigPath: configPath,
		Refresh: []model.Refresh{}, Projects: []model.Project{}, Lanes: []model.Lane{},
		Errors: []model.ScanError{}, Warnings: []model.ScanError{},
	}
	for _, record := range records {
		cached := record.Snapshot
		snapshot.Projects = append(snapshot.Projects, cached.Projects...)
		snapshot.Lanes = append(snapshot.Lanes, cached.Lanes...)
		snapshot.Refresh = append(snapshot.Refresh, cached.Refresh...)
		snapshot.Errors = append(snapshot.Errors, cached.Errors...)
		snapshot.Warnings = append(snapshot.Warnings, cached.Warnings...)
		if cached.GeneratedAt.After(snapshot.GeneratedAt) {
			snapshot.GeneratedAt = cached.GeneratedAt
		}
	}
	if snapshot.GeneratedAt.IsZero() {
		snapshot.GeneratedAt = now
	}
	sort.Slice(snapshot.Projects, func(i, j int) bool {
		if snapshot.Projects[i].Name != snapshot.Projects[j].Name {
			return snapshot.Projects[i].Name < snapshot.Projects[j].Name
		}
		return snapshot.Projects[i].GitHub < snapshot.Projects[j].GitHub
	})
	scan.Finalize(&snapshot)
	tracking.ApplyCached(&snapshot, trackingPath, now, recentWindow)
	return snapshot
}
