# BlackArch ToolBox 设计文档

- 日期：2026-08-16
- 状态：已确认（用户逐节认可）
- 需求来源：`~/Desktop/新建Markdown.md`

## 1. 项目定位

Arch Linux 桌面应用，用于管理 BlackArch 工具：分类浏览、一键运行（智能路由 本地/Podman/VM）、环境健康审计、产物统一归档。

## 2. 技术选型（已确认）

| 层级 | 技术 |
|---|---|
| GUI 框架 | Wails **v2.14.0**（稳定版；本机 webkit2gtk-4.1 2.52.4 兼容） |
| 后端 | Go（标准库为主，Go 1.26） |
| 数据库 | SQLite（modernc.org/sqlite，无 CGO） |
| 前端 | Vue 3 + Vite，手写深色 CSS，无 UI 组件库 |
| 进程管理 | os/exec（podman / ssh / systemctl） |
| 界面语言 | 中文 |

- 项目位置：`/home/karen/blackarch-toolbox`
- 测试策略：全面 TDD，每个包有单测，覆盖率 >80%
- 产物：`wails build` 生成单文件 `build/bin/blackarch-toolbox`

## 3. 总体架构与数据流（方案 A：Wails 原生 Bind + Events）

不使用内嵌 HTTP/SSE。Go 侧接口注册为 `runtime.Bind`，前端直接调用；执行日志经 Wails Events 推送（等价 SSE）。

```
┌───────────────────────────────────────────────┐
│  main.go (Wails 入口 + Bind 注册)              │
│  ┌─────────┬──────────┬──────────┬─────────┐  │
│  │ decision │ executor │ workspace │ audit  │  │
│  │ (路由引擎)│ local/   │ (产物管理) │ (健康) │  │
│  │          │ podman/vm│          │        │  │
│  └────┬─────┴────┬─────┴────┬─────┴────┬────┘  │
│       └──────────┴──── db (SQLite) ────┘       │
└───────────────────────────────────────────────┘
      Bind 函数              Events(日志流)
┌───────────────────────────────────────────────┐
│  frontend: Vue3 + Vite + 手写CSS(深色主题)      │
│  App / CategoryTree / ToolCard / RunDialog /   │
│  LogViewer / AuditPanel / StatusBar            │
└───────────────────────────────────────────────┘
```

**运行数据流**：点击【运行】→ 前端 `RunTool(tool, args, env)` → 决策引擎（偏好→高危→依赖冲突→本地存在→兜底VM）→ 执行器后台 goroutine 跑 `os/exec` → 每行 stdout/stderr 经 Events 推送 → 结束后产物写入 `~/BlackArch_Workspace/<tool>/<时间戳>/` 并写 `.metadata.json` → executions 表落库 → 状态栏刷新。

## 4. 后端模块接口（Bind 函数，对应需求文档第 9 节）

| 需求接口 | Wails Bind 实现 |
|---|---|
| GET /api/tools | `GetTools(category string) ([]Tool, error)`，空 category 返回全部 |
| POST /api/tools/run | `RunTool(req RunRequest) (RunResult, error)`，执行在后台 goroutine |
| GET /api/executions/{id}/log | Events：`log:line:{id}` + `log:end:{id}` |
| GET /api/health | `RunHealthCheck() ([]HealthCheck, error)`，并发执行 |
| GET /api/workspace | `ListWorkspace(path string) ([]FileEntry, error)` |
| GET /api/workspace/download | `OpenWorkspaceFile(path string) error`（xdg-open） |
| POST /api/preferences | `SetPreference(tool string, env string) error` |
| 额外：决策预览 | `DryRun(tool, args, env) (Decision, error)`，不执行，供 RunDialog 显示 |

### 4.1 核心类型（internal/model/types.go）

