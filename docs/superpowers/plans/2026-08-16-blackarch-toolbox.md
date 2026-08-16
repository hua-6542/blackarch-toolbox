# BlackArch ToolBox 实现计划

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 构建 BlackArch ToolBox——管理 BlackArch 工具的 Arch 桌面应用（分类浏览、智能路由运行、健康审计、产物归档），最终产出单文件 `build/bin/blackarch-toolbox`。

**Architecture:** Wails v2 绑定 Go 后端（Bind 函数 + Events 日志流），前端 Vue 3 + Vite 深色中文界面。后端按 decision/executor/workspace/audit/db/config 分模块，SQLite（modernc.org/sqlite，无 CGO）持久化。

**Tech Stack:** Go 1.26、Wails v2.14.0、Vue 3 + Vite（JS 模板）、modernc.org/sqlite、gopkg.in/yaml.v3、mvdan.cc/sh/v3。

**Spec:** `docs/superpowers/specs/2026-08-16-blackarch-toolbox-design.md`

## Global Constraints

- Wails **v2.14.0**（稳定版）；本机 webkit2gtk-4.1 已装
- 无 CGO：SQLite 驱动 `modernc.org/sqlite`，驱动名 `"sqlite"`
- Rootless Podman：所有命令**不加 sudo**
- SSH 参数固定：`-o BatchMode=yes -o StrictHostKeyChecking=no`（免密失败直接报错）
- 单文件产物：`build/bin/blackarch-toolbox`
- 界面语言：**中文**；深色主题
- 默认值：vm host `192.168.122.2` / user `blackarch` / port `22` / name `blackarch`；podman container `blackarch-tools`；workspace `~/BlackArch_Workspace`（开发机 EDITH 实际容器名为 `blackarch`，用 config 覆盖验证）
- TDD：每个 Go 包先写失败测试再实现，`go test ./internal/... -cover` 每包覆盖率 >80%
- 模块名：`blackarch-toolbox`；Go 包结构 `internal/{model,config,db,decision,executor,workspace,audit,app}`
- 参数解析用 `mvdan.cc/sh/v3/shell.Fields/Quote`，禁止拼接 shell 字符串
- 每次任务结束：`git add` 相关文件并 commit（作者 karen <karen@edith.local>，用 `git -c user.name=karen -c user.email=karen@edith.local commit`）

## 文件结构

```
blackarch-toolbox/
├── main.go                          # Wails 入口（薄壳）
├── go.mod / go.sum / wails.json
├── tools.json → 位于 internal/db/tools.json（go:embed 内嵌，单文件自包含）
├── README.md
├── scripts/import_tools.sh          # 从真机 /usr/share/blackarch 同步
├── docs/superpowers/…               # spec + 本计划
├── frontend/
│   ├── package.json / vite.config.js / index.html
│   └── src/
│       ├── main.js / style.css
│       ├── App.vue
│       └── components/{CategoryTree,ToolCard,RunDialog,LogViewer,AuditPanel,StatusBar}.vue
└── internal/
    ├── model/types.go
    ├── config/config.go
    ├── db/sqlite.go + tools.json（seed，go:embed）
    ├── decision/engine.go
    ├── executor/runner.go + local.go + podman.go + vm.go + helper_test.go
    ├── workspace/manager.go
    ├── audit/health.go
    └── app/app.go                    # Bind 方法全部在此
```

与 spec 的两处实现级偏差（均已在计划内说明）：tools.json 移到 `internal/db/`（go:embed 要求同包目录，满足单文件约束）；tools 表增加 `dependencies TEXT` 列（Tool 结构体含该字段，spec SQL 未含）。

## Task 1: Wails 项目脚手架

**Files:**
- Create: 项目根全部模板文件（wails init 生成）、`wails.json`
- Modify: 无

**Interfaces:**
- Consumes: 无
- Produces: 可构建的空 Wails 项目（模块名 `blackarch-toolbox`，vue 模板），`build/bin/blackarch-toolbox` 基线产物

- [ ] **Step 1: 安装 Wails CLI v2.14.0**

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0
ls ~/go/bin/wails
```

预期：`~/go/bin/wails` 存在。若 PATH 未含 `~/go/bin`，后续命令用绝对路径。

- [ ] **Step 2: 校验环境**

```bash
~/go/bin/wails doctor
```

预期：输出无红色错误（Go、Node、webkit2gtk、gtk3 均 OK；`wails` 命令路径提示可忽略）。

- [ ] **Step 3: 在临时目录生成 vue 模板**

```bash
mkdir -p /tmp/opencode/scaffold
~/go/bin/wails init -n blackarch-toolbox -t vue -d /tmp/opencode/scaffold/blackarch-toolbox
```

预期：`/tmp/opencode/scaffold/blackarch-toolbox/` 下出现 `main.go`、`app.go`、`wails.json`、`go.mod`、`frontend/`。

- [ ] **Step 4: 合并到项目根（保留 .git 与 docs/）**

```bash
cp -r /tmp/opencode/scaffold/blackarch-toolbox/. /home/karen/blackarch-toolbox/
ls /home/karen/blackarch-toolbox/  # 应见 main.go app.go go.mod wails.json frontend build docs .git
```

- [ ] **Step 5: 校准 wails.json**

将 `wails.json` 内容替换为：

```json
{
  "$schema": "https://wails.io/schemas/config.v2.json",
  "name": "blackarch-toolbox",
  "outputfilename": "blackarch-toolbox",
  "frontend:install": "npm install",
  "frontend:build": "npm run build",
  "frontend:dev:watcher": "npm run dev",
  "frontend:dev:serverUrl": "auto",
  "author": {
    "name": "karen",
    "email": "karen@edith.local"
  }
}
```

- [ ] **Step 6: 基线构建验证**

```bash
cd /home/karen/blackarch-toolbox && ~/go/bin/wails build
ls -la build/bin/blackarch-toolbox
```

预期：构建成功（首次会下载全部依赖，可能需数分钟），产物存在。若报缺 `libgtk-3-dev` 之类错误，`sudo pacman -S --needed webkit2gtk-4.1 gtk3` 后重试。

- [ ] **Step 7: Commit**

```bash
cd /home/karen/blackarch-toolbox
git add -A
git -c user.name=karen -c user.email=karen@edith.local commit -m "chore: wails v2.14.0 vue 模板脚手架，基线构建通过"
```

---

## Task 2: model 类型 + config 配置加载

**Files:**
- Create: `internal/model/types.go`、`internal/config/config.go`
- Test: `internal/config/config_test.go`

**Interfaces:**
- Consumes: 无
- Produces:
  - `model.Tool{ID int; Name, Category, Description, DefaultEnv string; IsHighRisk bool; Dependencies []string; Icon string; UseCount int}`（JSON tag：`id,name,category,description,default_env,is_high_risk,dependencies,icon,use_count`）
  - `model.RunRequest{Tool, Args, Env string}`、`model.RunResult{ExecutionID int64; EnvUsed, WorkDir string}`、`model.Decision{Env, Reason string; Priority int}`、`model.HealthCheck{Name, Status, Detail string}`、`model.FileEntry{Name, Path string; IsDir bool; Size int64}`、`model.Execution{ID, ToolID int64; ToolName, EnvUsed, Args, OutputPath string; StartTime, EndTime time.Time; ExitCode int}`、`model.Metadata{Tool, Environment, Command, OutputDir string; ExecutedAt time.Time; ExitCode int}`
  - `config.Load() (*config.Config, error)`；`config.Config{VM{Host,User,Name string; Port int}; Workspace{Path string}; Podman{Container string}}`；`(c *Config) WorkspacePath() string`（展开 `~`）

- [ ] **Step 1: 写失败测试 `internal/config/config_test.go`**

```go
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
```

- [ ] **Step 2: 运行测试确认失败**

```bash
cd /home/karen/blackarch-toolbox && go test ./internal/config/ -v
```

预期：FAIL（`undefined: config.Load` 等编译错误）。

- [ ] **Step 3: 实现 `internal/model/types.go`**

```go
package model

import "time"

type Tool struct {
	ID           int      `json:"id"`
	Name         string   `json:"name"`
	Category     string   `json:"category"`
	Description  string   `json:"description"`
	DefaultEnv   string   `json:"default_env"`
	IsHighRisk   bool     `json:"is_high_risk"`
	Dependencies []string `json:"dependencies,omitempty"`
	Icon         string   `json:"icon"`
	UseCount     int      `json:"use_count"`
}

type RunRequest struct {
	Tool string `json:"tool"`
	Args string `json:"args"`
	Env  string `json:"env"`
}

type RunResult struct {
	ExecutionID int64  `json:"execution_id"`
	EnvUsed     string `json:"env_used"`
	WorkDir     string `json:"work_dir"`
}

type Decision struct {
	Env      string `json:"env"`
	Reason   string `json:"reason"`
	Priority int    `json:"priority"`
}

type HealthCheck struct {
	Name   string `json:"name"`
	Status string `json:"status"`
	Detail string `json:"detail"`
}

type FileEntry struct {
	Name  string `json:"name"`
	Path  string `json:"path"`
	IsDir bool   `json:"is_dir"`
	Size  int64  `json:"size"`
}

type Execution struct {
	ID         int64     `json:"id"`
	ToolID     int64     `json:"tool_id"`
	ToolName   string    `json:"tool_name"`
	EnvUsed    string    `json:"env_used"`
	Args       string    `json:"args"`
	StartTime  time.Time `json:"start_time"`
	EndTime    time.Time `json:"end_time"`
	ExitCode   int       `json:"exit_code"`
	OutputPath string    `json:"output_path"`
}

type Metadata struct {
	Tool        string    `json:"tool"`
	Environment string    `json:"environment"`
	ExecutedAt  time.Time `json:"executed_at"`
	Command     string    `json:"command"`
	ExitCode    int       `json:"exit_code"`
	OutputDir   string    `json:"output_dir"`
}
```

- [ ] **Step 4: 实现 `internal/config/config.go`**

```go
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"gopkg.in/yaml.v3"
)

type Config struct {
	VM struct {
		Host string `yaml:"host"`
		User string `yaml:"user"`
		Port int    `yaml:"port"`
		Name string `yaml:"name"`
	} `yaml:"vm"`
	Workspace struct {
		Path string `yaml:"path"`
	} `yaml:"workspace"`
	Podman struct {
		Container string `yaml:"container"`
	} `yaml:"podman"`
}

func defaults() *Config {
	c := &Config{}
	c.VM.Host = "192.168.122.2"
	c.VM.User = "blackarch"
	c.VM.Port = 22
	c.VM.Name = "blackarch"
	c.Workspace.Path = "~/BlackArch_Workspace"
	c.Podman.Container = "blackarch-tools"
	return c
}

func Load() (*Config, error) {
	c := defaults()
	home, _ := os.UserHomeDir()
	path := filepath.Join(home, ".config", "blackarch-toolbox", "config.yaml")
	if data, err := os.ReadFile(path); err == nil {
		if err := yaml.Unmarshal(data, c); err != nil {
			return nil, fmt.Errorf("解析 %s 失败: %w", path, err)
		}
	}
	applyEnv(c)
	return c, nil
}

func applyEnv(c *Config) {
	if v := os.Getenv("TOOLBOX_VM_HOST"); v != "" {
		c.VM.Host = v
	}
	if v := os.Getenv("TOOLBOX_VM_USER"); v != "" {
		c.VM.User = v
	}
	if v := os.Getenv("TOOLBOX_VM_PORT"); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			c.VM.Port = n
		}
	}
	if v := os.Getenv("TOOLBOX_VM_NAME"); v != "" {
		c.VM.Name = v
	}
	if v := os.Getenv("TOOLBOX_WORKSPACE"); v != "" {
		c.Workspace.Path = v
	}
	if v := os.Getenv("TOOLBOX_PODMAN_CONTAINER"); v != "" {
		c.Podman.Container = v
	}
}

func (c *Config) WorkspacePath() string {
	p := c.Workspace.Path
	if p == "~" || len(p) > 1 && p[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			p = filepath.Join(home, p[2:])
		}
	}
	return p
}
```

- [ ] **Step 5: 运行测试确认通过**

```bash
go get gopkg.in/yaml.v3 && go test ./internal/config/ -v -cover
```

预期：PASS，覆盖率 >80%。

- [ ] **Step 6: Commit**

```bash
git add go.mod go.sum internal/
git -c user.name=karen -c user.email=karen@edith.local commit -m "feat: model 类型与 config 配置加载（含单测）"
```

---

## Task 3: SQLite 数据层（含 seed 导入）

**Files:**
- Create: `internal/db/sqlite.go`、`internal/db/tools.json`（本任务先放 3 条测试种子，Task 4 换完整数据）
- Test: `internal/db/sqlite_test.go`

**Interfaces:**
- Consumes: `model.Tool`
- Produces:
  - `db.Open(path string) (*DB, error)`：建父目录、打开、建表、tools 表空时用 `//go:embed tools.json` 种子导入
  - `(*DB) Close() error`
  - `(*DB) ListTools(category string) ([]model.Tool, error)`（category 空=全部，按 use_count DESC, name 排序）
  - `(*DB) GetTool(name string) (*model.Tool, error)`（不存在返回 `sql.ErrNoRows`）
  - `(*DB) GetPreference(toolID int64) (string, bool, error)`
  - `(*DB) SetPreference(toolID int64, env string) error`（upsert）
  - `(*DB) StartExecution(toolID int64, envUsed, args, outputPath string) (int64, error)`
  - `(*DB) FinishExecution(id int64, exitCode int) error`
  - `(*DB) IncrementUseCount(toolID int64) error`
  - `(*DB) RecentExecutions(limit int) ([]model.Execution, error)`（JOIN tools 取名字）

- [ ] **Step 1: 写失败测试 `internal/db/sqlite_test.go`**

