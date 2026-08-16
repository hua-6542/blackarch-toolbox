package db

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	_ "blackarch-toolbox/internal/model"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestOpenMigratesAndSeeds(t *testing.T) {
	d := newTestDB(t)
	var n int
	if err := d.conn.QueryRow("SELECT COUNT(*) FROM tools").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("seed 后 tools 行数 = %d, want 3", n)
	}
	for _, table := range []string{"tools", "executions", "preferences"} {
		var name string
		err := d.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Fatalf("表 %s 不存在: %v", table, err)
		}
	}
}

func TestSeedIdempotent(t *testing.T) {
	d := newTestDB(t)
	d.Seed()
	var n int
	d.conn.QueryRow("SELECT COUNT(*) FROM tools").Scan(&n)
	if n != 3 {
		t.Fatalf("重复 seed 后 = %d, want 3", n)
	}
}

func TestListToolsByCategory(t *testing.T) {
	d := newTestDB(t)
	all, err := d.ListTools("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("ListTools(\"\") = %d, want 3", len(all))
	}
	scan, err := d.ListTools("scanner")
	if err != nil {
		t.Fatal(err)
	}
	if len(scan) != 1 || scan[0].Name != "nmap" {
		t.Fatalf("ListTools(scanner) = %+v", scan)
	}
}

func TestGetTool(t *testing.T) {
	d := newTestDB(t)
	tool, err := d.GetTool("nmap")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Name != "nmap" || tool.Category != "scanner" {
		t.Fatalf("GetTool = %+v", tool)
	}
	if _, err := d.GetTool("不存在"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("不存在工具应返回 sql.ErrNoRows, got %v", err)
	}
}

func TestPreferences(t *testing.T) {
	d := newTestDB(t)
	tool, _ := d.GetTool("nmap")
	if _, ok, err := d.GetPreference(int64(tool.ID)); err != nil || ok {
		t.Fatalf("初始偏好应为空, ok=%v err=%v", ok, err)
	}
	if err := d.SetPreference(int64(tool.ID), "podman"); err != nil {
		t.Fatal(err)
	}
	env, ok, err := d.GetPreference(int64(tool.ID))
	if err != nil || !ok || env != "podman" {
		t.Fatalf("偏好读取失败: %q %v %v", env, ok, err)
	}
	if err := d.SetPreference(int64(tool.ID), "vm"); err != nil {
		t.Fatal(err)
	}
	if env, _, _ := d.GetPreference(int64(tool.ID)); env != "vm" {
		t.Fatalf("偏好 upsert 失败: %q", env)
	}
}

func TestExecutions(t *testing.T) {
	d := newTestDB(t)
	tool, _ := d.GetTool("nmap")
	id, err := d.StartExecution(int64(tool.ID), "local", "-sV x", "/tmp/out")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.FinishExecution(id, 0); err != nil {
		t.Fatal(err)
	}
	if err := d.IncrementUseCount(int64(tool.ID)); err != nil {
		t.Fatal(err)
	}
	list, err := d.RecentExecutions(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ToolName != "nmap" || list[0].EnvUsed != "local" || list[0].ExitCode != 0 {
		t.Fatalf("RecentExecutions = %+v", list)
	}
	updated, _ := d.GetTool("nmap")
	if updated.UseCount != 1 {
		t.Fatalf("use_count = %d, want 1", updated.UseCount)
	}
}

func TestGetPreferenceNotFound(t *testing.T) {
	d := newTestDB(t)
	if _, ok, err := d.GetPreference(999); err != nil || ok {
		t.Fatalf("不存在的 tool_id: ok=%v err=%v", ok, err)
	}
}
