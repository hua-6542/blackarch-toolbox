# BlackArch ToolBox

Arch Linux 桌面应用：管理 BlackArch 工具，提供分类浏览、一键运行（本地/Podman/VM 智能路由）、环境健康审计与产物归档。

## 构建

要求：Go ≥1.21、Node ≥18、webkit2gtk-4.1、gtk3、wails v2.14.0

```bash
go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0
wails build        # 产物: build/bin/blackarch-toolbox
```

## 新机器从零重建

```bash
git clone https://github.com/hua-6542/blackarch-toolbox.git
cd blackarch-toolbox
sudo pacman -S --needed webkit2gtk-4.1 gtk3   # 系统依赖
go install github.com/wailsapp/wails/v2/cmd/wails@v2.14.0
export PATH=$PATH:~/go/bin
wails build
./build/bin/blackarch-toolbox
```

注意：

- 首次启动自动建库（`~/.local/share/blackarch-toolbox/toolbox.db`）并导入内置的 131 个工具
- VM/容器连接走 `~/.config/blackarch-toolbox/config.yaml`（见下）；开发机实际容器名为 `blackarch`，示例默认值为 `blackarch-tools`
- VM 执行依赖 SSH 免密：先 `ssh-copy-id blackarch@192.168.122.2`，应用强制 `BatchMode=yes`，未配置免密会直接报错

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
