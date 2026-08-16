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

func TestVMRunCommandShape(t *testing.T) {
	record := filepath.Join(t.TempDir(), "argv.txt")
	v := NewVM("192.168.122.2", "blackarch", 22)
	v.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmd := helperCommandExit(0, args...)
		cmd.Env = append(cmd.Env, "RECORD_FILE="+record)
		return cmd
	}
	var mu sync.Mutex
	var lines []string
	code, err := v.Run(context.Background(), "nmap", "-sV 1.2.3.4", "/work/dir", func(l string) {
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
	data, _ := os.ReadFile(record)
	joined := string(data)
	for _, want := range []string{"-p", "22", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=no", "blackarch@192.168.122.2"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("argv 缺 %q: %s", want, joined)
		}
	}
	if !strings.Contains(joined, "mkdir -p /work/dir") || !strings.Contains(joined, "cd /work/dir") {
		t.Fatalf("远程命令缺 mkdir/cd: %s", joined)
	}
}

func TestVMSSHFail(t *testing.T) {
	v := NewVM("192.168.122.2", "blackarch", 22)
	v.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCommandExit(255, args...)
	}
	code, err := v.Run(context.Background(), "nmap", "-sV", "/work/dir", nil)
	if err != nil {
		t.Fatal(err)
	}
	if code != 255 {
		t.Fatalf("ssh 免密失败应透传退出码 255, got %d", code)
	}
}