```go
package db

import (
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"blackarch-toolbox/internal/model"
)

func newTestDB(t *testing.T) *DB {
	t.Helper()
	d, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatalf("Open: %v", err)
	}
	t.Cleanup(func() { d.Close() })
	return d
}

func TestOpenMigratesAndSeeds(t *testing.T) {
	d := newTestDB(t)
	var n int
	if err := d.conn.QueryRow("SELECT COUNT(*) FROM tools").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 3 {
		t.Fatalf("seed 后 tools 行数 = %d, want 3", n)
	}
	for _, table := range []string{"tools", "executions", "preferences"} {
		var name string
		err := d.conn.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
		if err != nil {
			t.Fatalf("表 %s 不存在: %v", table, err)
		}
	}
}

func TestSeedIdempotent(t *testing.T) {
	d := newTestDB(t)
	d.Seed()
	var n int
	d.conn.QueryRow("SELECT COUNT(*) FROM tools").Scan(&n)
	if n != 3 {
		t.Fatalf("重复 seed 后 = %d, want 3", n)
	}
}

func TestListToolsByCategory(t *testing.T) {
	d := newTestDB(t)
	all, err := d.ListTools("")
	if err != nil {
		t.Fatal(err)
	}
	if len(all) != 3 {
		t.Fatalf("ListTools(\"\") = %d, want 3", len(all))
	}
	scan, err := d.ListTools("scanner")
	if err != nil {
		t.Fatal(err)
	}
	if len(scan) != 1 || scan[0].Name != "nmap" {
		t.Fatalf("ListTools(scanner) = %+v", scan)
	}
}

func TestGetTool(t *testing.T) {
	d := newTestDB(t)
	tool, err := d.GetTool("nmap")
	if err != nil {
		t.Fatal(err)
	}
	if tool.Name != "nmap" || tool.Category != "scanner" {
		t.Fatalf("GetTool = %+v", tool)
	}
	if _, err := d.GetTool("不存在"); !errors.Is(err, sql.ErrNoRows) {
		t.Fatalf("不存在工具应返回 sql.ErrNoRows, got %v", err)
	}
}

func TestPreferences(t *testing.T) {
	d := newTestDB(t)
	tool, _ := d.GetTool("nmap")
	if _, ok, err := d.GetPreference(int64(tool.ID)); err != nil || ok {
		t.Fatalf("初始偏好应为空, ok=%v err=%v", ok, err)
	}
	if err := d.SetPreference(int64(tool.ID), "podman"); err != nil {
		t.Fatal(err)
	}
	env, ok, err := d.GetPreference(int64(tool.ID))
	if err != nil || !ok || env != "podman" {
		t.Fatalf("偏好读取失败: %q %v %v", env, ok, err)
	}
	if err := d.SetPreference(int64(tool.ID), "vm"); err != nil {
		t.Fatal(err)
	}
	if env, _, _ := d.GetPreference(int64(tool.ID)); env != "vm" {
		t.Fatalf("偏好 upsert 失败: %q", env)
	}
}

func TestExecutions(t *testing.T) {
	d := newTestDB(t)
	tool, _ := d.GetTool("nmap")
	id, err := d.StartExecution(int64(tool.ID), "local", "-sV x", "/tmp/out")
	if err != nil {
		t.Fatal(err)
	}
	if err := d.FinishExecution(id, 0); err != nil {
		t.Fatal(err)
	}
	if err := d.IncrementUseCount(int64(tool.ID)); err != nil {
		t.Fatal(err)
	}
	list, err := d.RecentExecutions(10)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 || list[0].ToolName != "nmap" || list[0].EnvUsed != "local" || list[0].ExitCode != 0 {
		t.Fatalf("RecentExecutions = %+v", list)
	}
	updated, _ := d.GetTool("nmap")
	if updated.UseCount != 1 {
		t.Fatalf("use_count = %d, want 1", updated.UseCount)
	}
}

func TestGetPreferenceNotFound(t *testing.T) {
	d := newTestDB(t)
	if _, ok, err := d.GetPreference(999); err != nil || ok {
		t.Fatalf("不存在的 tool_id: ok=%v err=%v", ok, err)
	}
}
```

- [ ] **Step 2: 创建临时种子 `internal/db/tools.json`（3 条，Task 4 会替换）**

```json
[
  {"name": "nmap", "category": "scanner", "description": "网络扫描与主机发现", "default_env": "local", "is_high_risk": false, "dependencies": [], "icon": "🎯"},
  {"name": "metasploit", "category": "exploitation", "description": "漏洞利用框架", "default_env": "vm", "is_high_risk": true, "dependencies": [], "icon": "💀"},
  {"name": "beef-xss", "category": "exploitation", "description": "浏览器漏洞利用框架", "default_env": "podman", "is_high_risk": false, "dependencies": ["python2"], "icon": "🐮"}
]
```

- [ ] **Step 3: 运行测试确认失败**

```bash
go test ./internal/db/ -v
```

预期：FAIL（package db 不存在）。

- [ ] **Step 4: 实现 `internal/db/sqlite.go`**

```go
package db

import (
	"database/sql"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"

	"blackarch-toolbox/internal/model"
)

//go:embed tools.json
var seedJSON []byte

type DB struct {
	conn *sql.DB
}

func Open(path string) (*DB, error) {
	if dir := filepath.Dir(path); dir != "." {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("创建数据目录: %w", err)
		}
	}
	conn, err := sql.Open("sqlite", path)
	if err != nil {
		return nil, err
	}
	if err := conn.Ping(); err != nil {
		return nil, err
	}
	d := &DB{conn: conn}
	if err := d.migrate(); err != nil {
		conn.Close()
		return nil, err
	}
	var n int
	if err := conn.QueryRow("SELECT COUNT(*) FROM tools").Scan(&n); err != nil {
		conn.Close()
		return nil, err
	}
	if n == 0 {
		if _, err := d.Seed(); err != nil {
			conn.Close()
			return nil, fmt.Errorf("种子导入: %w", err)
		}
	}
	return d, nil
}

func (d *DB) Close() error { return d.conn.Close() }

func (d *DB) migrate() error {
	schema := `
CREATE TABLE IF NOT EXISTS tools (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    category TEXT,
    description TEXT,
    default_env TEXT DEFAULT 'local',
    is_high_risk INTEGER DEFAULT 0,
    use_count INTEGER DEFAULT 0,
    icon TEXT,
    dependencies TEXT NOT NULL DEFAULT '[]'
);
CREATE TABLE IF NOT EXISTS executions (
    id INTEGER PRIMARY KEY,
    tool_id INTEGER,
    env_used TEXT,
    args TEXT,
    start_time DATETIME,
    end_time DATETIME,
    exit_code INTEGER,
    output_path TEXT,
    FOREIGN KEY(tool_id) REFERENCES tools(id)
);
CREATE TABLE IF NOT EXISTS preferences (
    tool_id INTEGER PRIMARY KEY,
    preferred_env TEXT,
    updated_at DATETIME,
    FOREIGN KEY(tool_id) REFERENCES tools(id)
);`
	_, err := d.conn.Exec(schema)
	return err
}

func (d *DB) Seed() (int, error) {
	var tools []model.Tool
	if err := json.Unmarshal(seedJSON, &tools); err != nil {
		return 0, fmt.Errorf("解析 tools.json: %w", err)
	}
	tx, err := d.conn.Begin()
	if err != nil {
		return 0, err
	}
	defer tx.Rollback()
	for _, t := range tools {
		deps, _ := json.Marshal(t.Dependencies)
		if t.Dependencies == nil {
			deps = []byte("[]")
		}
		_, err := tx.Exec(`
INSERT INTO tools (name, category, description, default_env, is_high_risk, icon, dependencies)
VALUES (?, ?, ?, ?, ?, ?, ?)
ON CONFLICT(name) DO UPDATE SET
  category=excluded.category,
  description=excluded.description,
  default_env=excluded.default_env,
  is_high_risk=excluded.is_high_risk,
  icon=excluded.icon,
  dependencies=excluded.dependencies`,
			t.Name, t.Category, t.Description, t.DefaultEnv, t.IsHighRisk, t.Icon, string(deps))
		if err != nil {
			return 0, err
		}
	}
	if err := tx.Commit(); err != nil {
		return 0, err
	}
	return len(tools), nil
}

func (d *DB) ListTools(category string) ([]model.Tool, error) {
	query := "SELECT id, name, category, description, default_env, is_high_risk, use_count, icon, dependencies FROM tools"
	args := []any{}
	if category != "" {
		query += " WHERE category = ?"
		args = append(args, category)
	}
	query += " ORDER BY use_count DESC, name"
	rows, err := d.conn.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Tool
	for rows.Next() {
		var t model.Tool
		var deps string
		if err := rows.Scan(&t.ID, &t.Name, &t.Category, &t.Description, &t.DefaultEnv, &t.IsHighRisk, &t.UseCount, &t.Icon, &deps); err != nil {
			return nil, err
		}
		json.Unmarshal([]byte(deps), &t.Dependencies)
		out = append(out, t)
	}
	return out, rows.Err()
}

func (d *DB) GetTool(name string) (*model.Tool, error) {
	row := d.conn.QueryRow("SELECT id, name, category, description, default_env, is_high_risk, use_count, icon, dependencies FROM tools WHERE name = ?", name)
	var t model.Tool
	var deps string
	if err := row.Scan(&t.ID, &t.Name, &t.Category, &t.Description, &t.DefaultEnv, &t.IsHighRisk, &t.UseCount, &t.Icon, &deps); err != nil {
		return nil, err
	}
	json.Unmarshal([]byte(deps), &t.Dependencies)
	return &t, nil
}

func (d *DB) GetPreference(toolID int64) (string, bool, error) {
	var env sql.NullString
	err := d.conn.QueryRow("SELECT preferred_env FROM preferences WHERE tool_id = ?", toolID).Scan(&env)
	if err == sql.ErrNoRows {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return env.String, env.Valid, nil
}

func (d *DB) SetPreference(toolID int64, env string) error {
	_, err := d.conn.Exec(`
INSERT INTO preferences (tool_id, preferred_env, updated_at) VALUES (?, ?, ?)
ON CONFLICT(tool_id) DO UPDATE SET preferred_env = excluded.preferred_env, updated_at = excluded.updated_at`,
		toolID, env, time.Now())
	return err
}

func (d *DB) StartExecution(toolID int64, envUsed, args, outputPath string) (int64, error) {
	res, err := d.conn.Exec(
		"INSERT INTO executions (tool_id, env_used, args, start_time, output_path) VALUES (?, ?, ?, ?, ?)",
		toolID, envUsed, args, time.Now(), outputPath)
	if err != nil {
		return 0, err
	}
	return res.LastInsertId()
}

func (d *DB) FinishExecution(id int64, exitCode int) error {
	_, err := d.conn.Exec("UPDATE executions SET end_time = ?, exit_code = ? WHERE id = ?", time.Now(), exitCode, id)
	return err
}

func (d *DB) IncrementUseCount(toolID int64) error {
	_, err := d.conn.Exec("UPDATE tools SET use_count = use_count + 1 WHERE id = ?", toolID)
	return err
}

func (d *DB) RecentExecutions(limit int) ([]model.Execution, error) {
	rows, err := d.conn.Query(`
