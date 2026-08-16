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
	if p == "~" {
		if home, err := os.UserHomeDir(); err == nil {
			return home
		}
		return p
	}
	if len(p) > 1 && p[:2] == "~/" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, p[2:])
		}
	}
	return p
}
