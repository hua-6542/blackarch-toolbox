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