SELECT e.id, e.tool_id, t.name, e.env_used, e.args, e.start_time, e.end_time, e.exit_code, e.output_path
FROM executions e JOIN tools t ON t.id = e.tool_id
ORDER BY e.id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []model.Execution
	for rows.Next() {
		var e model.Execution
		var end sql.NullTime
		var code sql.NullInt64
		if err := rows.Scan(&e.ID, &e.ToolID, &e.ToolName, &e.EnvUsed, &e.Args, &e.StartTime, &end, &code, &e.OutputPath); err != nil {
			return nil, err
		}
		if end.Valid {
			e.EndTime = end.Time
		}
		if code.Valid {
			e.ExitCode = int(code.Int64)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}
```

- [ ] **Step 5: 运行测试确认通过**

```bash
go get modernc.org/sqlite && go test ./internal/db/ -v -cover
```

预期：PASS，覆盖率 >80%。

- [ ] **Step 6: Commit**

```bash
git add internal/db/ go.mod go.sum
git -c user.name=karen -c user.email=karen@edith.local commit -m "feat: SQLite 数据层（建表/种子导入/工具查询/偏好/执行记录，含单测）"
```

## Task 4: 完整种子数据 tools.json（≥100 工具）

**Files:**
- Modify: `internal/db/tools.json`（替换为完整数据）
- Test: `internal/db/sqlite_test.go`（新增种子规模与关键字段测试）

**Interfaces:**
- Consumes: `db.Open` 的 seed 流程
- Produces: 完整工具目录（10 个分类，≥100 条）

- [ ] **Step 1: 写失败测试（追加到 `internal/db/sqlite_test.go`）**

```go
func TestSeedFullDataset(t *testing.T) {
	d := newTestDB(t)
	var n int
	if err := d.conn.QueryRow("SELECT COUNT(*) FROM tools").Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n < 100 {
		t.Fatalf("工具数 = %d, want >= 100", n)
	}
	var cats int
	if err := d.conn.QueryRow("SELECT COUNT(DISTINCT category) FROM tools").Scan(&cats); err != nil {
		t.Fatal(err)
	}
	if cats < 10 {
		t.Fatalf("分类数 = %d, want >= 10", cats)
	}
	nmap, err := d.GetTool("nmap")
	if err != nil {
		t.Fatalf("nmap 应存在: %v", err)
	}
	if nmap.Description == "" || nmap.Icon == "" {
		t.Fatalf("nmap 缺中文描述或图标: %+v", nmap)
	}
	msf, err := d.GetTool("metasploit")
	if err != nil {
		t.Fatalf("metasploit 应存在: %v", err)
	}
	if !msf.IsHighRisk {
		t.Fatalf("metasploit 应为高危")
	}
	beef, err := d.GetTool("beef-xss")
	if err != nil {
		t.Fatalf("beef-xss 应存在: %v", err)
	}
	if len(beef.Dependencies) == 0 {
		t.Fatalf("beef-xss 应声明依赖")
	}
	for _, name := range []string{"volatility", "reaver", "bully", "bettercap", "aircrack-ng", "mdk3", "mdk4"} {
		tool, err := d.GetTool(name)
		if err != nil {
			t.Fatalf("%s 应存在: %v", name, err)
		}
		if !tool.IsHighRisk {
			t.Fatalf("%s 应为高危", name)
		}
	}
}
```

注意：Task 3 的 `TestOpenMigratesAndSeeds` 断言 seed 后行数为 3，会与本任务冲突。**将 `want 3` 改为 `want >= 100`**（该文件内的 `TestSeedIdempotent` 同理改为 `>= 100`）。`TestListToolsByCategory` 同样需调整：`len(all) != 3` 改为 `len(all) < 100`，且 scanner 断言由 `len(scan) != 1` 改为 `len(scan) < 1`（同时校验每个元素 Category == "scanner"）。

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/db/ -run 'TestSeedFullDataset|TestOpenMigratesAndSeeds|TestSeedIdempotent' -v
```

预期：FAIL（工具数不足）。

- [ ] **Step 3: 写完整 `internal/db/tools.json`**

内容为 JSON 数组，每元素一个单行对象，共 110 条，覆盖 10 个分类。完整内容如下（直接整体替换文件）：

```json
[
{"name":"nmap","category":"scanner","description":"网络扫描与主机发现","default_env":"local","is_high_risk":false,"icon":"🎯"},
{"name":"masscan","category":"scanner","description":"极速端口扫描器","default_env":"local","is_high_risk":false,"icon":"⚡"},
{"name":"zmap","category":"scanner","description":"全网段高速扫描","default_env":"local","is_high_risk":false,"icon":"🗺️"},
{"name":"rustscan","category":"scanner","description":"Rust 快速端口扫描","default_env":"local","is_high_risk":false,"icon":"🦀"},
{"name":"naabu","category":"scanner","description":"Go 端口扫描器","default_env":"local","is_high_risk":false,"icon":"🔦"},
{"name":"arp-scan","category":"scanner","description":"ARP 局域网主机发现","default_env":"local","is_high_risk":false,"icon":"📶"},
{"name":"dnsenum","category":"scanner","description":"DNS 信息枚举","default_env":"local","is_high_risk":false,"icon":"🌐"},
{"name":"fierce","category":"scanner","description":"DNS 域暴力枚举","default_env":"local","is_high_risk":false,"icon":"🐉"},
{"name":"amass","category":"scanner","description":"子域名与资产发现","default_env":"local","is_high_risk":false,"icon":"🧭"},
{"name":"subfinder","category":"scanner","description":"被动子域名枚举","default_env":"local","is_high_risk":false,"icon":"🔍"},
{"name":"whatweb","category":"scanner","description":"网站指纹识别","default_env":"local","is_high_risk":false,"icon":"🕸️"},
{"name":"nikto","category":"scanner","description":"Web 服务器漏洞扫描","default_env":"local","is_high_risk":false,"icon":"🕷️"},
{"name":"nuclei","category":"scanner","description":"模板化漏洞扫描器","default_env":"local","is_high_risk":false,"icon":"☢️"},
{"name":"dirb","category":"scanner","description":"Web 目录爆破","default_env":"local","is_high_risk":false,"icon":"📂"},
{"name":"gobuster","category":"scanner","description":"目录与子域爆破","default_env":"local","is_high_risk":false,"icon":"💥"},
{"name":"dirsearch","category":"scanner","description":"Python 目录扫描器","default_env":"local","is_high_risk":false,"icon":"🗂️"},
{"name":"ffuf","category":"scanner","description":"高速 Web 模糊测试","default_env":"local","is_high_risk":false,"icon":"🎲"},
{"name":"sqlmap","category":"scanner","description":"SQL 注入检测与利用","default_env":"local","is_high_risk":false,"icon":"💉"},
{"name":"sslyze","category":"scanner","description":"SSL/TLS 配置审计","default_env":"local","is_high_risk":false,"icon":"🔐"},
{"name":"testssl","category":"scanner","description":"SSL/TLS 扫描套件","default_env":"local","is_high_risk":false,"icon":"🛡️"},
{"name":"snmpwalk","category":"scanner","description":"SNMP 信息枚举","default_env":"local","is_high_risk":false,"icon":"📡"},
{"name":"enum4linux","category":"scanner","description":"SMB/CIFS 信息枚举","default_env":"local","is_high_risk":false,"icon":"🪟"},
{"name":"smbmap","category":"scanner","description":"SMB 共享枚举","default_env":"local","is_high_risk":false,"icon":"🗄️"},
{"name":"netexec","category":"scanner","description":"网络综合枚举与利用","default_env":"local","is_high_risk":false,"icon":"🧨"},
{"name":"sparta","category":"scanner","description":"网络基础设施审计","default_env":"podman","is_high_risk":false,"dependencies":["python2"],"icon":"🏛️"},
{"name":"hydra","category":"cracker","description":"在线暴力破解","default_env":"local","is_high_risk":false,"icon":"🐉"},
{"name":"john","category":"cracker","description":"离线密码破解","default_env":"local","is_high_risk":false,"icon":"🔑"},
{"name":"hashcat","category":"cracker","description":"GPU 哈希破解","default_env":"local","is_high_risk":false,"icon":"🎮"},
{"name":"medusa","category":"cracker","description":"并行暴力破解","default_env":"local","is_high_risk":false,"icon":"🐍"},
{"name":"ncrack","category":"cracker","description":"网络认证破解","default_env":"local","is_high_risk":false,"icon":"🕳️"},
{"name":"patator","category":"cracker","description":"多功能暴力破解","default_env":"local","is_high_risk":false,"icon":"⚙️"},
{"name":"cewl","category":"cracker","description":"网站关键词字典生成","default_env":"local","is_high_risk":false,"icon":"🕷️"},
{"name":"crunch","category":"cracker","description":"规则字典生成","default_env":"local","is_high_risk":false,"icon":"🧮"},
{"name":"rsmangler","category":"cracker","description":"字典变形工具","default_env":"local","is_high_risk":false,"icon":"🔀"},
{"name":"crowbar","category":"cracker","description":"远程认证爆破","default_env":"local","is_high_risk":false,"icon":"🔨"},
{"name":"acccheck","category":"cracker","description":"SMB 口令检测","default_env":"local","is_high_risk":false,"icon":"✅"},
{"name":"metasploit","category":"exploitation","description":"漏洞利用框架(msfconsole)","default_env":"vm","is_high_risk":true,"icon":"💀"},
{"name":"msfvenom","category":"exploitation","description":"载荷生成与编码","default_env":"vm","is_high_risk":true,"icon":"🧪"},
{"name":"searchsploit","category":"exploitation","description":"Exploit-DB 本地搜索","default_env":"local","is_high_risk":false,"icon":"📚"},
{"name":"beef-xss","category":"exploitation","description":"浏览器利用框架","default_env":"podman","is_high_risk":false,"dependencies":["python2"],"icon":"🐮"},
{"name":"routersploit","category":"exploitation","description":"路由器漏洞利用","default_env":"local","is_high_risk":false,"icon":"📶"},
{"name":"set","category":"exploitation","description":"社工工具包","default_env":"vm","is_high_risk":true,"icon":"🎭"},
{"name":"sqliv","category":"exploitation","description":"SQL 注入批量检测","default_env":"podman","is_high_risk":false,"dependencies":["python2"],"icon":"💉"},
{"name":"pwntools","category":"exploitation","description":"CTF 漏洞利用框架","default_env":"local","is_high_risk":false,"icon":"🐍"},
{"name":"armitage","category":"exploitation","description":"MSF 图形化管理","default_env":"vm","is_high_risk":true,"icon":"🎖️"},
{"name":"pwncat","category":"exploitation","description":"交互式提权 Shell","default_env":"local","is_high_risk":false,"icon":"🐈"},
{"name":"volatility","category":"forensics","description":"内存取证分析","default_env":"vm","is_high_risk":true,"icon":"🧠"},
{"name":"volatility3","category":"forensics","description":"内存取证分析(v3)","default_env":"local","is_high_risk":false,"icon":"🧬"},
{"name":"foremost","category":"forensics","description":"文件雕刻恢复","default_env":"local","is_high_risk":false,"icon":"🪚"},
{"name":"scalpel","category":"forensics","description":"文件雕刻恢复","default_env":"local","is_high_risk":false,"icon":"🔪"},
{"name":"binwalk","category":"forensics","description":"固件分析与提取","default_env":"local","is_high_risk":false,"icon":"📦"},
{"name":"autopsy","category":"forensics","description":"数字取证平台","default_env":"local","is_high_risk":false,"icon":"🔬"},
{"name":"sleuthkit","category":"forensics","description":"磁盘取证工具集","default_env":"local","is_high_risk":false,"icon":"🕵️"},
{"name":"testdisk","category":"forensics","description":"分区恢复工具","default_env":"local","is_high_risk":false,"icon":"💾"},
{"name":"photorec","category":"forensics","description":"照片与文件恢复","default_env":"local","is_high_risk":false,"icon":"📷"},
{"name":"exiftool","category":"forensics","description":"元数据读取与编辑","default_env":"local","is_high_risk":false,"icon":"🏷️"},
{"name":"pdf-parser","category":"forensics","description":"PDF 结构分析","default_env":"local","is_high_risk":false,"icon":"📄"},
{"name":"ddrescue","category":"forensics","description":"磁盘镜像恢复","default_env":"local","is_high_risk":false,"icon":"🆘"},
{"name":"extundelete","category":"forensics","description":"EXT 文件恢复","default_env":"local","is_high_risk":false,"icon":"🗃️"},
{"name":"aircrack-ng","category":"wireless","description":"无线破解套件","default_env":"vm","is_high_risk":true,"icon":"📡"},
{"name":"airodump-ng","category":"wireless","description":"无线抓包分析","default_env":"vm","is_high_risk":true,"icon":"📻"},
{"name":"aireplay-ng","category":"wireless","description":"无线注入攻击","default_env":"vm","is_high_risk":true,"icon":"🎯"},
{"name":"reaver","category":"wireless","description":"WPS PIN 破解","default_env":"vm","is_high_risk":true,"icon":"🔓"},
{"name":"bully","category":"wireless","description":"WPS PIN 破解","default_env":"vm","is_high_risk":true,"icon":"🔓"},
{"name":"wifite","category":"wireless","description":"自动化无线审计","default_env":"vm","is_high_risk":true,"icon":"⚔️"},
{"name":"mdk3","category":"wireless","description":"无线网络干扰","default_env":"vm","is_high_risk":true,"icon":"💣"},
{"name":"mdk4","category":"wireless","description":"无线网络干扰","default_env":"vm","is_high_risk":true,"icon":"💣"},
{"name":"kismet","category":"wireless","description":"无线嗅探与IDS","default_env":"local","is_high_risk":false,"icon":"👁️"},
{"name":"pixiewps","category":"wireless","description":"WPS 离线破解","default_env":"vm","is_high_risk":true,"icon":"🧩"},
{"name":"hcxdumptool","category":"wireless","description":"WPA 握手抓取","default_env":"vm","is_high_risk":true,"icon":"🤝"},
{"name":"hcxtools","category":"wireless","description":"握手包转换工具","default_env":"local","is_high_risk":false,"icon":"🔁"},
{"name":"fluxion","category":"wireless","description":"钓鱼AP攻击","default_env":"vm","is_high_risk":true,"icon":"🎣"},
{"name":"wifiphisher","category":"wireless","description":"钓鱼AP攻击","default_env":"vm","is_high_risk":true,"icon":"🎣"},
{"name":"wfuzz","category":"web","description":"Web 模糊测试","default_env":"local","is_high_risk":false,"icon":"🌊"},
{"name":"skipfish","category":"web","description":"Web 安全扫描","default_env":"local","is_high_risk":false,"icon":"🐟"},
{"name":"commix","category":"web","description":"命令注入利用","default_env":"local","is_high_risk":false,"icon":"⌨️"},
{"name":"cadaver","category":"web","description":"WebDAV 客户端","default_env":"local","is_high_risk":false,"icon":"🧟"},
{"name":"davtest","category":"web","description":"WebDAV 上传测试","default_env":"podman","is_high_risk":false,"dependencies":["python2"],"icon":"📤"},
{"name":"joomscan","category":"web","description":"Joomla 漏洞扫描","default_env":"local","is_high_risk":false,"icon":"🪴"},
{"name":"wpscan","category":"web","description":"WordPress 安全扫描","default_env":"local","is_high_risk":false,"icon":"🅿️"},
{"name":"xsser","category":"web","description":"XSS 检测与利用","default_env":"local","is_high_risk":false,"icon":"✖️"},
{"name":"xsstrike","category":"web","description":"XSS 扫描器","default_env":"local","is_high_risk":false,"icon":"⚡"},
{"name":"wafw00f","category":"web","description":"WAF 识别工具","default_env":"local","is_high_risk":false,"icon":"🧱"},
{"name":"httrack","category":"web","description":"网站镜像工具","default_env":"local","is_high_risk":false,"icon":"🪞"},
{"name":"sublist3r","category":"web","description":"子域名枚举","default_env":"local","is_high_risk":false,"icon":"📋"},
{"name":"dalfox","category":"web","description":"XSS 参数挖掘","default_env":"local","is_high_risk":false,"icon":"🦊"},
{"name":"arjun","category":"web","description":"HTTP 参数发现","default_env":"local","is_high_risk":false,"icon":"🏹"},
{"name":"theHarvester","category":"recon","description":"邮箱与子域搜集","default_env":"local","is_high_risk":false,"icon":"🌾"},
{"name":"recon-ng","category":"recon","description":"侦察框架","default_env":"local","is_high_risk":false,"icon":"🛰️"},
{"name":"dmitry","category":"recon","description":"信息收集工具","default_env":"local","is_high_risk":false,"icon":"📇"},
{"name":"dnsrecon","category":"recon","description":"DNS 侦察工具","default_env":"local","is_high_risk":false,"icon":"🔎"},
{"name":"spiderfoot","category":"recon","description":"自动化侦察平台","default_env":"local","is_high_risk":false,"icon":"🕸️"},
{"name":"urlcrazy","category":"recon","description":"域名变形生成","default_env":"local","is_high_risk":false,"icon":"🌀"},
{"name":"gau","category":"recon","description":"历史 URL 收集","default_env":"local","is_high_risk":false,"icon":"🕰️"},
{"name":"waybackurls","category":"recon","description":"历史 URL 收集","default_env":"local","is_high_risk":false,"icon":"🗞️"},
{"name":"katana","category":"recon","description":"现代网页爬虫","default_env":"local","is_high_risk":false,"icon":"🥷"},
{"name":"httpx","category":"recon","description":"HTTP 探测套件","default_env":"local","is_high_risk":false,"icon":"🌡️"},
{"name":"massdns","category":"recon","description":"高速 DNS 解析","default_env":"local","is_high_risk":false,"icon":"🚀"},
{"name":"altdns","category":"recon","description":"子域排列枚举","default_env":"local","is_high_risk":false,"icon":"🔠"},
{"name":"steghide","category":"stego","description":"图像隐写工具","default_env":"local","is_high_risk":false,"icon":"🖼️"},
{"name":"stegseek","category":"stego","description":"隐写高速爆破","default_env":"local","is_high_risk":false,"icon":"⚡"},
{"name":"zsteg","category":"stego","description":"PNG/LSB 隐写检测","default_env":"local","is_high_risk":false,"icon":"🦓"},
{"name":"outguess","category":"stego","description":"图像隐写工具","default_env":"local","is_high_risk":false,"icon":"🤫"},
{"name":"stegcracker","category":"stego","description":"隐写口令爆破","default_env":"local","is_high_risk":false,"icon":"🔨"},
{"name":"snow","category":"stego","description":"文本空白隐写","default_env":"local","is_high_risk":false,"icon":"❄️"},
{"name":"exiv2","category":"stego","description":"图像元数据工具","default_env":"local","is_high_risk":false,"icon":"📝"},
{"name":"openstego","category":"stego","description":"隐写与水印工具","default_env":"local","is_high_risk":false,"icon":"💧"},
{"name":"jsteg","category":"stego","description":"JPEG 隐写工具","default_env":"local","is_high_risk":false,"icon":"🧩"},
{"name":"cupp","category":"social","description":"密码画像生成","default_env":"local","is_high_risk":false,"icon":"☕"},
{"name":"king-phisher","category":"social","description":"钓鱼攻击平台","default_env":"podman","is_high_risk":false,"dependencies":["python2"],"icon":"🎣"},
{"name":"gophish","category":"social","description":"钓鱼模拟平台","default_env":"local","is_high_risk":false,"icon":"📧"},
{"name":"socialfish","category":"social","description":"钓鱼攻击工具","default_env":"podman","is_high_risk":false,"dependencies":["python2"],"icon":"🐟"},
{"name":"sherlock","category":"social","description":"用户名跨站搜索","default_env":"local","is_high_risk":false,"icon":"🔎"},
{"name":"holehe","category":"social","description":"邮箱注册查询","default_env":"local","is_high_risk":false,"icon":"📮"},
{"name":"maigret","category":"social","description":"用户名跨站搜索","default_env":"local","is_high_risk":false,"icon":"🕵️"},
{"name":"blackeye","category":"social","description":"钓鱼页面生成","default_env":"podman","is_high_risk":false,"dependencies":["python2"],"icon":"👁️"},
{"name":"zphisher","category":"social","description":"钓鱼页面生成","default_env":"local","is_high_risk":false,"icon":"🎭"},
{"name":"netcat","category":"automation","description":"网络瑞士军刀","default_env":"local","is_high_risk":false,"icon":"🔗"},
{"name":"ncat","category":"automation","description":"netcat 增强版","default_env":"local","is_high_risk":false,"icon":"🔗"},
{"name":"socat","category":"automation","description":"网络连接工具","default_env":"local","is_high_risk":false,"icon":"🔌"},
{"name":"tcpdump","category":"automation","description":"网络抓包分析","default_env":"local","is_high_risk":false,"icon":"📟"},
{"name":"tshark","category":"automation","description":"Wireshark 命令行","default_env":"local","is_high_risk":false,"icon":"🦈"},
{"name":"tcpflow","category":"automation","description":"TCP 流重组","default_env":"local","is_high_risk":false,"icon":"🌊"},
{"name":"mitmproxy","category":"automation","description":"HTTP 中间人代理","default_env":"local","is_high_risk":false,"icon":"🔄"},
{"name":"ettercap","category":"automation","description":"中间人攻击套件","default_env":"vm","is_high_risk":true,"icon":"🎭"},
{"name":"bettercap","category":"automation","description":"网络攻击框架","default_env":"vm","is_high_risk":true,"icon":"🧢"},
{"name":"responder","category":"automation","description":"LLMNR/NBT 毒化","default_env":"vm","is_high_risk":true,"icon":"📢"},
{"name":"mitm6","category":"automation","description":"IPv6 中间人攻击","default_env":"vm","is_high_risk":true,"icon":"🛡️"},
{"name":"mitmf","category":"automation","description":"中间人框架","default_env":"podman","is_high_risk":false,"dependencies":["python2"],"icon":"🎪"},
{"name":"arpspoof","category":"automation","description":"ARP 欺骗工具","default_env":"vm","is_high_risk":true,"icon":"🃏"},
{"name":"macchanger","category":"automation","description":"MAC 地址伪装","default_env":"local","is_high_risk":false,"icon":"🎭"}
]
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/db/ -v -cover
```

预期：PASS，行数 ≥100 断言通过。

- [ ] **Step 5: Commit**

```bash
git add internal/db/tools.json internal/db/sqlite_test.go
git -c user.name=karen -c user.email=karen@edith.local commit -m "feat: 内置 110 个 BlackArch 工具种子数据（10 分类，中文描述）"
```

## Task 5: 决策引擎

**Files:**
- Create: `internal/decision/engine.go`
- Test: `internal/decision/engine_test.go`

**Interfaces:**
- Consumes: `model.Tool`, `model.Decision`
- Produces:
  - `var HighRiskTools = []string{"metasploit","volatility","reaver","bully","bettercap","aircrack-ng","mdk3","mdk4"}`
  - `type PreferenceStore interface{ GetPreference(toolID int64) (string, bool, error) }`
  - `type Engine struct{ Prefs PreferenceStore; Which func(string) (string, bool); ConflictDeps map[string]bool }`
  - `func New(prefs PreferenceStore) *Engine`（默认 Which=exec.LookPath；ConflictDeps 含 "python2"）
  - `func (e *Engine) Decide(tool model.Tool, requestedEnv string) (model.Decision, error)`

- [ ] **Step 1: 写失败测试 `internal/decision/engine_test.go`**

```go
package decision

import (
	"errors"
	"testing"

	"blackarch-toolbox/internal/model"
)

type fakePrefs struct {
	env string
	ok  bool
	err error
}

func (f fakePrefs) GetPreference(int64) (string, bool, error) { return f.env, f.ok, f.err }

func engineWith(prefs PreferenceStore, which map[string]string) *Engine {
	e := New(prefs)
	e.Which = func(bin string) (string, bool) {
		p, ok := which[bin]
		return p, ok
	}
	return e
}

func TestDecideRequestedEnv(t *testing.T) {
	e := engineWith(fakePrefs{}, nil)
	d, err := e.Decide(model.Tool{Name: "nmap"}, "podman")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env != "podman" || d.Priority != 1 {
		t.Fatalf("显式指定环境失败: %+v", d)
	}
}

func TestDecidePreferenceWins(t *testing.T) {
	e := engineWith(fakePrefs{env: "vm", ok: true}, map[string]string{"nmap": "/usr/bin/nmap"})
	d, err := e.Decide(model.Tool{ID: 1, Name: "nmap"}, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env != "vm" || d.Priority != 1 {
		t.Fatalf("偏好应优先: %+v", d)
	}
}

func TestDecideHighRisk(t *testing.T) {
	e := engineWith(fakePrefs{}, map[string]string{"metasploit": "/usr/bin/metasploit"})
	d, err := e.Decide(model.Tool{ID: 2, Name: "metasploit", IsHighRisk: true}, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env != "vm" || d.Priority != 2 {
		t.Fatalf("高危应走 vm: %+v", d)
	}
}

func TestDecideDependencyConflict(t *testing.T) {
	e := engineWith(fakePrefs{}, map[string]string{"beef-xss": "/usr/bin/beef-xss"})
	d, err := e.Decide(model.Tool{ID: 3, Name: "beef-xss", Dependencies: []string{"python2"}}, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env != "podman" || d.Priority != 3 {
		t.Fatalf("依赖冲突应走 podman: %+v", d)
	}
}

func TestDecideLocalExists(t *testing.T) {
	e := engineWith(fakePrefs{}, map[string]string{"nmap": "/usr/bin/nmap"})
	d, err := e.Decide(model.Tool{ID: 4, Name: "nmap"}, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env != "local" || d.Priority != 4 {
		t.Fatalf("本地存在应走 local: %+v", d)
	}
}

func TestDecideFallbackVM(t *testing.T) {
	e := engineWith(fakePrefs{}, nil)
	d, err := e.Decide(model.Tool{ID: 5, Name: "不存在工具"}, "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env != "vm" || d.Priority != 5 {
		t.Fatalf("兜底应走 vm: %+v", d)
	}
}

func TestDecideInvalidEnv(t *testing.T) {
	e := engineWith(fakePrefs{}, nil)
	if _, err := e.Decide(model.Tool{Name: "nmap"}, "docker"); err == nil {
		t.Fatal("非法环境应报错")
	}
}

func TestDecidePrefError(t *testing.T) {
	e := engineWith(fakePrefs{err: errors.New("boom")}, nil)
	if _, err := e.Decide(model.Tool{ID: 9, Name: "nmap"}, "auto"); err == nil {
		t.Fatal("偏好查询错误应上抛")
	}
}

func TestHighRiskListContainsRequired(t *testing.T) {
	want := []string{"metasploit", "volatility", "reaver", "bully", "bettercap", "aircrack-ng", "mdk3", "mdk4"}
	got := make(map[string]bool)
	for _, v := range HighRiskTools {
		got[v] = true
	}
	for _, w := range want {
		if !got[w] {
			t.Fatalf("HighRiskTools 缺 %s", w)
		}
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/decision/ -v
```

预期：FAIL（package 不存在）。

- [ ] **Step 3: 实现 `internal/decision/engine.go`**

```go
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
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/decision/ -v -cover
```

预期：PASS，覆盖率 >80%。

- [ ] **Step 5: Commit**

```bash
git add internal/decision/
git -c user.name=karen -c user.email=karen@edith.local commit -m "feat: 决策引擎（偏好/高危/依赖冲突/本地探测/兜底，含单测）"
```

---

## Task 6: 执行器（本地）

**Files:**
- Create: `internal/executor/runner.go`、`internal/executor/local.go`
- Test: `internal/executor/helper_test.go`、`internal/executor/local_test.go`

**Interfaces:**
- Consumes: 无
- Produces:
  - `type Runner interface{ Run(ctx context.Context, tool, args, workDir string, onLine func(string)) (int, error) }`
  - `type Local struct{ LookPath func(string) (string, error); ExecCmd func(ctx context.Context, name string, args ...string) *exec.Cmd }`
  - `func NewLocal() *Local`

**实现要点**：args 用 `shell.Fields` 拆分；`cmd.Dir = workDir`；StdoutPipe+StderrPipe 双扫描 goroutine 逐行回调 onLine；退出码取 `ProcessState.ExitCode()`。测试用 GO_WANT_HELPER_PROCESS 子进程模式（标准 exec 测试法）。

- [ ] **Step 1: 写共享测试助手 `internal/executor/helper_test.go`**

```go
package executor

import (
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
```

说明：TestMain 拦截子进程调用；RECORD_FILE 记录 argv（Task 7 用）。`helperCommand` 不指定具体测试名（TestMain 直接短路，无需 run 过滤）。

- [ ] **Step 2: 写失败测试 `internal/executor/local_test.go`**

```go
package executor

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"
)

func TestLocalRunCapturesLines(t *testing.T) {
	l := NewLocal()
	l.LookPath = func(string) (string, error) { return "/fake/nmap", nil }
	l.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCommandExit(0, args...)
	}
	var lines []string
	code, err := l.Run(context.Background(), "nmap", "-sV 192.168.1.1", "/tmp", func(line string) {
		lines = append(lines, line)
	})
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit = %d, want 0", code)
	}
	if len(lines) < 2 {
		t.Fatalf("行数 = %d, want >= 2", len(lines))
	}
	if lines[0] != "-sV" || lines[1] != "192.168.1.1" {
		t.Fatalf("参数拆分错误: %v", lines)
	}
	foundErr := false
	for _, l := range lines {
		if strings.Contains(l, "stderr-line") {
			foundErr = true
		}
	}
	if !foundErr {
		t.Fatal("应捕获 stderr 行")
	}
}

func TestLocalRunExitCode(t *testing.T) {
	l := NewLocal()
	l.LookPath = func(string) (string, error) { return "/fake/x", nil }
	l.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCommandExit(3, args...)
	}
	code, err := l.Run(context.Background(), "x", "", "/tmp", nil)
	if err != nil {
		t.Fatalf("退出码非 0 不应返回 error: %v", err)
	}
	if code != 3 {
		t.Fatalf("exit = %d, want 3", code)
	}
}

func TestLocalRunToolNotFound(t *testing.T) {
	l := NewLocal()
	l.LookPath = func(string) (string, error) { return "", errors.New("not found") }
	if _, err := l.Run(context.Background(), "ghost-tool", "", "/tmp", nil); err == nil {
		t.Fatal("工具不存在应报错")
	}
}

func TestLocalRunCancel(t *testing.T) {
	l := NewLocal()
	l.LookPath = func(string) (string, error) { return "/fake/slow", nil }
	l.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCommandExit(0, "sleep")
	}
	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()
	if _, err := l.Run(ctx, "slow", "", "/tmp", nil); err == nil {
		t.Fatal("ctx 超时应报错")
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

```bash
go test ./internal/executor/ -v
```

预期：FAIL（package 不存在）。

- [ ] **Step 4: 实现 `internal/executor/runner.go` 与 `internal/executor/local.go`**

`runner.go`：

```go
package executor

import "context"

type Runner interface {
	Run(ctx context.Context, tool, args, workDir string, onLine func(line string)) (int, error)
}
```

`local.go`：

```go
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
	fields, err := shell.Fields(args)
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
```

注意：`TestLocalRunCancel` 的 helper 子进程 sleep 30 秒——ctx 在 50ms 超时后，`exec.CommandContext` 会杀掉子进程，`cmd.Wait` 返回非 nil 错误，Run 先检查 `ctx.Err()` 并返回超时错误，测试稳定。

- [ ] **Step 5: 运行测试确认通过**

```bash
go get mvdan.cc/sh/v3 && go test ./internal/executor/ -v -cover
```

预期：PASS，覆盖率 >80%。

- [ ] **Step 6: Commit**

```bash
git add internal/executor/ go.mod go.sum
git -c user.name=karen -c user.email=karen@edith.local commit -m "feat: 本地执行器（参数拆分/逐行日志/退出码/取消，含单测）"
```

## Task 7: 执行器（Podman + VM/SSH）

**Files:**
- Create: `internal/executor/podman.go`、`internal/executor/vm.go`
- Test: `internal/executor/podman_test.go`、`internal/executor/vm_test.go`

**Interfaces:**
- Consumes: `Runner` 接口、`helper_test.go` 助手
- Produces:
  - `type Podman struct{ Container string; ExecCmd func(ctx context.Context, name string, args ...string) *exec.Cmd }`；`func NewPodman(container string) *Podman`
  - `type VM struct{ Host, User string; Port int; ExecCmd func(...) }`；`func NewVM(host, user string, port int) *VM`

**实现要点**：
- podman：`podman exec -i <container> /bin/bash -lc "cd <quoted workDir> && <tool> <args>"`；若退出码 125 且 stderr 含 "not running"，先 `podman start <container>` 重试一次。workDir 用 `shell.Quote` 包裹。
- vm：`ssh -p <port> -o BatchMode=yes -o StrictHostKeyChecking=no <user>@<host> "mkdir -p <quoted workDir> && cd <quoted workDir> && <tool> <args>"`。
- 两个测试都用 helper 子进程：断言 podman/ssh 被调用时 argv 的正确性，以及日志行透传。

- [ ] **Step 1: 写失败测试 `internal/executor/podman_test.go`**

```go
package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestPodmanRunCommandShape(t *testing.T) {
	record := filepath.Join(t.TempDir(), "argv.txt")
	p := NewPodman("blackarch-tools")
	p.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmd := helperCommandExit(0, args...)
		cmd.Env = append(cmd.Env, "RECORD_FILE="+record)
		return cmd
	}
	var lines []string
	code, err := p.Run(context.Background(), "nmap", "-sV 1.2.3.4", "/work/dir", func(l string) { lines = append(lines, l) })
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if len(lines) == 0 {
		t.Fatal("应透传日志行")
	}
	data, err := os.ReadFile(record)
	if err != nil {
		t.Fatal(err)
	}
	joined := string(data)
	for _, want := range []string{"exec", "-i", "blackarch-tools", "/bin/bash", "-lc"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("argv 缺 %q: %s", want, joined)
		}
	}
	if !strings.Contains(joined, "cd /work/dir") {
		t.Fatalf("argv 缺 cd 命令: %s", joined)
	}
}

