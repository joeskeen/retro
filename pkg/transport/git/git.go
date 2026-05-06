package git

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"os/user"
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
		url = "git@github.com:" + strings.TrimPrefix(url, "https://github.com/")
	} else if strings.HasPrefix(url, "https://gitlab.com/") {
		url = "git@gitlab.com:" + strings.TrimPrefix(url, "https://gitlab.com/")
	} else if strings.HasPrefix(url, "https://") {
		url = "git://" + strings.TrimPrefix(url, "https://")
	}
	return url
}

func (t *Transport) CloneOrOpen() error {
	gitDir := filepath.Join(t.localPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		if err := os.MkdirAll(t.localPath, 0755); err != nil {
			return fmt.Errorf("failed to create registry directory: %w", err)
		}
		cmd := exec.Command("git", "clone", t.remoteURL, t.localPath)

		if keyName := findDeployKey(t.remoteURL); keyName != "" {
			cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND=ssh -i ~/.ssh/"+keyName+" -o StrictHostKeyChecking=no")
			fmt.Printf("[git transport] Using SSH key: %s\n", keyName)
		} else {
			fmt.Printf("[git transport] No deploy key found, using default SSH\n")
		}

		out, err := cmd.CombinedOutput()
		if err != nil {
			return fmt.Errorf("git clone failed: %s", string(out))
		}
	}

	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return fmt.Errorf("not a git repository at %s", t.localPath)
	}
	return nil
}

func findDeployKey(url string) string {
	usr, _ := user.Current()
	sshPath := filepath.Join(usr.HomeDir, ".ssh")

	entries, _ := os.ReadDir(sshPath)
	var keys []string
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if strings.Contains(name, "deploy-key") || name == "id_rsa" || name == "id_ed25519" || strings.Contains(name, "_rsa") {
			keys = append(keys, name)
		}
	}
	fmt.Printf("[git transport] Available SSH keys: %v\n", keys)
	if len(keys) > 0 {
		return keys[0]
	}
	return ""
}

func (t *Transport) ensureLFS() error {
	fmt.Println("[git transport] Setting up Git LFS...")

	cmd := exec.Command("git", "-C", t.localPath, "lfs", "install")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git lfs install failed: %s", string(out))
	}
	fmt.Println("[git transport] LFS install complete")

	lfsTrackCmd := exec.Command("git", "-C", t.localPath, "lfs", "track", "*.layer")
	if out, err := lfsTrackCmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git lfs track failed: %s", string(out))
	}
	fmt.Println("[git transport] LFS track '*.layer' configured")

	attrsPath := filepath.Join(t.localPath, ".gitattributes")
	attrsContent := "*.layer filter=lfs diff=lfs merge=lfs -text\n"
	if err := os.WriteFile(attrsPath, []byte(attrsContent), 0644); err != nil {
		return fmt.Errorf("failed to write .gitattributes: %w", err)
	}
	fmt.Println("[git transport] .gitattributes written")

	cmd = exec.Command("git", "-C", t.localPath, "add", ".gitattributes")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add .gitattributes failed: %s", string(out))
	}

	cmd = exec.Command("git", "-C", t.localPath, "commit", "-m", "Configure Git LFS for .layer files")
	if out, err := cmd.CombinedOutput(); err != nil {
		outStr := string(out)
		if !strings.Contains(outStr, "nothing to commit") && !strings.Contains(outStr, "no changes added to commit") {
			return fmt.Errorf("git commit .gitattributes failed: %s", outStr)
		}
	}
	fmt.Println("[git transport] .gitattributes committed")

	return nil
}

func (t *Transport) migrateLFS() error {
	layersPath := filepath.Join(t.localPath, "layers")
	entries, err := os.ReadDir(layersPath)
	if err != nil {
		return nil
	}

	converted := 0
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".layer") {
			continue
		}
		layerPath := filepath.Join(layersPath, entry.Name())
		convertedNow, err := t.convertToLFS(layerPath)
		if err != nil {
			return fmt.Errorf("failed to convert %s to LFS: %w", entry.Name(), err)
		}
		if convertedNow {
			converted++
		}
	}
	fmt.Printf("[git transport] Converted %d layers to LFS pointers\n", converted)
	return nil
}

func (t *Transport) convertToLFS(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return false, err
	}

	if strings.HasPrefix(string(data), "version https://git-lfs.github.com/spec/v1") {
		return false, nil
	}

	oid := sha256Hash(data)
	size := len(data)

	pointer := fmt.Sprintf("version https://git-lfs.github.com/spec/v1\noid sha256:%s\nsize %d\n", oid, size)
	err = os.WriteFile(path, []byte(pointer), 0644)
	return true, err
}

func sha256Hash(data []byte) string {
	h := sha256.New()
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil))
}