```go
type Tool struct {
    ID          int      `json:"id"`
    Name        string   `json:"name"`
    Category    string   `json:"category"`
    Description string   `json:"description"`
    DefaultEnv  string   `json:"default_env"`
    IsHighRisk  bool     `json:"is_high_risk"`
    Dependencies []string `json:"dependencies,omitempty"`
    Icon        string   `json:"icon"`
    UseCount    int      `json:"use_count"`
}

type RunRequest struct {
    Tool string `json:"tool"`
    Args string `json:"args"`
    Env  string `json:"env"` // "auto"|"local"|"podman"|"vm"
}

type RunResult struct {
    ExecutionID int64  `json:"execution_id"`
    EnvUsed     string `json:"env_used"`
    WorkDir     string `json:"work_dir"`
}

type Decision struct {
    Env      string `json:"env"`      // local|podman|vm
    Reason   string `json:"reason"`
    Priority int    `json:"priority"` // 1=偏好 2=高危 3=依赖冲突 4=本地存在 5=兜底
}

type HealthCheck struct {
    Name   string `json:"name"`
    Status string `json:"status"` // ok|warning|error
    Detail string `json:"detail"`
}
```

## 5. 决策引擎（internal/decision）

决策顺序（优先级 1-5）：
1. 用户偏好（preferences 表）→ 使用偏好
2. 工具在高危列表 → vm
3. 工具依赖声明冲突（如 python2 在 Arch 已移除）→ podman
4. 工具在 `/usr/bin/` 存在 → local
5. 兜底 → vm（最安全）

高危列表硬编码 Go 常量：

```go
var HighRiskTools = []string{
    "metasploit", "volatility", "reaver", "bully",
    "bettercap", "aircrack-ng", "mdk3", "mdk4",
}
```

## 6. 执行器（internal/executor）

统一接口：

```go
type Runner interface {
    Run(ctx context.Context, tool, args, workDir string, onLine func(line string)) (int, error)
}
```

- **local.go**：直接 `os/exec`，命令经 shellquote 拆分
- **podman.go**：`podman exec -i <container> /bin/bash -c "cd workDir && tool args"`（rootless，无 sudo；容器未运行时先 `podman start`）
- **vm.go**：`ssh -o BatchMode=yes -o StrictHostKeyChecking=no <user>@<host> "mkdir -p workDir && cd workDir && tool args"`（VM 上同步创建同路径 workdir）

## 7. 产物管理器（internal/workspace）

- 根路径 `~/BlackArch_Workspace`（可由 config 覆盖）
- 每次执行：`<root>/<tool>/2006-01-02_15-04-05/`，写 `.metadata.json`：

```json
{
  "tool": "nmap",
  "environment": "local",
  "executed_at": "2026-08-16T14:30:22+08:00",
  "command": "nmap -sV 192.168.1.1",
  "exit_code": 0,
  "output_dir": "/home/user/BlackArch_Workspace/nmap/2026-08-16_14-30-22"
}
```

- 安全：`filepath.Clean` 后校验前缀必须落在 workspace 根内（防路径穿越）

## 8. 健康审计（internal/audit）

并发执行，单项 5 秒超时，共 6 项：

| 检查项 | 命令 | 判定 |
|---|---|---|
| Podman 服务 | `podman info` | 退出 0=ok，否则 error |
| 目标容器 | `podman ps -q -f name=<container>` | 运行=ok；镜像在但未运行=warning；无=error |
| libvirtd | `systemctl is-active libvirtd` | active=ok，否则 warning |
| VM 状态 | `virsh list --state-running` | 目标 VM 运行=ok；存在但关机=warning；无=error |
| 产物目录 | `os.Stat` + 写测试 | 可写=ok，否则 error |
| 磁盘空间 | `syscall.Statfs` | >2GB=ok，否则 error |

## 9. 数据库（internal/db，SQLite）

- 驱动：`modernc.org/sqlite`（无 CGO）
- 文件：`~/.local/share/blackarch-toolbox/toolbox.db`（懒加载）
- 表结构（需求文档第 8 节）：