func TestPodmanRetryOn125(t *testing.T) {
	record := filepath.Join(t.TempDir(), "argv.txt")
	p := NewPodman("blackarch-tools")
	calls := 0
	p.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		calls++
		if calls == 1 {
			return helperCommandExit(125, args...)
		}
		cmd := helperCommandExit(0, args...)
		cmd.Env = append(cmd.Env, "RECORD_FILE="+record)
		return cmd
	}
	code, err := p.Run(context.Background(), "nmap", "", "/work/dir", nil)
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("重试后应成功, exit = %d", code)
	}
	if calls != 3 {
		t.Fatalf("调用次数 = %d, want 3 (exec 失败 + start + exec 重试)", calls)
	}
	data, _ := os.ReadFile(record)
	joined := string(data)
	if !strings.Contains(joined, "exec") {
		t.Fatalf("重试应为 exec: %s", joined)
	}
}
```

注意：helper 子进程需记录 argv。修改 `helper_test.go` 的 `TestHelperProcess`，在打印参数前增加：

```go
	if f := os.Getenv("RECORD_FILE"); f != "" {
		os.WriteFile(f, []byte(strings.Join(args, "\n")+"\n"), 0o644)
	}
```

- [ ] **Step 2: 写失败测试 `internal/executor/vm_test.go`**

```go
package executor

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestVMRunCommandShape(t *testing.T) {
	record := filepath.Join(t.TempDir(), "argv.txt")
	v := NewVM("192.168.122.2", "blackarch", 22)
	v.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		cmd := helperCommandExit(0, args...)
		cmd.Env = append(cmd.Env, "RECORD_FILE="+record)
		return cmd
	}
	var lines []string
	code, err := v.Run(context.Background(), "nmap", "-sV 1.2.3.4", "/work/dir", func(l string) { lines = append(lines, l) })
	if err != nil {
		t.Fatal(err)
	}
	if code != 0 {
		t.Fatalf("exit = %d", code)
	}
	if len(lines) == 0 {
		t.Fatal("应透传日志行")
	}
	data, _ := os.ReadFile(record)
	joined := string(data)
	for _, want := range []string{"-p", "22", "-o", "BatchMode=yes", "-o", "StrictHostKeyChecking=no", "blackarch@192.168.122.2"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("argv 缺 %q: %s", want, joined)
		}
	}
	if !strings.Contains(joined, "mkdir -p /work/dir") || !strings.Contains(joined, "cd /work/dir") {
		t.Fatalf("远程命令缺 mkdir/cd: %s", joined)
	}
}

