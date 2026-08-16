package executor

import "context"

type Runner interface {
	Run(ctx context.Context, tool, args, workDir string, onLine func(line string)) (int, error)
}
