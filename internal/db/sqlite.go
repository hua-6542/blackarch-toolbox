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