func TestVMSSHFail(t *testing.T) {
	v := NewVM("192.168.122.2", "blackarch", 22)
	v.ExecCmd = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return helperCommandExit(255, args...)
	}
	code, err := v.Run(context.Background(), "nmap", "", "/work/dir", nil)
	if err != nil {
		t.Fatal(err)
	}
	if code != 255 {
		t.Fatalf("ssh 免密失败应透传退出码 255, got %d", code)
	}
}
```

- [ ] **Step 3: 运行测试确认失败**

```bash
go test ./internal/executor/ -run 'TestPodman|TestVM' -v
```

预期：FAIL（NewPodman/NewVM 未定义）。

- [ ] **Step 4: 实现 `internal/executor/podman.go`**

```go
package executor

import (
	"context"
	"fmt"
	"os/exec"

	"mvdan.cc/sh/v3/shell"
)

type Podman struct {
	Container string
	ExecCmd   func(ctx context.Context, name string, args ...string) *exec.Cmd
}

func NewPodman(container string) *Podman {
	return &Podman{Container: container, ExecCmd: exec.CommandContext}
}

func (p *Podman) Run(ctx context.Context, tool, args, workDir string, onLine func(string)) (int, error) {
	cmdLine := fmt.Sprintf("cd %s && %s %s", shell.Quote(workDir), tool, args)
	full := []string{"exec", "-i", p.Container, "/bin/bash", "-lc", cmdLine}
	code, err := p.runOnce(ctx, full, onLine)
	if err == nil && code == 125 {
		start := []string{"start", p.Container}
		if _, serr := p.runOnce(ctx, start, onLine); serr != nil {
			return -1, fmt.Errorf("podman start 失败: %w", serr)
		}
		return p.runOnce(ctx, full, onLine)
	}
	return code, err
}

func (p *Podman) runOnce(ctx context.Context, args []string, onLine func(string)) (int, error) {
	cmd := p.ExecCmd(ctx, "podman", args...)
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
		return -1, err
	}
	return 0, nil
}
```

- [ ] **Step 5: 实现 `internal/executor/vm.go`**

```go
package executor

import (
	"context"
	"fmt"
	"os/exec"
	"strconv"

	"mvdan.cc/sh/v3/shell"
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
	remote := fmt.Sprintf("mkdir -p %s && cd %s && %s %s", shell.Quote(workDir), shell.Quote(workDir), tool, args)
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
```

（`bufio`/`io` 未直接使用时删掉对应 import。）

- [ ] **Step 6: 运行测试确认通过**

```bash
go test ./internal/executor/ -v -cover
```

预期：PASS，覆盖率 >80%。

- [ ] **Step 7: Commit**

```bash
git add internal/executor/
git -c user.name=karen -c user.email=karen@edith.local commit -m "feat: podman 与 VM(SSH) 执行器（容器自愈重试/BatchMode 免密，含单测）"
```

---

## Task 8: 产物管理器

**Files:**
- Create: `internal/workspace/manager.go`
- Test: `internal/workspace/manager_test.go`

**Interfaces:**
- Consumes: `model.Metadata`, `model.FileEntry`
- Produces:
  - `func New(root string) *Manager`
  - `(*Manager) CreateRunDir(tool string, now time.Time) (string, error)`：`<root>/<tool>/2006-01-02_15-04-05`
  - `(*Manager) WriteMetadata(dir string, meta model.Metadata) error`：写 `.metadata.json`（0644，indent 2）
  - `(*Manager) List(rel string) ([]model.FileEntry, error)`：rel="" 列出根；条目含 IsDir/Size
  - `(*Manager) Resolve(rel string) (string, error)`：Clean 后强制前缀校验（防穿越）
  - `(*Manager) EnsureWritable() error`

- [ ] **Step 1: 写失败测试 `internal/workspace/manager_test.go`**

```go
package workspace

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"blackarch-toolbox/internal/model"
)

func newMgr(t *testing.T) *Manager {
	t.Helper()
	m := New(filepath.Join(t.TempDir(), "ws"))
	if err := os.MkdirAll(m.Root, 0o755); err != nil {
		t.Fatal(err)
	}
	return m
}

func TestCreateRunDir(t *testing.T) {
	m := newMgr(t)
	dir, err := m.CreateRunDir("nmap", time.Date(2026, 8, 16, 14, 30, 22, 0, time.Local))
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(m.Root, "nmap", "2026-08-16_14-30-22")
	if dir != want {
		t.Fatalf("dir = %q, want %q", dir, want)
	}
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("目录应已创建: %v", err)
	}
}

func TestWriteAndReadMetadata(t *testing.T) {
	m := newMgr(t)
	dir, _ := m.CreateRunDir("nmap", time.Now())
	meta := model.Metadata{
		Tool:        "nmap",
		Environment: "local",
		ExecutedAt:  time.Now(),
		Command:     "nmap -sV 1.2.3.4",
		ExitCode:    0,
		OutputDir:   dir,
	}
	if err := m.WriteMetadata(dir, meta); err != nil {
		t.Fatal(err)
	}
	entry, err := os.Stat(filepath.Join(dir, ".metadata.json"))
	if err != nil {
		t.Fatalf(".metadata.json 应存在: %v", err)
	}
	if entry.Size() == 0 {
		t.Fatal("metadata 为空")
	}
}

func TestListEntries(t *testing.T) {
	m := newMgr(t)
	dir, _ := m.CreateRunDir("nmap", time.Now())
	os.WriteFile(filepath.Join(dir, "scan.txt"), []byte("data"), 0o644)
	os.WriteFile(filepath.Join(dir, ".metadata.json"), []byte("{}"), 0o644)
	entries, err := m.List("nmap")
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) != 1 || entries[0].Name != "2026-08-16_14-30-22" || !entries[0].IsDir {
		t.Fatalf("List(nmap) = %+v", entries)
	}
	files, err := m.List(entries[0].Path)
	if err != nil {
		t.Fatal(err)
	}
	if len(files) != 2 {
		t.Fatalf("List(run dir) = %+v, want 2 文件", files)
	}
}

func TestResolveBlocksTraversal(t *testing.T) {
	m := newMgr(t)
	if _, err := m.Resolve("../etc"); err == nil {
		t.Fatal("路径穿越应被拒绝")
	}
	if _, err := m.Resolve("a/../../b"); err == nil {
		t.Fatal("间接穿越应被拒绝")
	}
	if _, err := m.Resolve("nmap/2026-01-01_00-00-00"); err != nil {
		t.Fatalf("正常路径应通过: %v", err)
	}
	if _, err := m.Resolve("/etc/passwd"); err == nil {
		t.Fatal("绝对路径应被拒绝")
	}
}

func TestEnsureWritable(t *testing.T) {
	m := newMgr(t)
	if err := m.EnsureWritable(); err != nil {
		t.Fatalf("应可写: %v", err)
	}
	if err := os.Chmod(m.Root, 0o555); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { os.Chmod(m.Root, 0o755) })
	if err := m.EnsureWritable(); err == nil {
		t.Fatal("只读目录应报错")
	}
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/workspace/ -v
```

预期：FAIL（package 不存在）。

- [ ] **Step 3: 实现 `internal/workspace/manager.go`**

```go
package workspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"blackarch-toolbox/internal/model"
)

type Manager struct {
	Root string
}

func New(root string) *Manager {
	return &Manager{Root: root}
}

func (m *Manager) CreateRunDir(tool string, now time.Time) (string, error) {
	dir := filepath.Join(m.Root, tool, now.Format("2006-01-02_15-04-05"))
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	return dir, nil
}

func (m *Manager) WriteMetadata(dir string, meta model.Metadata) error {
	data, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, ".metadata.json"), data, 0o644)
}

func (m *Manager) Resolve(rel string) (string, error) {
	if rel == "" {
		rel = "."
	}
	if filepath.IsAbs(rel) {
		return "", fmt.Errorf("非法路径: %q", rel)
	}
	clean := filepath.Clean(filepath.Join(m.Root, rel))
	if clean != m.Root && !strings.HasPrefix(clean, m.Root+string(filepath.Separator)) {
		return "", fmt.Errorf("非法路径: %q", rel)
	}
	return clean, nil
}

func (m *Manager) List(rel string) ([]model.FileEntry, error) {
	abs, err := m.Resolve(rel)
	if err != nil {
		return nil, err
	}
	if _, err := os.Stat(abs); err != nil {
		return nil, err
	}
	des, err := os.ReadDir(abs)
	if err != nil {
		return nil, err
	}
	entries := make([]model.FileEntry, 0, len(des))
	for _, de := range des {
		info, err := de.Info()
		if err != nil {
			continue
		}
		childAbs := filepath.Join(abs, de.Name())
		relPath, _ := filepath.Rel(m.Root, childAbs)
		entries = append(entries, model.FileEntry{
			Name:  de.Name(),
			Path:  relPath,
			IsDir: de.IsDir(),
			Size:  info.Size(),
		})
	}
	return entries, nil
}

