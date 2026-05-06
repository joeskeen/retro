package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type Transport struct {
	remoteURL string
	localPath string
}

func New(remoteURL, localPath string) *Transport {
	remoteURL = convertToGitProtocol(remoteURL)
	return &Transport{
		remoteURL: remoteURL,
		localPath: localPath,
	}
}

func convertToGitProtocol(url string) string {
	if strings.HasPrefix(url, "https://github.com/") {
		url = "git@github.com:" + strings.TrimPrefix(url, "https://")
	} else if strings.HasPrefix(url, "https://") {
		url = "git://" + strings.TrimPrefix(url, "https://")
	}
	return url
}

func (t *Transport) CloneOrOpen() error {
	if _, err := os.Stat(t.localPath); os.IsNotExist(err) {
		cmd := exec.Command("git", "clone", t.remoteURL, t.localPath)
		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git clone failed: %s", string(out))
		}
	}
	return nil
}

func (t *Transport) Push() error {
	cmd := exec.Command("git", "-C", t.localPath, "add", "-A")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s", string(out))
	}

	cmd = exec.Command("git", "-C", t.localPath, "status", "--porcelain")
	out, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("git status failed: %w", err)
	}
	if len(strings.TrimSpace(string(out))) == 0 {
		return nil
	}

	cmd = exec.Command("git", "-C", t.localPath, "commit", "-m", "Update layers and manifests")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git commit failed: %s", string(out))
	}

	cmd = exec.Command("git", "-C", t.localPath, "push")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git push failed: %s", string(out))
	}
	return nil
}

func (t *Transport) Pull() error {
	cmd := exec.Command("git", "-C", t.localPath, "pull", "--rebase")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s", string(out))
	}
	return nil
}

func (t *Transport) AddLayer(sha256 string, data []byte) error {
	layerPath := filepath.Join(t.localPath, "layers", sha256+".layer")
	if err := os.MkdirAll(filepath.Dir(layerPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(layerPath, data, 0644)
}

func (t *Transport) GetLayer(sha256 string) ([]byte, error) {
	layerPath := filepath.Join(t.localPath, "layers", sha256+".layer")
	return os.ReadFile(layerPath)
}

func (t *Transport) AddManifest(name, tag string, data []byte) error {
	manifestPath := filepath.Join(t.localPath, "manifests", name, tag, "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return err
	}
	return os.WriteFile(manifestPath, data, 0644)
}

func (t *Transport) GetManifest(name, tag string) ([]byte, error) {
	manifestPath := filepath.Join(t.localPath, "manifests", name, tag, "manifest.json")
	return os.ReadFile(manifestPath)
}

func (t *Transport) ListImages() ([]string, error) {
	manifestDir := filepath.Join(t.localPath, "manifests")
	entries, err := os.ReadDir(manifestDir)
	if err != nil {
		return nil, err
	}

	var images []string
	for _, entry := range entries {
		if entry.IsDir() {
			images = append(images, entry.Name())
		}
	}
	return images, nil
}
