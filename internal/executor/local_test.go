package executor

import (
	"context"
	"errors"
	"os/exec"
	"strings"
	"sync"
	"testing"
	"time"
)

func TestLocalRunCapturesLines(t *testing.T) {
	l := NewLocal()
	l.LookPath = func(string) (string, error) { return "/fake/nmap", nil }
	l.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCommandExit(0, args...)
	}
	var mu sync.Mutex
	var lines []string
	code, err := l.Run(context.Background(), "nmap", "-sV 192.168.1.1", "/tmp", func(line string) {
		mu.Lock()
		lines = append(lines, line)
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if len(lines) != 3 {
		t.Fatalf("行数 = %d, want 3", len(lines))
	}
	iS, iT := -1, -1
	foundErr := false
	for i, l := range lines {
		switch {
		case l == "-sV":
			iS = i
		case l == "192.168.1.1":
			iT = i
		case strings.Contains(l, "stderr-line"):
			foundErr = true
		}
	}
	if iS == -1 || iT == -1 || iS > iT {
		t.Fatalf("参数拆分错误: %v", lines)
	}
	if !foundErr {
		t.Fatal("应捕获 stderr 行")
	}
}

func TestLocalRunExitCode(t *testing.T) {
	l := NewLocal()
	l.LookPath = func(string) (string, error) { return "/fake/x", nil }
	l.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCommandExit(3, args...)
	}
	code, err := l.Run(context.Background(), "x", "a", "/tmp", nil)
	if err != nil {
		t.Fatalf("退出码非 0 不应返回 error: %v", err)
	}
	if code != 3 {
		t.Fatalf("exit = %d, want 3", code)
	}
}

func TestLocalRunToolNotFound(t *testing.T) {
	l := NewLocal()
	l.LookPath = func(string) (string, error) { return "", errors.New("not found") }
	if _, err := l.Run(context.Background(), "ghost-tool", "", "/tmp", nil); err == nil {
		t.Fatal("工具不存在应报错")
	}
}

func TestLocalRunCancel(t *testing.T) {
	l := NewLocal()
	l.LookPath = func(string) (string, error) { return "/fake/slow", nil }
	l.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCommandContext(ctx, 0, "sleep")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := l.Run(ctx, "slow", "", "/tmp", nil); err == nil {
		t.Fatal("ctx 超时应报错")
	}
}