func (m *Manager) EnsureWritable() error {
	if err := os.MkdirAll(m.Root, 0o755); err != nil {
		return err
	}
	probe := filepath.Join(m.Root, ".write-probe")
	if err := os.WriteFile(probe, []byte("x"), 0o644); err != nil {
		return fmt.Errorf("产物目录不可写: %w", err)
	}
	return os.Remove(probe)
}
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/workspace/ -v -cover
```

预期：PASS，覆盖率 >80%。

- [ ] **Step 5: Commit**

```bash
git add internal/workspace/
git -c user.name=karen -c user.email=karen@edith.local commit -m "feat: 产物管理器（运行目录/metadata/防穿越/可写检查，含单测）"
```

## Task 9: 健康审计

**Files:**
- Create: `internal/audit/health.go`
- Test: `internal/audit/health_test.go`

**Interfaces:**
- Consumes: `model.HealthCheck`
- Produces:
  - `type Runner func(ctx context.Context, name string, args ...string) (string, int, error)`（返回 stdout、退出码；已注入）
  - `type Auditor struct{ Container, VMName, WorkspacePath string; MinDiskFree uint64; Timeout time.Duration; Run Runner; StatDir func(string) error; DiskFree func(string) uint64 }`
  - `func New(container, vmName, workspacePath string) *Auditor`（默认 Run=exec.CommandContext+CombinedOutput、StatDir=写探测、DiskFree=Statfs 可用字节、Timeout=5s、MinDiskFree=2GB）
  - `func (a *Auditor) CheckAll(ctx context.Context) []model.HealthCheck`（6 项并发，每项独立超时）

**6 项判定**（见 spec 第 8 节）：
1. podman 服务：`podman info` 退出 0=ok
2. 目标容器：`podman ps -q -f name=<container>` 非空=ok；空则 `podman image exists <container>` 退出 0=warning("容器未运行")，否则 error
3. libvirtd：`systemctl is-active libvirtd` 输出 active=ok，否则 warning
4. VM 状态：`virsh list --state-running` 含 VMName=ok；不含则 `virsh list --all` 含=warning("VM 已关机")，否则 error
5. 产物目录：StatDir() 无错=ok
6. 磁盘空间：DiskFree(workspace) >= 2GB=ok

- [ ] **Step 1: 写失败测试 `internal/audit/health_test.go`**

```go
package audit

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"blackarch-toolbox/internal/model"
)

type fakeRun struct {
	results map[string]fakeResult
	calls   []string
}

type fakeResult struct {
	out string
	code int
	err  error
}

func (f *fakeRun) run(ctx context.Context, name string, args ...string) (string, int, error) {
	key := name + " " + strings.Join(args, " ")
	f.calls = append(f.calls, key)
	r, ok := f.results[key]
	if !ok {
		return "", 127, nil
	}
	return r.out, r.code, r.err
}

func newAuditor(f *fakeRun) *Auditor {
	a := New("blackarch-tools", "blackarch", "/tmp/ws")
	a.Run = f.run
	a.Timeout = time.Second
	a.StatDir = func(string) error { return nil }
	a.DiskFree = func(string) uint64 { return 10 << 30 }
	return a
}

func TestAllOK(t *testing.T) {
	f := &fakeRun{results: map[string]fakeResult{
		"podman info":                           {out: "ok", code: 0},
		"podman ps -q -f name=blackarch-tools":  {out: "abc123", code: 0},
		"systemctl is-active libvirtd":          {out: "active", code: 0},
		"virsh list --state-running":            {out: " Id Name\n 1  blackarch\n", code: 0},
	}}
	checks := newAuditor(f).CheckAll(context.Background())
	if len(checks) != 6 {
		t.Fatalf("应返回 6 项, got %d", len(checks))
	}
	for _, c := range checks {
		if c.Status != "ok" {
			t.Fatalf("%s: status = %s, want ok (%s)", c.Name, c.Status, c.Detail)
		}
	}
}

func TestContainerNotRunning(t *testing.T) {
	f := &fakeRun{results: map[string]fakeResult{
		"podman info":                          {code: 0},
		"podman ps -q -f name=blackarch-tools": {code: 0, out: ""},
		"podman image exists blackarch-tools":  {code: 0},
		"systemctl is-active libvirtd":         {out: "active", code: 0},
		"virsh list --state-running":           {out: " Id Name\n", code: 0},
		"virsh list --all":                     {out: " -  blackarch 关闭\n", code: 0},
	}}
	checks := newAuditor(f).CheckAll(context.Background())
	if statusOf(checks, "目标容器") != "warning" {
		t.Fatalf("容器未运行应 warning: %+v", checks)
	}
	if statusOf(checks, "VM 状态") != "warning" {
		t.Fatalf("VM 关机应 warning: %+v", checks)
	}
}

func TestPodmanDownAndVMGone(t *testing.T) {
	f := &fakeRun{results: map[string]fakeResult{
		"podman info":                          {code: 1, out: "cannot connect"},
		"podman ps -q -f name=blackarch-tools": {code: 1},
		"podman image exists blackarch-tools":  {code: 1},
		"systemctl is-active libvirtd":         {out: "inactive", code: 3},
		"virsh list --state-running":           {code: 0, out: " Id Name\n"},
		"virsh list --all":                     {code: 0, out: " Id Name\n"},
	}}
	checks := newAuditor(f).CheckAll(context.Background())
	if statusOf(checks, "Podman 服务") != "error" {
		t.Fatalf("podman 服务应 error: %+v", checks)
	}
	if statusOf(checks, "VM 状态") != "error" {
		t.Fatalf("VM 不存在应 error: %+v", checks)
	}
}

func TestWorkspaceAndDisk(t *testing.T) {
	f := &fakeRun{}
	a := newAuditor(f)
	a.StatDir = func(string) error { return errors.New("no dir") }
	a.DiskFree = func(string) uint64 { return 1 << 30 }
	checks := a.CheckAll(context.Background())
	if statusOf(checks, "产物目录") != "error" {
		t.Fatalf("产物目录应 error: %+v", checks)
	}
	if statusOf(checks, "磁盘空间") != "error" {
		t.Fatalf("磁盘空间应 error: %+v", checks)
	}
}

func statusOf(checks []model.HealthCheck, name string) string {
	for _, c := range checks {
		if c.Name == name {
			return c.Status
		}
	}
	return ""
}
```

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/audit/ -v
```

预期：FAIL（package 不存在）。

- [ ] **Step 3: 实现 `internal/audit/health.go`**

```go
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
```

- [ ] **Step 4: 运行测试确认通过**

```bash
go test ./internal/audit/ -v -cover
```

预期：PASS，覆盖率 >80%。

- [ ] **Step 5: Commit**

```bash
git add internal/audit/
git -c user.name=karen -c user.email=karen@edith.local commit -m "feat: 健康审计（6 项并发检查/超时兜底，含单测）"
```

## Task 10: App 服务层 + main.go 接线

**Files:**
- Create: `internal/app/app.go`
- Modify: `main.go`（替换模板）、删除模板 `app.go`（根目录，若 wails 模板生成）
- Test: `internal/app/app_test.go`

**Interfaces:**
- Consumes: db/decision/executor/workspace/audit/config 全部产物
- Produces:
  - `func New(cfg *config.Config, dbPath string) (*App, error)`
  - `(*App) GetTools(category string) ([]model.Tool, error)`
  - `(*App) DryRun(toolName, args, env string) (model.Decision, error)`
  - `(*App) RunTool(req model.RunRequest) (model.RunResult, error)`
  - `(*App) RunHealthCheck() ([]model.HealthCheck, error)`
  - `(*App) ListWorkspace(path string) ([]model.FileEntry, error)`
  - `(*App) OpenWorkspaceFile(path string) error`（xdg-open，可注入 openFn）
  - `(*App) SetPreference(toolName, env string) error`
  - `(*App) SetEventEmitter(fn func(event string, data ...any))`
  - Events：`toolbox:log:{id}`（逐行）、`toolbox:logend:{id}`（结束，payload map：exit_code, work_dir, env）

**实现要点**：RunTool 同步返回（校验工具/决策/建目录/落库 executions 行），实际执行在后台 goroutine：跑 Runner、写 `output.log`、写 `.metadata.json`、FinishExecution、IncrementUseCount、发 logend 事件。事件常量 `EventLogPrefix = "toolbox:log:"`、`EventLogEndPrefix = "toolbox:logend:"`。App 内部持有 `Exec map[string]executor.Runner`、`Dec *decision.Engine`、`DB *db.DB`、`WS *workspace.Manager`、`Aud *audit.Auditor`、`OpenFn`（默认 xdg-open）。

- [ ] **Step 1: 写失败测试 `internal/app/app_test.go`**

```go
package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"sync"
	"testing"
	"time"

	"blackarch-toolbox/internal/config"
	"blackarch-toolbox/internal/executor"
	"blackarch-toolbox/internal/model"
)

type fakeRunner struct {
	mu     sync.Mutex
	calls  []string
	lines  []string
	code   int
	sleep  time.Duration
}

func (f *fakeRunner) Run(ctx context.Context, tool, args, workDir string, onLine func(string)) (int, error) {
	f.mu.Lock()
	f.calls = append(f.calls, tool+"|"+args+"|"+workDir)
	f.mu.Unlock()
	if f.sleep > 0 {
		time.Sleep(f.sleep)
	}
	for _, l := range f.lines {
		onLine(l)
	}
	return f.code, nil
}

func newTestApp(t *testing.T) (*App, *fakeRunner, chan string) {
	t.Helper()
	cfg := &config.Config{}
	cfg.VM.Host = "192.168.122.2"
	cfg.VM.User = "blackarch"
	cfg.VM.Port = 22
	cfg.VM.Name = "blackarch"
	cfg.Workspace.Path = filepath.Join(t.TempDir(), "ws")
	cfg.Podman.Container = "blackarch-tools"
	a, err := New(cfg, filepath.Join(t.TempDir(), "app.db"))
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	events := make(chan string, 100)
	a.SetEventEmitter(func(event string, _ ...any) { events <- event })
	fr := &fakeRunner{lines: []string{"line1", "line2"}, code: 0}
	a.Exec = map[string]executor.Runner{"local": fr, "podman": fr, "vm": fr}
	t.Cleanup(func() { a.DB.Close() })
	return a, fr, events
}

func TestGetTools(t *testing.T) {
	a, _, _ := newTestApp(t)
	tools, err := a.GetTools("")
	if err != nil {
		t.Fatal(err)
	}
	if len(tools) < 100 {
		t.Fatalf("工具数 = %d, want >= 100", len(tools))
	}
	scanners, err := a.GetTools("scanner")
	if err != nil {
		t.Fatal(err)
	}
	for _, s := range scanners {
		if s.Category != "scanner" {
			t.Fatalf("分类过滤失败: %+v", s)
		}
	}
}

func TestRunToolEndToEnd(t *testing.T) {
	a, fr, events := newTestApp(t)
	res, err := a.RunTool(model.RunRequest{Tool: "nmap", Args: "-sV 192.168.1.1", Env: "auto"})
	if err != nil {
		t.Fatal(err)
	}
	if res.ExecutionID == 0 || res.EnvUsed == "" || res.WorkDir == "" {
		t.Fatalf("RunResult 不完整: %+v", res)
	}
	deadline := time.After(3 * time.Second)
	for {
		select {
		case ev := <-events:
			if ev == EventLogEndPrefix+strconv.FormatInt(res.ExecutionID, 10) {
				goto done
			}
		case <-deadline:
			t.Fatal("等待 logend 事件超时")
		}
	}
done:
	fr.mu.Lock()
	if len(fr.calls) != 1 {
		fr.mu.Unlock()
		t.Fatalf("runner 调用数 = %d", len(fr.calls))
	}
	fr.mu.Unlock()
	if _, err := os.Stat(filepath.Join(res.WorkDir, "output.log")); err != nil {
		t.Fatalf("output.log 应存在: %v", err)
	}
	metaData, err := os.ReadFile(filepath.Join(res.WorkDir, ".metadata.json"))
	if err != nil {
		t.Fatalf(".metadata.json 应存在: %v", err)
	}
	var meta model.Metadata
	if err := json.Unmarshal(metaData, &meta); err != nil {
		t.Fatal(err)
	}
	if meta.Tool != "nmap" || meta.Environment != res.EnvUsed || meta.ExitCode != 0 {
		t.Fatalf("metadata 不符: %+v", meta)
	}
}

func TestRunToolUnknownTool(t *testing.T) {
	a, _, _ := newTestApp(t)
	if _, err := a.RunTool(model.RunRequest{Tool: "ghost-tool", Env: "auto"}); err == nil {
		t.Fatal("未知工具应报错")
	}
}

func TestDryRunAndPreference(t *testing.T) {
	a, _, _ := newTestApp(t)
	d, err := a.DryRun("nmap", "", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d.Env == "" {
		t.Fatal("DryRun 应返回决策")
	}
	if err := a.SetPreference("nmap", "podman"); err != nil {
		t.Fatal(err)
	}
	d2, err := a.DryRun("nmap", "", "auto")
	if err != nil {
		t.Fatal(err)
	}
	if d2.Env != "podman" {
		t.Fatalf("偏好应生效: %+v", d2)
	}
	if err := a.SetPreference("ghost", "vm"); err == nil {
		t.Fatal("未知工具设偏好应报错")
	}
}

func TestHealthAndWorkspace(t *testing.T) {
	a, _, _ := newTestApp(t)
	a.Aud.Run = func(ctx context.Context, name string, args ...string) (string, int, error) {
		return "", 0, nil
	}
	a.Aud.StatDir = func(string) error { return nil }
	a.Aud.DiskFree = func(string) uint64 { return 10 << 30 }
	checks, err := a.RunHealthCheck()
	if err != nil {
		t.Fatal(err)
	}
	if len(checks) != 6 {
		t.Fatalf("应 6 项, got %d", len(checks))
	}
	entries, err := a.ListWorkspace("")
	if err != nil {
		t.Fatal(err)
	}
	_ = entries
	if _, err := a.ListWorkspace("../../etc"); err == nil {
		t.Fatal("穿越应报错")
	}
}

func TestOpenWorkspaceFile(t *testing.T) {
	a, _, _ := newTestApp(t)
	dir, _ := a.WS.CreateRunDir("nmap", time.Now())
	os.WriteFile(filepath.Join(dir, "x.txt"), []byte("hi"), 0o644)
	var opened []string
	a.OpenFn = func(name string, args ...string) error {
		opened = append(opened, args...)
		return nil
	}
	rel, _ := filepath.Rel(a.WS.Root, filepath.Join(dir, "x.txt"))
	if err := a.OpenWorkspaceFile(rel); err != nil {
		t.Fatal(err)
	}
	if len(opened) != 1 || opened[0] != filepath.Join(a.WS.Root, rel) {
		t.Fatalf("xdg-open 参数错误: %v", opened)
	}
}
```

