package config

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"
)

type Remote struct {
	Name string `toml:"name"`
	URL  string `toml:"url"`
}

type Config struct {
	DefaultRemote string   `toml:"default-remote"`
	Remotes       []Remote `toml:"remote"`
}

func DefaultPath() (string, error) {
	usr, err := user.Current()
	if err != nil {
		return "", err
	}
	return filepath.Join(usr.HomeDir, ".retro", "config.toml"), nil
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	cfg := &Config{}
	if err := parseConfig(string(data), cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func Save(cfg *Config, path string) error {
	data, err := serializeConfig(cfg)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}
	return os.WriteFile(path, []byte(data), 0644)
}

func (c *Config) GetRemote(name string) (*Remote, error) {
	for _, r := range c.Remotes {
		if r.Name == name {
			return &r, nil
		}
	}
	return nil, fmt.Errorf("remote not found: %s", name)
}

func (c *Config) GetDefaultRemote() (*Remote, error) {
	if c.DefaultRemote == "" {
		return nil, fmt.Errorf("no default remote configured")
	}
	return c.GetRemote(c.DefaultRemote)
}

func (c *Config) ListRemotes() []Remote {
	return c.Remotes
}

func parseConfig(data string, cfg *Config) error {
	lines := strings.Split(data, "\n")
	var currentRemote *Remote

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		if strings.HasPrefix(line, "[") && strings.HasSuffix(line, "]") {
			section := strings.Trim(line, "[]")
			if strings.HasPrefix(section, "remote ") {
				remoteName := strings.TrimPrefix(section, "remote ")
				remoteName = strings.Trim(remoteName, "\"")
				if currentRemote != nil {
					cfg.Remotes = append(cfg.Remotes, *currentRemote)
				}
				currentRemote = &Remote{Name: remoteName}
			} else if section == "default" {
				// default section handled differently
			}
		} else if strings.HasPrefix(line, "url") {
			if currentRemote != nil {
				parts := strings.SplitN(line, "=", 2)
				if len(parts) == 2 {
					currentRemote.URL = strings.Trim(strings.Trim(parts[1], " \""), "\"")
				}
			}
		} else if strings.HasPrefix(line, "default-remote") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				cfg.DefaultRemote = strings.Trim(parts[1], " \"")
			}
		}
	}

	if currentRemote != nil {
		cfg.Remotes = append(cfg.Remotes, *currentRemote)
	}

	return nil
}

func serializeConfig(cfg *Config) (string, error) {
	var sb strings.Builder

	if cfg.DefaultRemote != "" {
		sb.WriteString(fmt.Sprintf("default-remote = %q\n\n", cfg.DefaultRemote))
	}

	for _, r := range cfg.Remotes {
		sb.WriteString(fmt.Sprintf("[remote %q]\nurl = %q\n", r.Name, r.URL))
	}

	return sb.String(), nil
}
