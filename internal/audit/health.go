package audit

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"syscall"
	"time"

	"blackarch-toolbox/internal/model"
)

type Runner func(ctx context.Context, name string, args ...string) (string, int, error)

type Auditor struct {
	Container     string
	VMName        string
	WorkspacePath string
	MinDiskFree   uint64
	Timeout       time.Duration
	Run           Runner
	StatDir       func(string) error
	DiskFree      func(string) uint64
}

func New(container, vmName, workspacePath string) *Auditor {
	return &Auditor{
		Container:     container,
		VMName:        vmName,
		WorkspacePath: workspacePath,
		MinDiskFree:   2 << 30,
		Timeout:       5 * time.Second,
		Run: func(ctx context.Context, name string, args ...string) (string, int, error) {
			cmd := exec.CommandContext(ctx, name, args...)
			out, err := cmd.CombinedOutput()
			if err != nil {
				if ee, ok := err.(*exec.ExitError); ok {
					return string(out), ee.ProcessState.ExitCode(), nil
				}
				return string(out), -1, err
			}
			return string(out), 0, nil
		},
		StatDir: func(path string) error {
			if err := os.MkdirAll(path, 0o755); err != nil {
				return err
			}
			probe := filepath.Join(path, ".write-probe")
			if err := os.WriteFile(probe, []byte("x"), 0o644); err != nil {
				return err
			}
			return os.Remove(probe)
		},
		DiskFree: func(path string) uint64 {
			var st syscall.Statfs_t
			if err := syscall.Statfs(path, &st); err != nil {
				return 0
			}
			return st.Bavail * uint64(st.Bsize)
		},
	}
}

func (a *Auditor) CheckAll(ctx context.Context) []model.HealthCheck {
	checks := make([]model.HealthCheck, 6)
	var wg sync.WaitGroup
	jobs := []struct {
		name string
		fn   func(ctx context.Context) model.HealthCheck
	}{
		{"Podman 服务", a.checkPodman},
		{"目标容器", a.checkContainer},
		{"libvirtd", a.checkLibvirt},
		{"VM 状态", a.checkVM},
		{"产物目录", a.checkWorkspace},
		{"磁盘空间", a.checkDisk},
	}
	for i, job := range jobs {
		wg.Add(1)
		go func(i int, job struct {
			name string
			fn   func(ctx context.Context) model.HealthCheck
		}) {
			defer wg.Done()
			cctx, cancel := context.WithTimeout(ctx, a.Timeout)
			defer cancel()
			checks[i] = job.fn(cctx)
		}(i, job)
	}
	wg.Wait()
	return checks
}

func (a *Auditor) ok(name, detail string) model.HealthCheck {
	return model.HealthCheck{Name: name, Status: "ok", Detail: detail}
}

func (a *Auditor) warn(name, detail string) model.HealthCheck {
	return model.HealthCheck{Name: name, Status: "warning", Detail: detail}
}

func (a *Auditor) bad(name, detail string) model.HealthCheck {
	return model.HealthCheck{Name: name, Status: "error", Detail: detail}
}

func timedOut(ctx context.Context) bool { return ctx.Err() != nil }

func (a *Auditor) checkPodman(ctx context.Context) model.HealthCheck {
	out, code, err := a.Run(ctx, "podman", "info")
	if timedOut(ctx) {
		return a.bad("Podman 服务", "检查超时")
	}
	if err != nil || code != 0 {
		return a.bad("Podman 服务", fmt.Sprintf("podman info 失败(%d): %s", code, strings.TrimSpace(out)))
	}
	return a.ok("Podman 服务", "rootless podman 正常")
}

func (a *Auditor) checkContainer(ctx context.Context) model.HealthCheck {
	out, code, err := a.Run(ctx, "podman", "ps", "-q", "-f", "name="+a.Container)
	if timedOut(ctx) {
		return a.bad("目标容器", "检查超时")
	}
	if err == nil && code == 0 && strings.TrimSpace(out) != "" {
		return a.ok("目标容器", a.Container+" 运行中")
	}
	_, icode, _ := a.Run(ctx, "podman", "image", "exists", a.Container)
	if icode == 0 {
		return a.warn("目标容器", a.Container+" 存在但未运行")
	}
	return a.bad("目标容器", a.Container+" 不存在")
}

func (a *Auditor) checkLibvirt(ctx context.Context) model.HealthCheck {
	out, code, err := a.Run(ctx, "systemctl", "is-active", "libvirtd")
	if timedOut(ctx) {
		return a.bad("libvirtd", "检查超时")
	}
	if err == nil && code == 0 && strings.TrimSpace(out) == "active" {
		return a.ok("libvirtd", "服务运行中")
	}
	return a.warn("libvirtd", "服务未运行，VM 执行不可用")
}

func (a *Auditor) checkVM(ctx context.Context) model.HealthCheck {
	out, _, _ := a.Run(ctx, "virsh", "list", "--state-running")
	if timedOut(ctx) {
		return a.bad("VM 状态", "检查超时")
	}
	if strings.Contains(out, a.VMName) {
		return a.ok("VM 状态", a.VMName+" 运行中")
	}
	all, _, _ := a.Run(ctx, "virsh", "list", "--all")
	if strings.Contains(all, a.VMName) {
		return a.warn("VM 状态", a.VMName+" 已关机")
	}
	return a.bad("VM 状态", a.VMName+" 不存在")
}

func (a *Auditor) checkWorkspace(ctx context.Context) model.HealthCheck {
	done := make(chan error, 1)
	go func() { done <- a.StatDir(a.WorkspacePath) }()
	select {
	case err := <-done:
		if err != nil {
			return a.bad("产物目录", err.Error())
		}
		return a.ok("产物目录", a.WorkspacePath)
	case <-ctx.Done():
		return a.bad("产物目录", "检查超时")
	}
}

func (a *Auditor) checkDisk(ctx context.Context) model.HealthCheck {
	done := make(chan uint64, 1)
	go func() { done <- a.DiskFree(a.WorkspacePath) }()
	select {
	case free := <-done:
		if free < a.MinDiskFree {
			return a.bad("磁盘空间", fmt.Sprintf("剩余 %.1fGB，低于 2GB 阈值", float64(free)/(1<<30)))
		}
		return a.ok("磁盘空间", fmt.Sprintf("剩余 %.1fGB", float64(free)/(1<<30)))
	case <-ctx.Done():
		return a.bad("磁盘空间", "检查超时")
	}
}
