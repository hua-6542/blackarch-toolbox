package executor

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os/exec"

	"mvdan.cc/sh/v3/shell"
)

type Local struct {
	LookPath func(string) (string, error)
	ExecCmd  func(ctx context.Context, name string, args ...string) *exec.Cmd
}

func NewLocal() *Local {
	return &Local{
		LookPath: exec.LookPath,
		ExecCmd:  exec.CommandContext,
	}
}

func (l *Local) Run(ctx context.Context, tool, args, workDir string, onLine func(string)) (int, error) {
	bin, err := l.LookPath(tool)
	if err != nil {
		return -1, fmt.Errorf("本机未找到工具 %s: %w", tool, err)
	}
	fields, err := shell.Fields(args, nil)
	if err != nil {
		return -1, fmt.Errorf("参数解析失败: %w", err)
	}
	cmd := l.ExecCmd(ctx, bin, fields...)
	cmd.Dir = workDir
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
	emit := func(line string) {
		if onLine != nil {
			onLine(line)
		}
	}
	done := make(chan struct{}, 2)
	go scanLines(stdout, emit, done)
	go scanLines(stderr, emit, done)
	<-done
	<-done
	err = cmd.Wait()
	if ctxErr := ctx.Err(); ctxErr != nil {
		return -1, ctxErr
	}
	if err != nil {
		if ee, ok := err.(*exec.ExitError); ok {
			return ee.ProcessState.ExitCode(), nil
		}
		return -1, err
	}
	return 0, nil
}

func scanLines(rd io.Reader, emit func(string), done chan<- struct{}) {
	defer func() { done <- struct{}{} }()
	sc := bufio.NewScanner(rd)
	sc.Buffer(make([]byte, 0, 64*1024), 4*1024*1024)
	for sc.Scan() {
		emit(sc.Text())
	}
}
