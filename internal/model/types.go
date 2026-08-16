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
