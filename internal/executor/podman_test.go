package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

func TestPodmanRunCommandShape(t *testing.T) {
	record := filepath.Join(t.TempDir(), "argv.txt")
	p := NewPodman("blackarch-tools")
	p.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmd := helperCommandExit(0, args...)
		cmd.Env = append(cmd.Env, "RECORD_FILE="+record)
		return cmd
	}
	var mu sync.Mutex
	var lines []string
	code, err := p.Run(context.Background(), "nmap", "-sV 1.2.3.4", "/work/dir", func(l string) {
		mu.Lock()
		lines = append(lines, l)
		mu.Unlock()
	})
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if len(lines) == 0 {
		t.Fatal("应透传日志行")
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	joined := string(data)
	for _, want := range []string{"exec", "-i", "blackarch-tools", "/bin/bash", "-lc"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("argv 缺 %q: %s", want, joined)
		}
	}
	if !strings.Contains(joined, "cd /work/dir") {
		t.Fatalf("argv 缺 cd 命令: %s", joined)
	}
}

func TestPodmanRetryOn125(t *testing.T) {
	record := filepath.Join(t.TempDir(), "argv.txt")
	p := NewPodman("blackarch-tools")
	calls := 0
	p.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		calls++
		if calls == 1 {
			return helperCommandExit(125, args...)
		}
		cmd := helperCommandExit(0, args...)
		cmd.Env = append(cmd.Env, "RECORD_FILE="+record)
		return cmd
	}
	code, err := p.Run(context.Background(), "nmap", "-sV", "/work/dir", nil)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("重试后应成功, exit = %d", code)
	}
	if calls != 3 {
		t.Fatalf("调用次数 = %d, want 3 (exec 失败 + start + exec 重试)", calls)
	}
	data, _ := os.ReadFile(record)
	joined := string(data)
	if !strings.Contains(joined, "exec") {
		t.Fatalf("重试应为 exec: %s", joined)
	}
}

func TestPodmanRunQuotesMetachars(t *testing.T) {
	p := NewPodman("blackarch-tools")
	p.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCommandExit(0, args...)
	}
	if _, err := p.Run(context.Background(), "nmap", "-sV; echo pwned", "/work/dir", nil); err == nil {
		t.Fatal("含 ; 的参数应被 shell.Fields 拒绝，禁止注入")
	}

	record := filepath.Join(t.TempDir(), "argv.txt")
	p.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmd := helperCommandExit(0, args...)
		cmd.Env = append(cmd.Env, "RECORD_FILE="+record)
		return cmd
	}
	code, err := p.Run(context.Background(), "nmap", "-p *", "/work/dir", nil)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	data, _ := os.ReadFile(record)
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	remote := lines[len(lines)-1]
	if strings.Contains(remote, "-p *") {
		t.Fatalf("glob 未引用: %s", remote)
	}
	if !strings.Contains(remote, "'*'") {
		t.Fatalf("glob 应被引用为单 token: %s", remote)
	}
}