func (t *Transport) Push() error {
	if err := t.ensureLFS(); err != nil {
		return err
	}

	if err := t.migrateLFS(); err != nil {
		return err
	}

	cmd := exec.Command("git", "-C", t.localPath, "add", "-A")
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("git add failed: %s", string(out))
	}
	fmt.Printf("[git transport] Staged changes\n")

	cmd = exec.Command("git", "-C", t.localPath, "status")
	statusOut, statusErr := cmd.Output()
	if statusErr != nil {
		return fmt.Errorf("git status failed: %w", statusErr)
	}
	fmt.Printf("[git transport] Status:\n%s\n", string(statusOut))
	statusStr := string(statusOut)
	needsPush := false
	if strings.Contains(statusStr, "Your branch is ahead") {
		needsPush = true
	}
	if strings.Contains(statusStr, "nothing to commit") || strings.Contains(statusStr, "no changes added to commit") {
		fmt.Printf("[git transport] Nothing to commit\n")
		if !needsPush {
			return nil
		}
		fmt.Printf("[git transport] There are unpushed commits, forcing push anyway\n")
	} else {
		cmd = exec.Command("git", "-C", t.localPath, "commit", "-m", "Update layers and manifests")
		if out, err := cmd.CombinedOutput(); err != nil {
			return fmt.Errorf("git commit failed: %s", string(out))
		}
	}

	cmd = exec.Command("git", "-C", t.localPath, "push", "--force")
	if keyName := findDeployKey(t.remoteURL); keyName != "" {
		cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND=ssh -i ~/.ssh/"+keyName+" -o StrictHostKeyChecking=no")
	}
	pushOut, pushErr := cmd.CombinedOutput()
	if pushErr != nil {
		return fmt.Errorf("git push failed: %s", string(pushOut))
	}
	if strings.Contains(string(pushOut), "error") || strings.Contains(string(pushOut), "rejected") {
		return fmt.Errorf("git push failed: %s", string(pushOut))
	}
	fmt.Printf("[git transport] Push complete\n")
	return nil
}

func (t *Transport) Pull() error {
	cmd := exec.Command("git", "-C", t.localPath, "pull", "--rebase")
	if keyName := findDeployKey(t.remoteURL); keyName != "" {
		cmd.Env = append(os.Environ(), "GIT_SSH_COMMAND=ssh -i ~/.ssh/"+keyName+" -o StrictHostKeyChecking=no")
	}
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git pull failed: %s", string(out))
	}
	return nil
}

func (t *Transport) AddLayer(sha256 string, data []byte) error {
	layerPath := filepath.Join(t.localPath, "layers", sha256+".layer")
	fmt.Printf("[git transport] Adding layer %s (size: %d bytes)\n", sha256[:8], len(data))
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
	fmt.Printf("[git transport] Adding manifest at %s\n", manifestPath)
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return err
	}
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return err
	}
	fmt.Printf("[git transport] Manifest written, size: %d bytes\n", len(data))
	return nil
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

func (t *Transport) FindAvailableTag(imageName string) string {
	manifestPath := filepath.Join(t.localPath, "manifests", imageName)
	entries, err := os.ReadDir(manifestPath)
	if err != nil || len(entries) == 0 {
		return ""
	}
	for _, entry := range entries {
		if entry.IsDir() {
			return entry.Name()
		}
	}
	return ""
}

func (t *Transport) Prune() (int, error) {
	referencedLayers := make(map[string]bool)
	manifestDir := filepath.Join(t.localPath, "manifests")
	manifestEntries, err := os.ReadDir(manifestDir)
	if err != nil {
		return 0, fmt.Errorf("failed to read manifests dir: %w", err)
	}

	for _, imageEntry := range manifestEntries {
		if !imageEntry.IsDir() {
			continue
		}
		imagePath := filepath.Join(manifestDir, imageEntry.Name())
		tagEntries, err := os.ReadDir(imagePath)
		if err != nil {
			continue
		}
		for _, tagEntry := range tagEntries {
			if !tagEntry.IsDir() {
				continue
			}
			manifestPath := filepath.Join(imagePath, tagEntry.Name(), "manifest.json")
			data, err := os.ReadFile(manifestPath)
			if err != nil {
				continue
			}
			var m struct {
				Layers []string `json:"layers"`
			}
			if err := json.Unmarshal(data, &m); err != nil {
				continue
			}
			for _, layer := range m.Layers {
				referencedLayers[layer] = true
			}
		}
	}

	layersPath := filepath.Join(t.localPath, "layers")
	layerEntries, err := os.ReadDir(layersPath)
	if err != nil {
		return 0, fmt.Errorf("failed to read layers dir: %w", err)
	}

	pruned := 0
	for _, entry := range layerEntries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".layer") {
			continue
		}
		sha := strings.TrimSuffix(entry.Name(), ".layer")
		if !referencedLayers[sha] {
			layerPath := filepath.Join(layersPath, entry.Name())
			if err := os.Remove(layerPath); err != nil {
				fmt.Printf("Warning: failed to remove layer %s: %v\n", entry.Name(), err)
				continue
			}
			pruned++
		}
	}

	return pruned, nil
}
