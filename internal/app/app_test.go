package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"blackarch-toolbox/internal/config"
	"blackarch-toolbox/internal/executor"
	"blackarch-toolbox/internal/model"
)

type fakeRunner struct {
	mu    sync.Mutex
	calls []string
	lines []string
	code  int
	sleep time.Duration
}

func (f *fakeRunner) Run(ctx context.Context, tool, args, workDir string, onLine func(string)) (int, error) {
	f.mu.Lock()
	f.calls = append(f.calls, tool+"|"+args+"|"+workDir)
	f.mu.Unlock()
	if f.sleep > 0 {
		time.Sleep(f.sleep)
	}
	for _, l := range f.lines {
		onLine(l)
	}
	return f.code, nil
}

func newTestApp(t *testing.T) (*App, *fakeRunner, chan string) {
	t.Helper()
	cfg := &config.Config{}
	cfg.VM.Host = "192.168.122.2"
	cfg.VM.User = "blackarch"
	cfg.VM.Port = 22
	cfg.VM.Name = "blackarch"
	cfg.Workspace.Path = filepath.Join(t.TempDir(), "ws")
	cfg.Podman.Container = "blackarch-tools"
	a, err := New(cfg, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	events := make(chan string, 100)
	a.SetEventEmitter(func(event string, _ ...any) { events <- event })
	fr := &fakeRunner{lines: []string{"line1", "line2"}, code: 0}
	a.Exec = map[string]executor.Runner{"local": fr, "podman": fr, "vm": fr}
	t.Cleanup(func() { a.DB.Close() })
	return a, fr, events
}

func TestGetTools(t *testing.T) {
	a, _, _ := newTestApp(t)
	tools, err := a.GetTools("")
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) < 100 {
		t.Fatalf("工具数 = %d, want >= 100", len(tools))
	}
	scanners, err := a.GetTools("scanner")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range scanners {
		if s.Category != "scanner" {
			t.Fatalf("分类过滤失败: %+v", s)
		}
	}
}

