package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"mvdan.cc/sh/v3/shell"
	"mvdan.cc/sh/v3/syntax"
)

type VM struct {
	Host    string
	User    string
	Port    int
	ExecCmd func(ctx context.Context, name string, args ...string) *exec.Cmd
}

func NewVM(host, user string, port int) *VM {
	return &VM{Host: host, User: user, Port: port, ExecCmd: exec.CommandContext}
}

func (v *VM) Run(ctx context.Context, tool, args, workDir string, onLine func(string)) (int, error) {
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
	remote := fmt.Sprintf("mkdir -p %s && cd %s && %s", wd, wd, strings.Join(quoted, " "))
	full := []string{
		"-p", strconv.Itoa(v.Port),
		"-o", "BatchMode=yes",
		"-o", "StrictHostKeyChecking=no",
		fmt.Sprintf("%s@%s", v.User, v.Host),
		remote,
	}
	cmd := v.ExecCmd(ctx, "ssh", full...)
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return -1, err
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return -1, err
	}
	if err := cmd.Start(); err != nil {
		return -1, err
	}
	done := make(chan struct{}, 2)
	emit := func(line string) {
		if onLine != nil {
			onLine(line)
		}
	}
	go scanLines(stdout, emit, done)
	go scanLines(stderr, emit, done)
	<-done
	<-done
	if ctxErr := ctx.Err(); ctxErr != nil {
		return -1, ctxErr
	}
	err = cmd.Wait()
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ProcessState.ExitCode(), nil
		}
		return -1, fmt.Errorf("ssh 执行失败（免密未配置或主机不可达）: %w", err)
	}
	return 0, nil
}
