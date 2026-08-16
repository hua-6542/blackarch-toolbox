package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
	"sync"

	"mvdan.cc/sh/v3/shell"
	"mvdan.cc/sh/v3/syntax"
)

type Podman struct {
	Container string
	ExecCmd   func(ctx context.Context, name string, args ...string) *exec.Cmd
}

func NewPodman(container string) *Podman {
	return &Podman{Container: container, ExecCmd: exec.CommandContext}
}

func (p *Podman) Run(ctx context.Context, tool, args, workDir string, onLine func(string)) (int, error) {
	wd, err := syntax.Quote(workDir, syntax.LangBash)
	if err != nil {
		return -1, fmt.Errorf("workDir 无法引用: %w", err)
	}
	tokens, err := shell.Fields(args, nil)
	if err != nil {
		return -1, fmt.Errorf("参数解析失败: %w", err)
	}
	quoted, err := quoteCommand(tool, tokens)
	if err != nil {
		return -1, err
	}
	cmdLine := fmt.Sprintf("cd %s && %s", wd, strings.Join(quoted, " "))
	full := []string{"exec", "-i", p.Container, "/bin/bash", "-lc", cmdLine}
	code, stderrText, err := p.runOnce(ctx, full, onLine)
	if err == nil && code == 125 && strings.Contains(stderrText, "not running") {
		start := []string{"start", p.Container}
		if _, _, serr := p.runOnce(ctx, start, onLine); serr != nil {
			return -1, fmt.Errorf("podman start 失败: %w", serr)
		}
		code, _, err = p.runOnce(ctx, full, onLine)
	}
	return code, err
}

func (p *Podman) runOnce(ctx context.Context, args []string, onLine func(string)) (int, string, error) {
	cmd := p.ExecCmd(ctx, "podman", args...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return -1, "", err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return -1, "", err
	}
	if err := cmd.Start(); err != nil {
		return -1, "", err
	}
	done := make(chan struct{}, 2)
	emit := func(line string) {
		if onLine != nil {
			onLine(line)
		}
	}
	var stderrMu sync.Mutex
	var stderrBuf strings.Builder
	go scanLines(stdout, emit, done)
	go scanLines(stderr, func(line string) {
		stderrMu.Lock()
		stderrBuf.WriteString(line + "\n")
		stderrMu.Unlock()
		emit(line)
	}, done)
	<-done
	<-done
	if ctxErr := ctx.Err(); ctxErr != nil {
		return -1, "", ctxErr
	}
	err = cmd.Wait()
	stderrMu.Lock()
	stderrText := stderrBuf.String()
	stderrMu.Unlock()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ProcessState.ExitCode(), stderrText, nil
		}
		return -1, stderrText, err
	}
	return 0, stderrText, nil
}