说明：为可测试性，`App.Exec` 字段是 `map[string]executor.Runner`；`App` 暴露 `WS`、`Aud`、`DB`、`OpenFn` 字段供测试注入。测试中 `fakeRunner` 实现 `executor.Runner` 接口。

- [ ] **Step 2: 运行测试确认失败**

```bash
go test ./internal/app/ -v
```

预期：FAIL（package 不存在）。

- [ ] **Step 3: 实现 `internal/app/app.go`**

```go
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
	onLine := func(line string) {
		logFile.WriteString(line + "\n")
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
```

- [ ] **Step 4: 实现 `main.go`（替换 wails 模板）**

```go
package main

import (
	"context"
	"embed"
	"fmt"
	"os"
	"path/filepath"

	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
	"github.com/wailsapp/wails/v2/pkg/runtime"

	"blackarch-toolbox/internal/app"
	"blackarch-toolbox/internal/config"
)

//go:embed all:frontend/dist
var assets embed.FS

func main() {
	cfg, err := config.Load()
	if err != nil {
		fmt.Fprintln(os.Stderr, "配置加载失败:", err)
		os.Exit(1)
	}
	dbPath := filepath.Join(dataHome(), "blackarch-toolbox", "toolbox.db")
	application, err := app.New(cfg, dbPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "初始化失败:", err)
		os.Exit(1)
	}
	defer application.DB.Close()
	err = wails.Run(&options.App{
		Title:            "BlackArch ToolBox",
		Width:            1280,
		Height:           800,
		MinWidth:         1000,
		MinHeight:        640,
		BackgroundColour: &options.RGBA{R: 18, G: 18, B: 24, A: 1},
		AssetServer: &assetserver.Options{
			Assets: assets,
		},
		OnStartup: func(ctx context.Context) {
			application.SetEventEmitter(func(event string, data ...any) {
				runtime.EventsEmit(ctx, event, data...)
			})
		},
		Bind: []interface{}{application},
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "运行失败:", err)
		os.Exit(1)
	}
}

func dataHome() string {
	if v := os.Getenv("XDG_DATA_HOME"); v != "" {
		return v
	}
	home, _ := os.UserHomeDir()
	return filepath.Join(home, ".local", "share")
}
```

若 wails 模板生成了根目录 `app.go`（Greet 示例），删除它：

```bash
rm -f /home/karen/blackarch-toolbox/app.go
```

- [ ] **Step 5: 运行测试确认通过**

```bash
go build ./... && go test ./internal/... -cover
```

预期：全部 PASS，`internal/app` 覆盖率 >80%。

- [ ] **Step 6: Commit**

```bash
git add internal/app/ main.go
git -c user.name=karen -c user.email=karen@edith.local commit -m "feat: app 服务层（Bind 方法/后台执行/事件流）与 main.go 接线"
```

## Task 11: 前端界面（Vue 3）

**Files:**
- Create: `frontend/src/components/CategoryTree.vue`、`ToolCard.vue`、`RunDialog.vue`、`LogViewer.vue`、`AuditPanel.vue`、`StatusBar.vue`
- Modify: `frontend/src/App.vue`（整体替换）、`frontend/src/style.css`（整体替换）、`frontend/src/main.js`（保持模板默认）

**Interfaces:**
- Consumes: Wails 自动生成的 bindings（`wailsjs/go/app/App.js` 由 `wails build`/`wails dev` 生成，对应 Task 10 的 App 方法：GetTools/RunTool/DryRun/RunHealthCheck/ListWorkspace/OpenWorkspaceFile/SetPreference）；Events 名 `toolbox:log:{id}`、`toolbox:logend:{id}`
- Produces: 完整中文界面（分类树/卡片网格/运行对话框/日志/审计面板/状态栏）

**注意**：bindings 生成前 `npm run build` 会因 import 缺失报错——本项目开发验证一律走 `wails build`（自动生成 bindings 后构建前端），不要在纯 vite 下跑前端构建。

- [ ] **Step 1: 写 `frontend/src/components/CategoryTree.vue`**

```vue
<template>
  <aside class="sidebar">
    <div class="brand">🛠️ BlackArch ToolBox</div>
    <ul class="tree">
      <li
        v-for="cat in categories"
        :key="cat.key"
        :class="{ active: cat.key === selected }"
        @click="$emit('select', cat.key)"
      >
        <span class="icon">{{ cat.icon }}</span>{{ cat.label }}
      </li>
    </ul>
  </aside>
</template>

<script setup>
defineProps({
  categories: { type: Array, required: true },
  selected: { type: String, required: true },
})
defineEmits(['select'])
</script>
```

- [ ] **Step 2: 写 `frontend/src/components/ToolCard.vue`**

```vue
<template>
  <div class="card">
    <div class="card-head">
      <span class="tool-icon">{{ tool.icon }}</span>
      <span class="tool-name">{{ tool.name }}</span>
      <span class="env-badge" :class="envClass">{{ envLabel }}</span>
    </div>
    <p class="tool-desc">{{ tool.description }}</p>
    <div class="card-foot">
      <span class="use-count">已运行 {{ tool.use_count }} 次</span>
      <button class="run-btn" @click="$emit('run', tool)">▶ 运行</button>
    </div>
  </div>
</template>

<script setup>
import { computed } from 'vue'
const props = defineProps({ tool: { type: Object, required: true } })
defineEmits(['run'])
const envLabel = computed(() => {
  if (props.tool.is_high_risk) return '🔴 VM高危'
  switch (props.tool.default_env) {
    case 'local': return '🟢 本地'
    case 'podman': return '🟡 容器'
    case 'vm': return '🔴 VM'
    default: return '⚪ 未知'
  }
})
const envClass = computed(() => {
  if (props.tool.is_high_risk) return 'badge-vm'
  switch (props.tool.default_env) {
    case 'local': return 'badge-local'
    case 'podman': return 'badge-podman'
    default: return 'badge-vm'
  }
})
</script>
```

- [ ] **Step 3: 写 `frontend/src/components/RunDialog.vue`**

```vue
<template>
  <div class="dialog-mask" @click.self="$emit('close')">
    <div class="dialog">
      <h3>运行 {{ tool.name }}</h3>
      <p class="desc">{{ tool.description }}</p>
      <label>参数</label>
      <input v-model="args" class="args-input" placeholder="例如: -sV 192.168.1.1" @keydown.enter="run" />
      <label>运行环境</label>
      <select v-model="env" @change="refreshPreview">
        <option value="auto">自动（智能路由）</option>
        <option value="local">本地</option>
        <option value="podman">Podman 容器</option>
        <option value="vm">虚拟机</option>
      </select>
      <div v-if="decision" class="decision-box">
        将使用：<b>{{ envName(decision.env) }}</b>（{{ decision.reason }}）
      </div>
      <div v-if="tool.is_high_risk" class="danger-box">
        ⚠️ 该工具为高危工具，建议在 VM 隔离环境运行
      </div>
      <div class="dialog-actions">
        <button class="btn-ghost" @click="$emit('close')">取消</button>
        <button class="btn-primary" @click="run">执行</button>
      </div>
      <p v-if="error" class="error">{{ error }}</p>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { DryRun, RunTool } from '../../wailsjs/go/app/App'

const props = defineProps({ tool: { type: Object, required: true } })
const emit = defineEmits(['close', 'started'])
const args = ref('')
const env = ref('auto')
const decision = ref(null)
const error = ref('')

const envName = (e) => ({ local: '本地', podman: '容器', vm: '虚拟机' }[e] || e)

async function refreshPreview() {
  try {
    decision.value = await DryRun(props.tool.name, args.value, env.value)
  } catch (e) {
    decision.value = null
  }
}
async function run() {
  error.value = ''
  try {
    const res = await RunTool({ tool: props.tool.name, args: args.value, env: env.value })
    emit('started', { ...res, tool: props.tool.name })
    emit('close')
  } catch (e) {
    error.value = String(e)
  }
}
onMounted(refreshPreview)
</script>
```

- [ ] **Step 4: 写 `frontend/src/components/LogViewer.vue`**

```vue
<template>
  <div v-if="exec" class="log-panel">
    <div class="log-head">
      <span>📋 执行日志 — {{ exec.tool }}（{{ envName(exec.env_used) }}）</span>
      <span v-if="finished" class="exit" :class="exitClass">
        退出码: {{ exitCode }}
      </span>
      <button class="btn-ghost" @click="close">关闭</button>
    </div>
    <div ref="body" class="log-body">
      <div v-for="(l, i) in lines" :key="i" class="log-line">{{ l }}</div>
      <div v-if="!finished" class="log-line dim">运行中…</div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, watch, nextTick, onUnmounted } from 'vue'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'

const props = defineProps({ exec: { type: Object, default: null } })
const emit = defineEmits(['close'])
const lines = ref([])
const finished = ref(false)
const exitCode = ref(null)
const body = ref(null)
let boundId = null

const envName = (e) => ({ local: '本地', podman: '容器', vm: '虚拟机' }[e] || e)
const exitClass = computed(() => (exitCode.value === 0 ? 'exit-ok' : 'exit-bad'))

async function bind() {
  if (!props.exec) return
  if (boundId !== null) unbind()
  boundId = props.exec.execution_id
  finished.value = false
  exitCode.value = null
  lines.value = []
  const id = props.exec.execution_id
  await EventsOn(`toolbox:log:${id}`, (line) => {
    if (boundId !== id) return
    lines.value.push(line)
    nextTick(() => { if (body.value) body.value.scrollTop = body.value.scrollHeight })
  })
  await EventsOn(`toolbox:logend:${id}`, (payload) => {
    if (boundId !== id) return
    finished.value = true
    exitCode.value = payload.exit_code ?? -1
  })
}
function unbind() {
  if (boundId === null) return
  EventsOff(`toolbox:log:${boundId}`)
  EventsOff(`toolbox:logend:${boundId}`)
  boundId = null
}
function close() {
  unbind()
  emit('close')
}
watch(() => props.exec, () => bind(), { immediate: true })
onUnmounted(unbind)
</script>
```

- [ ] **Step 5: 写 `frontend/src/components/AuditPanel.vue`**

```vue
<template>
  <div class="dialog-mask" @click.self="$emit('close')">
    <div class="dialog audit">
      <h3>🩺 环境健康审计</h3>
      <ul class="audit-list">
        <li v-for="c in checks" :key="c.name">
          <span class="dot" :class="dotClass(c.status)"></span>
          <span class="check-name">{{ c.name }}</span>
          <span class="check-detail">{{ c.detail }}</span>
        </li>
      </ul>
      <div class="dialog-actions">
        <button class="btn-ghost" @click="$emit('refresh')">🔄 重新检查</button>
        <button class="btn-primary" @click="$emit('close')">关闭</button>
      </div>
    </div>
  </div>
</template>

<script setup>
defineProps({ checks: { type: Array, required: true } })
defineEmits(['close', 'refresh'])
const dotClass = (s) => ({ ok: 'dot-ok', warning: 'dot-warn', error: 'dot-err' }[s] || 'dot-err')
</script>
```

- [ ] **Step 6: 写 `frontend/src/components/StatusBar.vue`**

```vue
<template>
  <footer class="statusbar">
    <span class="health" :class="healthClass">{{ healthIcon }} {{ healthText }}</span>
    <span class="spacer"></span>
    <span>📋 就绪</span>
    <button class="link-btn" @click="$emit('workspace')">📂 产物目录</button>
    <span v-if="lastRun !== null">执行时间: {{ lastRun }}</span>
  </footer>
</template>

<script setup>
import { computed } from 'vue'
const props = defineProps({
  checks: { type: Array, default: () => [] },
  lastRun: { type: String, default: null },
})
defineEmits(['workspace'])
const health = computed(() => {
  if (!props.checks.length) return { level: 'ok', icon: '🟢', text: '全部正常' }
  const errs = props.checks.filter((c) => c.status === 'error').length
  const warns = props.checks.filter((c) => c.status === 'warning').length
  if (errs > 0) return { level: 'error', icon: '🔴', text: '关键服务故障' }
  if (warns > 0) return { level: 'warning', icon: '🟡', text: '容器或 VM 未启动' }
  return { level: 'ok', icon: '🟢', text: '全部正常' }
})
const healthIcon = computed(() => health.value.icon)
const healthText = computed(() => health.value.text)
const healthClass = computed(() => 'health-' + health.value.level)
</script>
```

- [ ] **Step 7: 替换 `frontend/src/App.vue`**

