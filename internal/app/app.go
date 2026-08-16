package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"sync"
	"time"

	"blackarch-toolbox/internal/audit"
	"blackarch-toolbox/internal/config"
	"blackarch-toolbox/internal/db"
	"blackarch-toolbox/internal/decision"
	"blackarch-toolbox/internal/executor"
	"blackarch-toolbox/internal/model"
	"blackarch-toolbox/internal/workspace"
)

const (
	EventLogPrefix    = "toolbox:log:"
	EventLogEndPrefix = "toolbox:logend:"
)

type App struct {
	Cfg    *config.Config
	DB     *db.DB
	Dec    *decision.Engine
	Exec   map[string]executor.Runner
	WS     *workspace.Manager
	Aud    *audit.Auditor
	OpenFn func(name string, args ...string) error
	emit   func(event string, data ...any)
}

func New(cfg *config.Config, dbPath string) (*App, error) {
	database, err := db.Open(dbPath)
	if err != nil {
		return nil, err
	}
	ws := workspace.New(cfg.WorkspacePath())
	if err := os.MkdirAll(ws.Root, 0o755); err != nil {
		database.Close()
		return nil, err
	}
	a := &App{
		Cfg: cfg,
		DB:  database,
		Dec: decision.New(database),
		Exec: map[string]executor.Runner{
			"local":  executor.NewLocal(),
			"podman": executor.NewPodman(cfg.Podman.Container),
			"vm":     executor.NewVM(cfg.VM.Host, cfg.VM.User, cfg.VM.Port),
		},
		WS:  ws,
		Aud: audit.New(cfg.Podman.Container, cfg.VM.Name, ws.Root),
		OpenFn: func(name string, args ...string) error {
			return exec.Command(name, args...).Start()
		},
	}
	return a, nil
}

func (a *App) SetEventEmitter(fn func(event string, data ...any)) { a.emit = fn }

func (a *App) emitEvent(event string, data ...any) {
	if a.emit != nil {
		a.emit(event, data...)
	}
}

func (a *App) GetTools(category string) ([]model.Tool, error) {
	return a.DB.ListTools(category)
}

func (a *App) DryRun(toolName, args, env string) (model.Decision, error) {
	tool, err := a.DB.GetTool(toolName)
	if err != nil {
		return model.Decision{}, fmt.Errorf("工具不存在: %s", toolName)
	}
	return a.Dec.Decide(*tool, env)
}

func (a *App) RunTool(req model.RunRequest) (model.RunResult, error) {
	tool, err := a.DB.GetTool(req.Tool)
	if err != nil {
		return model.RunResult{}, fmt.Errorf("工具不存在: %s", req.Tool)
	}
	d, err := a.Dec.Decide(*tool, req.Env)
	if err != nil {
		return model.RunResult{}, err
	}
	workDir, err := a.WS.CreateRunDir(tool.Name, time.Now())
	if err != nil {
		return model.RunResult{}, err
	}
	id, err := a.DB.StartExecution(int64(tool.ID), d.Env, req.Args, workDir)
	if err != nil {
		return model.RunResult{}, err
	}
	runner := a.Exec[d.Env]
	if runner == nil {
		return model.RunResult{}, fmt.Errorf("环境 %s 无可用执行器", d.Env)
	}
	go a.execute(id, *tool, d.Env, req.Args, workDir, runner)
	return model.RunResult{ExecutionID: id, EnvUsed: d.Env, WorkDir: workDir}, nil
}

func (a *App) execute(id int64, tool model.Tool, env, args, workDir string, runner executor.Runner) {
	ctx, cancel := context.WithTimeout(context.Background(), 6*time.Hour)
	defer cancel()
	logPath := filepath.Join(workDir, "output.log")
	logFile, err := os.Create(logPath)
	if err != nil {
		a.finish(id, tool, env, args, workDir, -1, err)
		return
	}
	var mu sync.Mutex
	onLine := func(line string) {
		mu.Lock()
		logFile.WriteString(line + "\n")
		mu.Unlock()
		a.emitEvent(EventLogPrefix+strconv.FormatInt(id, 10), line)
	}
	code, runErr := runner.Run(ctx, tool.Name, args, workDir, onLine)
	logFile.Close()
	a.finish(id, tool, env, args, workDir, code, runErr)
}

func (a *App) finish(id int64, tool model.Tool, env, args, workDir string, code int, runErr error) {
	meta := model.Metadata{
		Tool:        tool.Name,
		Environment: env,
		ExecutedAt:  time.Now(),
		Command:     tool.Name + " " + args,
		ExitCode:    code,
		OutputDir:   workDir,
	}
	a.WS.WriteMetadata(workDir, meta)
	a.DB.FinishExecution(id, code)
	a.DB.IncrementUseCount(int64(tool.ID))
	payload := map[string]any{"exit_code": code, "work_dir": workDir, "env": env}
	if runErr != nil {
		payload["error"] = runErr.Error()
	}
	a.emitEvent(EventLogEndPrefix+strconv.FormatInt(id, 10), payload)
}

func (a *App) RunHealthCheck() ([]model.HealthCheck, error) {
	return a.Aud.CheckAll(context.Background()), nil
}

func (a *App) ListWorkspace(path string) ([]model.FileEntry, error) {
	return a.WS.List(path)
}

func (a *App) OpenWorkspaceFile(path string) error {
	abs, err := a.WS.Resolve(path)
	if err != nil {
		return err
	}
	if info, err := os.Stat(abs); err != nil || info.IsDir() {
		return fmt.Errorf("文件不存在: %s", path)
	}
	return a.OpenFn("xdg-open", abs)
}

func (a *App) SetPreference(toolName, env string) error {
	switch env {
	case "local", "podman", "vm":
	default:
		return fmt.Errorf("无效环境: %q", env)
	}
	tool, err := a.DB.GetTool(toolName)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return fmt.Errorf("工具不存在: %s", toolName)
		}
		return err
	}
	return a.DB.SetPreference(int64(tool.ID), env)
}
