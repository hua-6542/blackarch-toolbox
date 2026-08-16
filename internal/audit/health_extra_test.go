package audit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestAllChecksTimeout(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	checks := newAuditor(&fakeRun{}).CheckAll(ctx)
	if len(checks) != 6 {
		t.Fatalf("应返回 6 项, got %d", len(checks))
	}
	for _, c := range checks {
		if c.Status != "error" || !strings.Contains(c.Detail, "超时") {
			t.Fatalf("%s: status=%s detail=%s, 应超时", c.Name, c.Status, c.Detail)
		}
	}
}

func TestNewDefaults(t *testing.T) {
	tmp := t.TempDir()
	success := filepath.Join(tmp, "okcmd")
	failure := filepath.Join(tmp, "failcmd")
	if err := os.WriteFile(success, []byte("#!/bin/sh\necho fake-out\nexit 0\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(failure, []byte("#!/bin/sh\necho fake-err\nexit 1\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp)

	a := New("c", "vm", tmp)

	out, code, err := a.Run(context.Background(), "okcmd")
	if err != nil || code != 0 || !strings.Contains(out, "fake-out") {
		t.Fatalf("Run 成功路径: out=%q code=%d err=%v", out, code, err)
	}

	out, code, err = a.Run(context.Background(), "failcmd")
	if err != nil || code != 1 || !strings.Contains(out, "fake-err") {
		t.Fatalf("Run ExitError 路径: out=%q code=%d err=%v", out, code, err)
	}

	_, code, err = a.Run(context.Background(), "definitely-not-a-real-binary")
	if err == nil || code != -1 {
		t.Fatalf("Run 非 ExitError 路径: code=%d err=%v", code, err)
	}

	ws := filepath.Join(tmp, "nested", "workspace")
	if err := a.StatDir(ws); err != nil {
		t.Fatalf("StatDir 默认实现失败: %v", err)
	}
	if _, err := os.Stat(filepath.Join(ws, ".write-probe")); !os.IsNotExist(err) {
		t.Fatalf("探测文件应被清理, stat err=%v", err)
	}
	if free := a.DiskFree(ws); free == 0 {
		t.Fatal("DiskFree 默认实现返回 0（Statfs 失败）")
	}
}
