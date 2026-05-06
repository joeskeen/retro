package commands

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"retrogame/pkg/config"
	"retrogame/pkg/transport/git"
)

func RunClone() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: retro clone <git-url> [local-name]")
	}

	gitURL := os.Args[2]
	localName := ""

	if len(os.Args) >= 4 {
		localName = os.Args[3]
	} else {
		parts := strings.Split(gitURL, "/")
		lastPart := parts[len(parts)-1]
		localName = strings.TrimSuffix(lastPart, ".git")
	}

	cfgPath := getConfigPath()
	cfg := &config.Config{}

	existingCfg, err := config.Load(cfgPath)
	if err == nil {
		cfg = existingCfg
	}

	for _, r := range cfg.Remotes {
		if r.Name == localName {
			return fmt.Errorf("remote %q already exists", localName)
		}
	}

	gitPath := filepath.Join(getRegistryPath(), "git", localName)
	t := git.New(gitURL, gitPath)

	if err := t.CloneOrOpen(); err != nil {
		return fmt.Errorf("clone failed: %w", err)
	}

	cfg.Remotes = append(cfg.Remotes, config.Remote{Name: localName, URL: gitURL})
	if cfg.DefaultRemote == "" {
		cfg.DefaultRemote = localName
	}

	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Cloned %s as '%s'\n", gitURL, localName)
	fmt.Printf("Run 'retro images' to see available games\n")
	return nil
}
