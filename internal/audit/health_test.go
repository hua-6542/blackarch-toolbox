package audit

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"

	"blackarch-toolbox/internal/model"
)

type fakeRun struct {
	mu      sync.Mutex
	results map[string]fakeResult
	calls   []string
}

type fakeResult struct {
	out  string
	code int
	err  error
}

func (f *fakeRun) run(ctx context.Context, name string, args ...string) (string, int, error) {
	key := name + " " + strings.Join(args, " ")
	f.mu.Lock()
	f.calls = append(f.calls, key)
	f.mu.Unlock()
	r, ok := f.results[key]
	if !ok {
		return "", 127, nil
	}
	return r.out, r.code, r.err
}

func newAuditor(f *fakeRun) *Auditor {
	a := New("blackarch-tools", "blackarch", "/tmp/ws")
	a.Run = f.run
	a.Timeout = time.Second
	a.StatDir = func(string) error { return nil }
	a.DiskFree = func(string) uint64 { return 10 << 30 }
	return a
}

func TestAllOK(t *testing.T) {
	f := &fakeRun{results: map[string]fakeResult{
		"podman info":                          {out: "ok", code: 0},
		"podman ps -q -f name=blackarch-tools": {out: "abc123", code: 0},
		"systemctl is-active libvirtd":         {out: "active", code: 0},
		"virsh list --state-running":           {out: " Id Name\n 1  blackarch\n", code: 0},
	}}
	checks := newAuditor(f).CheckAll(context.Background())
	if len(checks) != 6 {
		t.Fatalf("应返回 6 项, got %d", len(checks))
	}
	for _, c := range checks {
		if c.Status != "ok" {
			t.Fatalf("%s: status = %s, want ok (%s)", c.Name, c.Status, c.Detail)
		}
	}
}

func TestContainerNotRunning(t *testing.T) {
	f := &fakeRun{results: map[string]fakeResult{
		"podman info":                          {code: 0},
		"podman ps -q -f name=blackarch-tools": {code: 0, out: ""},
		"podman image exists blackarch-tools":  {code: 0},
		"systemctl is-active libvirtd":         {out: "active", code: 0},
		"virsh list --state-running":           {out: " Id Name\n", code: 0},
		"virsh list --all":                     {out: " -  blackarch 关闭\n", code: 0},
	}}
	checks := newAuditor(f).CheckAll(context.Background())
	if statusOf(checks, "目标容器") != "warning" {
		t.Fatalf("容器未运行应 warning: %+v", checks)
	}
	if statusOf(checks, "VM 状态") != "warning" {
		t.Fatalf("VM 关机应 warning: %+v", checks)
	}
}

func TestPodmanDownAndVMGone(t *testing.T) {
	f := &fakeRun{results: map[string]fakeResult{
		"podman info":                          {code: 1, out: "cannot connect"},
		"podman ps -q -f name=blackarch-tools": {code: 1},
		"podman image exists blackarch-tools":  {code: 1},
		"systemctl is-active libvirtd":         {out: "inactive", code: 3},
		"virsh list --state-running":           {code: 0, out: " Id Name\n"},
		"virsh list --all":                     {code: 0, out: " Id Name\n"},
	}}
	checks := newAuditor(f).CheckAll(context.Background())
	if statusOf(checks, "Podman 服务") != "error" {
		t.Fatalf("podman 服务应 error: %+v", checks)
	}
	if statusOf(checks, "VM 状态") != "error" {
		t.Fatalf("VM 不存在应 error: %+v", checks)
	}
}

func TestWorkspaceAndDisk(t *testing.T) {
	f := &fakeRun{}
	a := newAuditor(f)
	a.StatDir = func(string) error { return errors.New("no dir") }
	a.DiskFree = func(string) uint64 { return 1 << 30 }
	checks := a.CheckAll(context.Background())
	if statusOf(checks, "产物目录") != "error" {
		t.Fatalf("产物目录应 error: %+v", checks)
	}
	if statusOf(checks, "磁盘空间") != "error" {
		t.Fatalf("磁盘空间应 error: %+v", checks)
	}
}

func statusOf(checks []model.HealthCheck, name string) string {
	for _, c := range checks {
		if c.Name == name {
			return c.Status
		}
	}
	return ""
}
