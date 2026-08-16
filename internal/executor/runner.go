package executor

import (
	"context"
	"fmt"

	"mvdan.cc/sh/v3/syntax"
)

type Runner interface {
	Run(ctx context.Context, tool, args, workDir string, onLine func(line string)) (int, error)
}

func quoteCommand(tool string, tokens []string) ([]string, error) {
	quoted := make([]string, 0, len(tokens)+1)
	qt, err := syntax.Quote(tool, syntax.LangBash)
	if err != nil {
		return nil, fmt.Errorf("工具名无法引用: %w", err)
	}
	quoted = append(quoted, qt)
	for _, tok := range tokens {
		q, err := syntax.Quote(tok, syntax.LangBash)
		if err != nil {
			return nil, fmt.Errorf("参数引用失败: %w", err)
		}
		quoted = append(quoted, q)
	}
	return quoted, nil
}
