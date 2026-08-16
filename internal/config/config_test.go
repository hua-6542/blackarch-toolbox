package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaults(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.VM.Host != "192.168.122.2" || cfg.VM.User != "blackarch" || cfg.VM.Port != 22 || cfg.VM.Name != "blackarch" {
		t.Fatalf("vm 默认值不符: %+v", cfg.VM)
	}
	if cfg.Podman.Container != "blackarch-tools" {
		t.Fatalf("podman 默认值不符: %q", cfg.Podman.Container)
	}
	want := filepath.Join(home, "BlackArch_Workspace")
	if got := cfg.WorkspacePath(); got != want {
		t.Fatalf("WorkspacePath = %q, want %q", got, want)
	}
}

func TestLoadFileAndOverride(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "blackarch-toolbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("vm:\n  host: 10.0.0.9\n  user: root\n  port: 2222\n  name: kali\nworkspace:\n  path: ~/MyOut\npodman:\n  container: blackarch\n"), 0o644)
	if err != nil {
		t.Fatal(err)
	}
	cfg, err := Load()
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.VM.Host != "10.0.0.9" || cfg.VM.User != "root" || cfg.VM.Port != 2222 || cfg.VM.Name != "kali" {
		t.Fatalf("vm 文件值不符: %+v", cfg.VM)
	}
	if cfg.Podman.Container != "blackarch" {
		t.Fatalf("podman 文件值不符: %q", cfg.Podman.Container)
	}
	if got := cfg.WorkspacePath(); got != filepath.Join(home, "MyOut") {
		t.Fatalf("WorkspacePath = %q", got)
	}
	t.Setenv("TOOLBOX_VM_HOST", "1.2.3.4")
	t.Setenv("TOOLBOX_PODMAN_CONTAINER", "env-container")
	cfg2, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.VM.Host != "1.2.3.4" {
		t.Fatalf("env 覆盖失败: %+v", cfg2.VM)
	}
	if cfg2.Podman.Container != "env-container" {
		t.Fatalf("env 覆盖失败: %q", cfg2.Podman.Container)
	}
}

func TestBadYAML(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	dir := filepath.Join(home, ".config", "blackarch-toolbox")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "config.yaml"), []byte("vm: [broken"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(); err == nil {
		t.Fatal("坏 YAML 应报错")
	}
}
