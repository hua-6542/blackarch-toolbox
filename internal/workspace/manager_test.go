package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"blackarch-toolbox/internal/model"
)

func newMgr(t *testing.T) *Manager {
	t.Helper()
	m := New(filepath.Join(t.TempDir(), "ws"))
	if err := os.MkdirAll(m.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestCreateRunDir(t *testing.T) {
	m := newMgr(t)
	dir, err := m.CreateRunDir("nmap", time.Date(2026, 8, 16, 14, 30, 22, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(m.Root, "nmap", "2026-08-16_14-30-22")
	if dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("目录应已创建: %v", err)
	}
}

func TestWriteAndReadMetadata(t *testing.T) {
	m := newMgr(t)
	dir, _ := m.CreateRunDir("nmap", time.Now())
	meta := model.Metadata{
		Tool:        "nmap",
		Environment: "local",
		ExecutedAt:  time.Now(),
		Command:     "nmap -sV 1.2.3.4",
		ExitCode:    0,
		OutputDir:   dir,
	}
	if err := m.WriteMetadata(dir, meta); err != nil {
		t.Fatal(err)
	}
	entry, err := os.Stat(filepath.Join(dir, ".metadata.json"))
	if err != nil {
		t.Fatalf(".metadata.json 应存在: %v", err)
	}
	if entry.Size() == 0 {
		t.Fatal("metadata 为空")
	}
}

func TestListEntries(t *testing.T) {
	m := newMgr(t)
	dir, _ := m.CreateRunDir("nmap", time.Now())
	os.WriteFile(filepath.Join(dir, "scan.txt"), []byte("data"), 0o644)
	os.WriteFile(filepath.Join(dir, ".metadata.json"), []byte("{}"), 0o644)
	entries, err := m.List("nmap")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != filepath.Base(dir) || !entries[0].IsDir {
		t.Fatalf("List(nmap) = %+v", entries)
	}
	files, err := m.List(entries[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("List(run dir) = %+v, want 2 文件", files)
	}
}

func TestResolveBlocksTraversal(t *testing.T) {
	m := newMgr(t)
	if _, err := m.Resolve("../etc"); err == nil {
		t.Fatal("路径穿越应被拒绝")
	}
	if _, err := m.Resolve("a/../../b"); err == nil {
		t.Fatal("间接穿越应被拒绝")
	}
	if _, err := m.Resolve("nmap/2026-01-01_00-00-00"); err != nil {
		t.Fatalf("正常路径应通过: %v", err)
	}
	if _, err := m.Resolve("/etc/passwd"); err == nil {
		t.Fatal("绝对路径应被拒绝")
	}
}

func TestEnsureWritable(t *testing.T) {
	m := newMgr(t)
	if err := m.EnsureWritable(); err != nil {
		t.Fatalf("应可写: %v", err)
	}
	if err := os.Chmod(m.Root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(m.Root, 0o755) })
	if err := m.EnsureWritable(); err == nil {
		t.Fatal("只读目录应报错")
	}
}

func TestListRootWithEmptyRel(t *testing.T) {
	m := newMgr(t)
	m.CreateRunDir("nmap", time.Now())
	entries, err := m.List("")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || !entries[0].IsDir {
		t.Fatalf("List(\"\") = %+v", entries)
	}
}

func TestListRejectsTraversal(t *testing.T) {
	m := newMgr(t)
	if _, err := m.List("../etc"); err == nil {
		t.Fatal("穿越应被拒绝")
	}
}

func TestListMissingDir(t *testing.T) {
	m := newMgr(t)
	if _, err := m.List("nope"); err == nil {
		t.Fatal("不存在的目录应报错")
	}
}
