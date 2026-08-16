package executor

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestMain(m *testing.M) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") == "1" {
		helperProcess()
		return
	}
	os.Exit(m.Run())
}

func helperProcess() {
	args := os.Args
	for len(args) > 0 {
		if args[0] == "--" {
			args = args[1:]
			break
		}
		args = args[1:]
	}
	if len(args) == 0 {
		os.Exit(2)
	}
	if f := os.Getenv("RECORD_FILE"); f != "" {
		os.WriteFile(f, []byte(strings.Join(args, "\n")+"\n"), 0o644)
	}
	if args[0] == "sleep" {
		time.Sleep(30 * time.Second)
	}
	for _, line := range args {
		fmt.Fprintln(os.Stdout, line)
	}
	fmt.Fprintln(os.Stderr, "stderr-line")
	code, _ := strconv.Atoi(os.Getenv("EXIT_CODE"))
	os.Exit(code)
}

func helperCommand(args ...string) *exec.Cmd {
	cmd := exec.Command(os.Args[0], append([]string{"-test.run=TestMain", "--"}, args...)...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
	return cmd
}

func helperCommandExit(code int, args ...string) *exec.Cmd {
	cmd := helperCommand(args...)
	cmd.Env = append(cmd.Env, "EXIT_CODE="+strconv.Itoa(code))
	return cmd
}

func helperCommandContext(ctx context.Context, code int, args ...string) *exec.Cmd {
	cmd := exec.CommandContext(ctx, os.Args[0], append([]string{"-test.run=TestMain", "--"}, args...)...)
	cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1", "EXIT_CODE="+strconv.Itoa(code))
	return cmd
}
