package decision

import (
	"fmt"
	"os/exec"

	"blackarch-toolbox/internal/model"
)

var HighRiskTools = []string{
	"metasploit", "volatility", "reaver", "bully",
	"bettercap", "aircrack-ng", "mdk3", "mdk4",
}

type PreferenceStore interface {
	GetPreference(toolID int64) (string, bool, error)
}

type Engine struct {
	Prefs        PreferenceStore
	Which        func(string) (string, bool)
	ConflictDeps map[string]bool
}

func New(prefs PreferenceStore) *Engine {
	return &Engine{
		Prefs: prefs,
		Which: func(bin string) (string, bool) {
			p, err := exec.LookPath(bin)
			return p, err == nil
		},
		ConflictDeps: map[string]bool{"python2": true},
	}
}

func (e *Engine) highRisk(name string) bool {
	for _, v := range HighRiskTools {
		if v == name {
			return true
		}
	}
	return false
}

func (e *Engine) Decide(tool model.Tool, requestedEnv string) (model.Decision, error) {
	if requestedEnv != "" && requestedEnv != "auto" {
		switch requestedEnv {
		case "local", "podman", "vm":
			return model.Decision{Env: requestedEnv, Reason: "用户指定环境", Priority: 1}, nil
		default:
			return model.Decision{}, fmt.Errorf("无效环境: %q", requestedEnv)
		}
	}
	if env, ok, err := e.Prefs.GetPreference(int64(tool.ID)); err != nil {
		return model.Decision{}, err
	} else if ok {
		return model.Decision{Env: env, Reason: "用户偏好设置", Priority: 1}, nil
	}
	if e.highRisk(tool.Name) || tool.IsHighRisk {
		return model.Decision{Env: "vm", Reason: "高危工具，强制 VM 隔离", Priority: 2}, nil
	}
	for _, dep := range tool.Dependencies {
		if e.ConflictDeps[dep] {
			return model.Decision{Env: "podman", Reason: fmt.Sprintf("依赖 %s 与本机冲突，使用容器", dep), Priority: 3}, nil
		}
	}
	if _, ok := e.Which(tool.Name); ok {
		return model.Decision{Env: "local", Reason: fmt.Sprintf("本机存在 %s", tool.Name), Priority: 4}, nil
	}
	return model.Decision{Env: "vm", Reason: "兜底：VM 最安全", Priority: 5}, nil
}
