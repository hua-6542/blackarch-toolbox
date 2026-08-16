package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"blackarch-toolbox/internal/model"
)

type Manager struct {
	Root string
}

func New(root string) *Manager {
	return &Manager{Root: root}
}

func (m *Manager) CreateRunDir(tool string, now time.Time) (string, error) {
	dir := filepath.Join(m.Root, tool, now.Format("2006-01-02_15-04-05"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (m *Manager) WriteMetadata(dir string, meta model.Metadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ".metadata.json"), data, 0o644)
}

func (m *Manager) Resolve(rel string) (string, error) {
	if rel == "" {
		rel = "."
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("非法路径: %q", rel)
	}
	clean := filepath.Clean(filepath.Join(m.Root, rel))
	if clean != m.Root && !strings.HasPrefix(clean, m.Root+string(filepath.Separator)) {
		return "", fmt.Errorf("非法路径: %q", rel)
	}
	return clean, nil
}

func (m *Manager) List(rel string) ([]model.FileEntry, error) {
	abs, err := m.Resolve(rel)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, err
	}
	des, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	entries := make([]model.FileEntry, 0, len(des))
	for _, de := range des {
		info, err := de.Info()
		if err != nil {
			continue
		}
		childAbs := filepath.Join(abs, de.Name())
		relPath, _ := filepath.Rel(m.Root, childAbs)
		entries = append(entries, model.FileEntry{
			Name:  de.Name(),
			Path:  relPath,
			IsDir: de.IsDir(),
			Size:  info.Size(),
		})
	}
	return entries, nil
}

func (m *Manager) EnsureWritable() error {
	if err := os.MkdirAll(m.Root, 0o755); err != nil {
		return err
	}
	probe := filepath.Join(m.Root, ".write-probe")
	if err := os.WriteFile(probe, []byte("x"), 0o644); err != nil {
		return fmt.Errorf("产物目录不可写: %w", err)
	}
	return os.Remove(probe)
}
