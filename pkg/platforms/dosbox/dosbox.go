package dosbox

import (
	"archive/tar"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"retrogame/pkg/layers"
	"retrogame/pkg/manifest"
)

type DOSBoxPlatform struct {
	registryPath string
	layerManager *layers.Manager
	workDir      string
}

func New(registryPath string) *DOSBoxPlatform {
	workDir := filepath.Join(os.TempDir(), "retrogame")
	os.MkdirAll(workDir, 0755)

	return &DOSBoxPlatform{
		registryPath: registryPath,
		layerManager: layers.NewManager(registryPath),
		workDir:      workDir,
	}
}

func (p *DOSBoxPlatform) Name() string {
	return "dosbox"
}

func (p *DOSBoxPlatform) PrepareInstall(layerSHAs []string, installCmd string) (string, error) {
	workDir, err := os.MkdirTemp("", "retro-install-")
	if err != nil {
		return "", err
	}

	for _, sha := range layerSHAs {
		layerPath := p.layerManager.GetLayerPath(sha)
		data, err := os.ReadFile(layerPath)
		if err != nil {
			continue
		}

		if isTarArchive(data) {
			if err := p.extractTar(workDir, data); err != nil {
				return "", err
			}
			continue
		}

		fl, err := layers.DeserializeFileLayer(data)
		if err != nil {
			continue
		}

		destPath := filepath.Join(workDir, fl.Name)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return "", err
		}
		if err := os.WriteFile(destPath, fl.Content, 0644); err != nil {
			return "", err
		}
	}

	configPath := filepath.Join(workDir, "dosbox-install.conf")
	installDir := dosDir(installCmd)

	config := fmt.Sprintf(`[autoexec]
mount C %s
C:
CD %s
%s
exit
`, workDir, installDir, installCmd)

	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return "", err
	}

	return configPath, nil
}

func (p *DOSBoxPlatform) Run(m *manifest.Manifest) error {
	imageDir, err := p.extractLayers(m)
	if err != nil {
		return err
	}

	configPath := filepath.Join(imageDir, "dosbox.conf")
	if err := p.generateDOSBoxConfig(m, imageDir, configPath); err != nil {
		return err
	}

	cmd := exec.Command("dosbox", "-conf", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func (p *DOSBoxPlatform) extractLayers(m *manifest.Manifest) (string, error) {
	imageDir := filepath.Join(p.workDir, m.Name+"-"+m.Tag)
	if err := os.MkdirAll(imageDir, 0755); err != nil {
		return "", err
	}

	for _, layerSHA := range m.Layers {
		layerPath := p.layerManager.GetLayerPath(layerSHA)
		data, err := os.ReadFile(layerPath)
		if err != nil {
			return "", fmt.Errorf("failed to read layer %s: %w", layerSHA, err)
		}

		if isTarArchive(data) {
			if err := p.extractTar(imageDir, data); err != nil {
				return "", fmt.Errorf("failed to extract tar layer %s: %w", layerSHA, err)
			}
		} else {
			fl, err := layers.DeserializeFileLayer(data)
			if err != nil {
				continue
			}

			destPath := filepath.Join(imageDir, fl.Name)
			if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
				return "", err
			}
			if err := os.WriteFile(destPath, fl.Content, 0644); err != nil {
				return "", fmt.Errorf("failed to write layer %s: %w", layerSHA, err)
			}
		}
	}

	return imageDir, nil
}

func isTarArchive(data []byte) bool {
	return len(data) > 262 && data[257] == 'u' && data[258] == 's' && data[259] == 't' && data[260] == 'a' && data[261] == 'r'
}

func (p *DOSBoxPlatform) extractTar(destDir string, data []byte) error {
	tr := tar.NewReader(strings.NewReader(string(data)))
	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		target := filepath.Join(destDir, header.Name)
		target = filepath.FromSlash(target)
		if header.Typeflag == tar.TypeDir {
			if err := os.MkdirAll(target, 0755); err != nil {
				return err
			}
		} else {
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return err
			}
			outFile, err := os.Create(target)
			if err != nil {
				return err
			}
			if _, err := io.Copy(outFile, tr); err != nil {
				outFile.Close()
				return err
			}
			outFile.Close()
			os.Chmod(target, 0644)
		}
	}
	return nil
}

func dosDir(path string) string {
	if strings.Contains(path, ":") {
		parts := strings.Split(path, ":")
		path = parts[1]
	}
	lastSlash := strings.LastIndex(path, "\\")
	if lastSlash < 0 {
		lastSlash = strings.LastIndex(path, "/")
	}
	if lastSlash < 0 {
		return path
	}
	return path[:lastSlash]
}

func dosPath(path string) string {
	path = strings.ReplaceAll(path, "/", "\\")
	return path
}

func (p *DOSBoxPlatform) generateDOSBoxConfig(m *manifest.Manifest, imageDir, configPath string) error {
	workDir := m.WorkingDir
	if workDir == "" {
		workDir = dosDir(m.Entrypoint)
	} else if strings.Contains(workDir, ":") {
		parts := strings.Split(workDir, ":")
		workDir = parts[1]
	}
	workDir = dosPath(workDir)

	config := fmt.Sprintf(`[autoexec]
mount C %s
C:
CD %s
%s
exit
`, imageDir, workDir, filepath.Base(m.Entrypoint))

	return os.WriteFile(configPath, []byte(config), 0644)
}