func TestRunToolEndToEnd(t *testing.T) {
	a, fr, events := newTestApp(t)
	res, err := a.RunTool(model.RunRequest{Tool: "nmap", Args: "-sV 192.168.1.1", Env: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExecutionID == 0 || res.EnvUsed == "" || res.WorkDir == "" {
		t.Fatalf("RunResult 不完整: %+v", res)
	}
	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev == EventLogEndPrefix+strconv.FormatInt(res.ExecutionID, 10) {
				goto done
			}
		case <-deadline:
			t.Fatal("等待 logend 事件超时")
		}
	}
done:
	fr.mu.Lock()
	if len(fr.calls) != 1 {
		fr.mu.Unlock()
		t.Fatalf("runner 调用数 = %d", len(fr.calls))
	}
	fr.mu.Unlock()
	if _, err := os.Stat(filepath.Join(res.WorkDir, "output.log")); err != nil {
		t.Fatalf("output.log 应存在: %v", err)
	}
	metaData, err := os.ReadFile(filepath.Join(res.WorkDir, ".metadata.json"))
	if err != nil {
		t.Fatalf(".metadata.json 应存在: %v", err)
	}
	var meta model.Metadata
	if err := json.Unmarshal(metaData, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Tool != "nmap" || meta.Environment != res.EnvUsed || meta.ExitCode != 0 {
		t.Fatalf("metadata 不符: %+v", meta)
	}
}

func TestRunToolUnknownTool(t *testing.T) {
	a, _, _ := newTestApp(t)
	if _, err := a.RunTool(model.RunRequest{Tool: "ghost-tool", Env: "auto"}); err == nil {
		t.Fatal("未知工具应报错")
	}
}

func TestDryRunAndPreference(t *testing.T) {
	a, _, _ := newTestApp(t)
	d, err := a.DryRun("nmap", "", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env == "" {
		t.Fatal("DryRun 应返回决策")
	}
	if err := a.SetPreference("nmap", "podman"); err != nil {
		t.Fatal(err)
	}
	d2, err := a.DryRun("nmap", "", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d2.Env != "podman" {
		t.Fatalf("偏好应生效: %+v", d2)
	}
	if err := a.SetPreference("ghost", "vm"); err == nil {
		t.Fatal("未知工具设偏好应报错")
	}
}

func TestHealthAndWorkspace(t *testing.T) {
	a, _, _ := newTestApp(t)
	a.Aud.Run = func(ctx context.Context, name string, args ...string) (string, int, error) {
		return "", 0, nil
	}
	a.Aud.StatDir = func(string) error { return nil }
	a.Aud.DiskFree = func(string) uint64 { return 10 << 30 }
	checks, err := a.RunHealthCheck()
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 6 {
		t.Fatalf("应 6 项, got %d", len(checks))
	}
	entries, err := a.ListWorkspace("")
	if err != nil {
		t.Fatal(err)
	}
	_ = entries
	if _, err := a.ListWorkspace("../../etc"); err == nil {
		t.Fatal("穿越应报错")
	}
}

func TestDryRunUnknownTool(t *testing.T) {
	a, _, _ := newTestApp(t)
	if _, err := a.DryRun("ghost-tool", "", "auto"); err == nil {
		t.Fatal("未知工具 DryRun 应报错")
	}
}

func TestRunToolInvalidEnv(t *testing.T) {
	a, _, _ := newTestApp(t)
	if _, err := a.RunTool(model.RunRequest{Tool: "nmap", Env: "docker"}); err == nil {
		t.Fatal("非法环境应报错")
	}
}

func TestRunToolNoRunner(t *testing.T) {
	a, _, _ := newTestApp(t)
	a.Exec["podman"] = nil
	a.Exec["vm"] = nil
	if _, err := a.RunTool(model.RunRequest{Tool: "nmap", Env: "local"}); err != nil {
		t.Fatal(err)
	}
	a.Exec["local"] = nil
	a.DB.SetPreference(1, "local")
	if _, err := a.RunTool(model.RunRequest{Tool: "nmap", Env: "local"}); err == nil {
		t.Fatal("无执行器应报错")
	}
}

func TestExecuteLogCreateError(t *testing.T) {
	a, _, events := newTestApp(t)
	dir, _ := a.WS.CreateRunDir("nmap", time.Now())
	os.RemoveAll(dir)
	id, _ := a.DB.StartExecution(1, "local", "-sV", dir)
	a.execute(id, model.Tool{ID: 1, Name: "nmap"}, "local", "-sV", dir, &fakeRunner{})
	select {
	case ev := <-events:
		if ev != EventLogEndPrefix+strconv.FormatInt(id, 10) {
			t.Fatalf("期望 logend 事件, got %s", ev)
		}
	case <-time.After(3 * time.Second):
		t.Fatal("等待 logend 事件超时")
	}
}

func TestSetPreferenceInvalidEnv(t *testing.T) {
	a, _, _ := newTestApp(t)
	if err := a.SetPreference("nmap", "docker"); err == nil {
		t.Fatal("非法环境应报错")
	}
}

func TestOpenWorkspaceFileMissing(t *testing.T) {
	a, _, _ := newTestApp(t)
	if err := a.OpenWorkspaceFile("no-such-file.txt"); err == nil {
		t.Fatal("不存在文件应报错")
	}
	if err := a.OpenWorkspaceFile("nmap"); err == nil {
		t.Fatal("目录应报错")
	}
}

func TestNewErrors(t *testing.T) {
	block := filepath.Join(t.TempDir(), "blocked")
	os.WriteFile(block, []byte("x"), 0o644)
	cfg := &config.Config{}
	cfg.Workspace.Path = filepath.Join(block, "ws")
	if _, err := New(cfg, filepath.Join(block, "app.db")); err == nil {
		t.Fatal("db 打开失败应报错")
	}
	cfg.Workspace.Path = filepath.Join(block, "ws")
	if _, err := New(cfg, filepath.Join(t.TempDir(), "app.db")); err == nil {
		t.Fatal("workspace 创建失败应报错")
	}
}

func TestOpenWorkspaceFile(t *testing.T) {
	a, _, _ := newTestApp(t)
	dir, _ := a.WS.CreateRunDir("nmap", time.Now())
	os.WriteFile(filepath.Join(dir, "x.txt"), []byte("hi"), 0o644)
	var opened []string
	a.OpenFn = func(name string, args ...string) error {
		opened = append(opened, args...)
		return nil
	}
	rel, _ := filepath.Rel(a.WS.Root, filepath.Join(dir, "x.txt"))
	if err := a.OpenWorkspaceFile(rel); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != filepath.Join(a.WS.Root, rel) {
		t.Fatalf("xdg-open 参数错误: %v", opened)
	}
}