```sql
CREATE TABLE tools (
    id INTEGER PRIMARY KEY,
    name TEXT UNIQUE NOT NULL,
    category TEXT,
    description TEXT,
    default_env TEXT DEFAULT 'local',
    is_high_risk INTEGER DEFAULT 0,
    use_count INTEGER DEFAULT 0,
    icon TEXT
);

CREATE TABLE executions (
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

CREATE TABLE preferences (
    tool_id INTEGER PRIMARY KEY,
    preferred_env TEXT,
    updated_at DATETIME,
    FOREIGN KEY(tool_id) REFERENCES tools(id)
);
```

- 种子数据：内置 `tools.json`（≥100 个 BlackArch 常用工具，中文描述 + emoji 图标，分类 scanner/cracker/exploitation/forensics/wireless/web/recon/stego/social/automation），首次启动导入
- `scripts/import_tools.sh`：从真机 `/usr/share/blackarch/` 同步（备用）

## 10. 配置（internal/config）

- 文件：`~/.config/blackarch-toolbox/config.yaml`（可选），环境变量可覆盖，无文件时用默认值：

```yaml
vm:
  host: "192.168.122.2"
  user: "blackarch"
  port: 22
workspace:
  path: "~/BlackArch_Workspace"
podman:
  container: "blackarch-tools"
```

> 注意：开发机（EDITH）上实际容器名为 `blackarch`（当前 Exited 状态）。默认值保持需求文档的 `blackarch-tools`，开发机通过 config.yaml 覆盖为 `blackarch` 进行验证。

## 11. 前端界面（Vue 3 + Vite，中文，深色主题）

- **App.vue**：三栏布局（左分类树 / 右卡片网格 / 底部状态栏）
- **CategoryTree.vue**：分类筛选（全部/扫描/破解/利用/取证/无线/Web/隐写/社工/自动化）
- **ToolCard.vue**：emoji 图标、名称、中文描述、环境徽标（🟢本地/🟡容器/🔴VM高危）、使用次数、【▶运行】
- **RunDialog.vue**：参数输入 + 环境选择（自动/本地/容器/VM）+ DryRun 决策预览；高危工具红色警示条
- **LogViewer.vue**：实时日志（Events 订阅、ANSI 颜色、自动滚动、退出码）
- **AuditPanel.vue**：6 项检查结果列表
- **StatusBar.vue**：健康指示灯（🟢/🟡/🔴）+ 就绪状态 + 产物目录 + 上次执行耗时

## 12. 安全约束

- 参数用 shellquote 拆分（`mvdan.cc/sh/v3/shell`），杜绝 shell 注入拼接
- SSH 强制 `BatchMode=yes`，免密失败直接报错
- 全程无 sudo；SQLite prepared statement；Bind 校验 tool 名存在性
- 产物访问防路径穿越

## 13. 性能目标

- 单文件产物 `build/bin/blackarch-toolbox`
- 启动 <500ms（SQLite 懒加载、前端产物内嵌）
- 内存 <50MB

## 14. 项目结构

```
blackarch-toolbox/
├── main.go                 # Wails 入口 + Bind 注册
├── frontend/               # Vue3 + Vite
│   └── src/
│       ├── components/
│       │   ├── ToolCard.vue
│       │   ├── CategoryTree.vue
│       │   ├── RunDialog.vue
│       │   ├── LogViewer.vue
│       │   ├── AuditPanel.vue
│       │   └── StatusBar.vue
│       └── App.vue
├── internal/
│   ├── decision/engine.go
│   ├── executor/{local,podman,vm}.go
│   ├── workspace/manager.go
│   ├── audit/health.go
│   ├── db/sqlite.go
│   ├── config/config.go
│   └── model/types.go
├── tools.json              # 内置种子数据
├── scripts/import_tools.sh
├── wails.json
└── go.mod
```

## 15. 交付顺序

1. 项目初始化（wails init，目录结构）
2. model/types.go + config
3. db/sqlite.go（含种子导入）
4. decision/engine.go
5. executor（local/podman/vm）
6. workspace/manager.go
7. audit/health.go
8. 前端界面（App.vue + 6 组件）
9. main.go Bind 注册
10. 打包验证（wails build）

每个模块 TDD：先写测试再实现，覆盖率 >80%。