```vue
<template>
  <div class="layout">
    <CategoryTree :categories="categories" :selected="selected" @select="select" />
    <div class="main">
      <div class="search-bar">
        <input v-model="search" class="search" placeholder="🔍 输入工具名..." />
        <button class="link-btn" @click="openAudit">🩺 健康审计</button>
      </div>
      <div class="grid">
        <ToolCard v-for="t in filtered" :key="t.id" :tool="t" @run="openRun" />
      </div>
      <div v-if="filtered.length === 0" class="empty">未找到匹配的工具</div>
    </div>
    <RunDialog v-if="runTool" :tool="runTool" @close="runTool = null" @started="onStarted" />
    <LogViewer :exec="currentExec" @close="currentExec = null" />
    <AuditPanel v-if="showAudit" :checks="checks" @close="showAudit = false" @refresh="refreshHealth" />
    <StatusBar :checks="checks" :last-run="lastRunText" @workspace="openWorkspace" />
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import CategoryTree from './components/CategoryTree.vue'
import ToolCard from './components/ToolCard.vue'
import RunDialog from './components/RunDialog.vue'
import LogViewer from './components/LogViewer.vue'
import AuditPanel from './components/AuditPanel.vue'
import StatusBar from './components/StatusBar.vue'
import { GetTools, RunHealthCheck, OpenWorkspaceFile } from '../wailsjs/go/app/App'

const categories = [
  { key: '', label: '全部', icon: '📁' },
  { key: 'scanner', label: '扫描', icon: '📁' },
  { key: 'cracker', label: '破解', icon: '📁' },
  { key: 'exploitation', label: '利用', icon: '📁' },
  { key: 'forensics', label: '取证', icon: '📁' },
  { key: 'wireless', label: '无线', icon: '📁' },
  { key: 'web', label: 'Web', icon: '📁' },
  { key: 'recon', label: '侦察', icon: '📁' },
  { key: 'stego', label: '隐写', icon: '📁' },
  { key: 'social', label: '社工', icon: '📁' },
  { key: 'automation', label: '自动化', icon: '📁' },
]

const tools = ref([])
const selected = ref('')
const search = ref('')
const runTool = ref(null)
const currentExec = ref(null)
const showAudit = ref(false)
const checks = ref([])
const lastRunText = ref(null)

const filtered = computed(() => {
  const q = search.value.trim().toLowerCase()
  return tools.value.filter((t) => {
    if (selected.value && t.category !== selected.value) return false
    if (!q) return true
    return t.name.toLowerCase().includes(q) || (t.description || '').toLowerCase().includes(q)
  })
})

function select(key) { selected.value = key }

async function refreshTools() {
  try { tools.value = await GetTools('') } catch (e) { console.error(e) }
}
async function refreshHealth() {
  try { checks.value = await RunHealthCheck() } catch (e) { console.error(e) }
}
function openRun(tool) { runTool.value = tool }
function openAudit() { showAudit.value = true }
function onStarted(res) {
  currentExec.value = res
  lastRunText.value = new Date().toLocaleTimeString()
  refreshTools()
  refreshHealth()
}
async function openWorkspace() {
  try { await OpenWorkspaceFile('') } catch (e) { console.error(e) }
}
onMounted(() => { refreshTools(); refreshHealth() })
</script>
```

- [ ] **Step 8: 替换 `frontend/src/style.css`**

```css
:root {
  --bg: #121218; --panel: #1b1b23; --panel2: #22222c; --border: #2e2e3a;
  --text: #e6e6ee; --dim: #9a9ab0; --accent: #4f9dff; --ok: #3ecf8e;
  --warn: #e5b567; --err: #ff6b6b;
}
* { margin: 0; padding: 0; box-sizing: border-box; }
body, html, #app { height: 100%; }
body { background: var(--bg); color: var(--text); font-family: "Noto Sans CJK SC", "Microsoft YaHei", sans-serif; font-size: 14px; }
.layout { display: flex; flex-direction: column; height: 100%; }
.layout > .main { flex: 1; overflow-y: auto; padding: 16px; }
.sidebar { width: 200px; background: var(--panel); border-right: 1px solid var(--border); padding: 12px 0; flex-shrink: 0; }
.layout { flex-direction: row; flex-wrap: wrap; }
.sidebar { height: calc(100% - 40px); overflow-y: auto; }
.brand { font-weight: 700; padding: 8px 16px 16px; color: var(--accent); }
.tree { list-style: none; }
.tree li { padding: 8px 16px; cursor: pointer; color: var(--dim); user-select: none; }
.tree li:hover { background: var(--panel2); color: var(--text); }
.tree li.active { background: var(--panel2); color: var(--accent); border-right: 2px solid var(--accent); }
.tree .icon { margin-right: 8px; }
.search-bar { display: flex; gap: 8px; margin-bottom: 16px; }
.search { flex: 1; background: var(--panel); border: 1px solid var(--border); border-radius: 8px; color: var(--text); padding: 10px 14px; outline: none; }
.search:focus { border-color: var(--accent); }
.grid { display: grid; grid-template-columns: repeat(auto-fill, minmax(240px, 1fr)); gap: 12px; }
.card { background: var(--panel); border: 1px solid var(--border); border-radius: 10px; padding: 14px; display: flex; flex-direction: column; gap: 10px; }
.card:hover { border-color: var(--accent); }
.card-head { display: flex; align-items: center; gap: 8px; }
.tool-icon { font-size: 20px; }
.tool-name { font-weight: 700; flex: 1; }
.env-badge { font-size: 12px; padding: 2px 8px; border-radius: 10px; }
.badge-local { background: rgba(62,207,142,.15); color: var(--ok); }
.badge-podman { background: rgba(229,181,103,.15); color: var(--warn); }
.badge-vm { background: rgba(255,107,107,.15); color: var(--err); }
.tool-desc { color: var(--dim); font-size: 13px; min-height: 36px; }
.card-foot { display: flex; align-items: center; justify-content: space-between; }
.use-count { color: var(--dim); font-size: 12px; }
.run-btn { background: var(--accent); color: #fff; border: none; border-radius: 6px; padding: 6px 14px; cursor: pointer; }
.run-btn:hover { filter: brightness(1.1); }
.dialog-mask { position: fixed; inset: 0; background: rgba(0,0,0,.55); display: flex; align-items: center; justify-content: center; z-index: 10; }
.dialog { background: var(--panel); border: 1px solid var(--border); border-radius: 12px; padding: 20px; width: 480px; display: flex; flex-direction: column; gap: 10px; }
.dialog h3 { color: var(--text); }
.dialog .desc { color: var(--dim); font-size: 13px; }
.dialog label { color: var(--dim); font-size: 12px; margin-top: 6px; }
.dialog input, .dialog select { background: var(--panel2); border: 1px solid var(--border); color: var(--text); border-radius: 6px; padding: 8px 10px; outline: none; }
.decision-box { background: var(--panel2); border-radius: 6px; padding: 10px; font-size: 13px; }
.danger-box { background: rgba(255,107,107,.12); color: var(--err); border-radius: 6px; padding: 10px; font-size: 13px; }
.dialog-actions { display: flex; justify-content: flex-end; gap: 8px; margin-top: 6px; }
.btn-primary { background: var(--accent); color: #fff; border: none; border-radius: 6px; padding: 8px 18px; cursor: pointer; }
.btn-ghost { background: transparent; color: var(--dim); border: 1px solid var(--border); border-radius: 6px; padding: 8px 14px; cursor: pointer; }
.error { color: var(--err); font-size: 13px; }
.log-panel { position: fixed; left: 220px; right: 0; bottom: 40px; height: 220px; background: var(--panel); border-top: 1px solid var(--border); z-index: 9; display: flex; flex-direction: column; }
.log-head { display: flex; align-items: center; gap: 12px; padding: 8px 14px; border-bottom: 1px solid var(--border); }
.log-head .exit { font-size: 12px; }
.exit-ok { color: var(--ok); }
.exit-bad { color: var(--err); }
.log-body { flex: 1; overflow-y: auto; padding: 8px 14px; font-family: "JetBrains Mono", monospace; font-size: 12px; }
.log-line { white-space: pre-wrap; word-break: break-all; }
.dim { color: var(--dim); }
.audit { width: 520px; }
.audit-list { list-style: none; display: flex; flex-direction: column; gap: 8px; max-height: 320px; overflow-y: auto; }
.audit-list li { display: flex; align-items: center; gap: 10px; background: var(--panel2); border-radius: 6px; padding: 8px 12px; }
.dot { width: 10px; height: 10px; border-radius: 50%; flex-shrink: 0; }
.dot-ok { background: var(--ok); }
.dot-warn { background: var(--warn); }
.dot-err { background: var(--err); }
.check-name { font-weight: 600; min-width: 90px; }
.check-detail { color: var(--dim); font-size: 12px; }
.statusbar { height: 40px; background: var(--panel); border-top: 1px solid var(--border); display: flex; align-items: center; gap: 16px; padding: 0 16px; font-size: 13px; color: var(--dim); position: fixed; bottom: 0; left: 0; right: 0; }
.statusbar .spacer { flex: 1; }
.health-ok { color: var(--ok); }
.health-warning { color: var(--warn); }
.health-error { color: var(--err); }
.link-btn { background: transparent; border: none; color: var(--accent); cursor: pointer; font-size: 13px; padding: 4px 8px; }
.empty { color: var(--dim); text-align: center; padding: 60px 0; }
```

- [ ] **Step 9: 构建验证**

```bash
cd /home/karen/blackarch-toolbox && ~/go/bin/wails build
ls -la build/bin/blackarch-toolbox
```

预期：构建成功（bindings 自动生成 + npm build 内嵌）。若报 import 找不到 `wailsjs`，检查 `wails.json` 的 frontend 配置与 `frontend/src` 相对路径（`../wailsjs/...` 从 `src/` 起、`../../wailsjs/...` 从 `src/components/` 起）。

- [ ] **Step 10: 冒烟运行**

```bash
timeout 8 ./build/bin/blackarch-toolbox & sleep 5; ls -la ~/.local/share/blackarch-toolbox/toolbox.db 2>&1; ls ~/BlackArch_Workspace 2>&1
```

预期：进程存活至 timeout 结束（窗口正常显示），DB 已创建。若无显示环境报错，跳过并告知用户手动验证。

- [ ] **Step 11: Commit**

```bash
git add frontend/
git -c user.name=karen -c user.email=karen@edith.local commit -m "feat: 中文深色前端（分类树/卡片/运行对话框/日志/审计/状态栏）"
```

---

## Task 12: 导入脚本、README 与最终打包

**Files:**
- Create: `scripts/import_tools.sh`、`README.md`
- Modify: 无

**Interfaces:**
- Consumes: `internal/db/tools.json` 格式
- Produces: 可交付项目（README 说明构建/运行/配置）

- [ ] **Step 1: 写 `scripts/import_tools.sh`**

```bash
#!/usr/bin/env bash
# 从本机 BlackArch 仓库同步工具名到 tools.json（保留已有描述/图标/分类）
# 用法: ./scripts/import_tools.sh [blackarch元数据目录] [输出文件]
set -euo pipefail

SRC="${1:-/usr/share/blackarch}"
OUT="${2:-$(dirname "$0")/../internal/db/tools.json}"

if [[ ! -d "$SRC" ]]; then
  echo "未找到 BlackArch 元数据目录: $SRC" >&2
  echo "请先安装 blackarch 相关包组，或手动指定路径" >&2
  exit 1
fi

python3 - "$SRC" "$OUT" <<'PY'
import json, os, sys
src, out = sys.argv[1], sys.argv[2]
existing = {}
if os.path.exists(out):
    try:
        for t in json.load(open(out, encoding="utf-8")):
            existing[t["name"]] = t
    except json.JSONDecodeError:
        pass
tools = []
for cat in sorted(os.listdir(src)):
    d = os.path.join(src, cat)
    if not os.path.isdir(d):
        continue
    for name in sorted(os.listdir(d)):
        t = existing.get(name, {})
        t.setdefault("name", name)
        t.setdefault("category", cat)
        t.setdefault("description", "")
        t.setdefault("default_env", "local")
        t.setdefault("is_high_risk", False)
        t.setdefault("icon", "🛠️")
        tools.append(t)
with open(out, "w", encoding="utf-8") as f:
    json.dump(tools, f, ensure_ascii=False, indent=2)
print(f"已写入 {len(tools)} 个工具到 {out}")
PY

echo "提示: 重新构建应用以内嵌新的 tools.json"
```

- [ ] **Step 2: 写 `README.md`**

```markdown
# BlackArch ToolBox

Arch Linux 桌面应用：管理 BlackArch 工具，提供分类浏览、一键运行（本地/Podman/VM 智能路由）、环境健康审计与产物归档。

## 构建

要求：Go ≥1.21、Node ≥18、webkit2gtk-4.1、gtk3、wails v2.14.0

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0
wails build        # 产物: build/bin/blackarch-toolbox
```

## 运行

```bash
./build/bin/blackarch-toolbox
```

数据与产物：
- SQLite: `~/.local/share/blackarch-toolbox/toolbox.db`
- 产物: `~/BlackArch_Workspace/<工具>/<时间戳>/`（含 output.log 与 .metadata.json）
- 配置: `~/.config/blackarch-toolbox/config.yaml`（可选）

```yaml
vm:
  host: "192.168.122.2"
  user: "blackarch"
  port: 22
  name: "blackarch"
workspace:
  path: "~/BlackArch_Workspace"
podman:
  container: "blackarch-tools"
```

环境变量可覆盖：`TOOLBOX_VM_HOST` `TOOLBOX_VM_USER` `TOOLBOX_VM_PORT` `TOOLBOX_VM_NAME` `TOOLBOX_WORKSPACE` `TOOLBOX_PODMAN_CONTAINER`

## 智能路由

偏好设置 → 高危列表(强制 VM) → 依赖冲突(容器) → 本机存在(本地) → 兜底 VM。

## 测试

```bash
go test ./internal/... -cover
```

## 同步工具列表

```bash
./scripts/import_tools.sh /usr/share/blackarch internal/db/tools.json
```
```

- [ ] **Step 3: 全量验证（测试 + 构建 + 覆盖率）**

```bash
cd /home/karen/blackarch-toolbox
go vet ./...
go test ./internal/... -cover
~/go/bin/wails build
```

预期：vet 无输出；所有包 PASS 且覆盖率 >80%；构建产出单文件。

- [ ] **Step 4: Commit**

```bash
git add scripts/ README.md
git -c user.name=karen -c user.email=karen@edith.local commit -m "docs: 导入脚本与 README；最终验证通过"
```

---

## 完成定义

- [ ] 所有 12 个任务完成，`git log` 每个任务至少一个 commit
- [ ] `go test ./internal/... -cover`：每包 >80%
- [ ] `wails build` 产出 `build/bin/blackarch-toolbox`
- [ ] 冒烟运行：窗口启动、`~/BlackArch_Workspace` 与 toolbox.db 生成
- [ ] README 可复现构建

