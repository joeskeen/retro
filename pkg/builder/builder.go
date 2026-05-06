package builder

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
	"retrogame/pkg/parser"
	"retrogame/pkg/platforms"
)

type Builder struct {
	registryPath string
	layerManager *layers.Manager
	platform     platforms.Platform
	buildWorkDir string
}

func NewBuilder(registryPath string, platform platforms.Platform) *Builder {
	buildWorkDir := filepath.Join(registryPath, "build")
	os.MkdirAll(buildWorkDir, 0755)
	return &Builder{
		registryPath: registryPath,
		layerManager: layers.NewManager(registryPath),
		platform:     platform,
		buildWorkDir: buildWorkDir,
	}
}

func (b *Builder) tarDirectoryWithPrefix(dirPath, prefix string) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "retro-tar-*.tar")
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	tw := tar.NewWriter(tmpFile)
	prefix = strings.ReplaceAll(prefix, "\\", "/")
	filepath.Walk(dirPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		relPath, err := filepath.Rel(dirPath, path)
		if err != nil {
			return err
		}

		if relPath == "." {
			return nil
		}

		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		relPath = strings.ReplaceAll(relPath, "\\", "/")
		header.Name = prefix + "/" + relPath

		if err := tw.WriteHeader(header); err != nil {
			return err
		}

		if !info.IsDir() {
			data, err := os.ReadFile(path)
			if err != nil {
				return err
			}
			if _, err := tw.Write(data); err != nil {
				return err
			}
		}

		return nil
	})
	if err := tw.Close(); err != nil {
		return nil, err
	}

	data, err := os.ReadFile(tmpFile.Name())
	return data, err
}

func (b *Builder) Build(rf *parser.Retrofile, contextPath string) (*manifest.Manifest, error) {
	if rf.Tag == "" {
		return nil, fmt.Errorf("Retrofile requires TAG instruction")
	}

	imageName := parseName(rf.Tag)
	imageTag := parseTag(rf.Tag)
	imagePath := filepath.Join(b.registryPath, imageName, imageTag)
	if err := os.MkdirAll(imagePath, 0755); err != nil {
		return nil, err
	}

	layerSHAs := []string{}

	baseImageLayer := &layers.Layer{
		Type:    layers.LayerTypeBase,
		Content: rf.BaseImage.Name,
	}
	baseImageLayer.SHA256 = layers.ComputeSHA256([]byte(baseImageLayer.Content))
	layerSHAs = append(layerSHAs, baseImageLayer.SHA256)
	if err := b.layerManager.StoreLayer(baseImageLayer); err != nil {
		return nil, err
	}

	for _, copyInstr := range rf.Copy {
		srcPath := filepath.Join(contextPath, copyInstr.Source)
		srcInfo, err := os.Stat(srcPath)
		if err != nil {
			return nil, fmt.Errorf("copy source not found: %s", srcPath)
		}

		var data []byte
		if srcInfo.IsDir() {
			destDir := dosName(copyInstr.Dest)
			data, err = b.tarDirectoryWithPrefix(srcPath, destDir)
		} else {
			data, err = os.ReadFile(srcPath)
		}
		if err != nil {
			return nil, err
		}

		layer := &layers.Layer{
			Type:    layers.LayerTypeCopy,
			Content: string(data),
		}
		if srcInfo.IsDir() {
			layer.SHA256 = layers.ComputeSHA256(data)
		} else {
			fl := &layers.FileLayer{
				Name:    dosBase(srcPath),
				Content: data,
			}
			layerData, _ := fl.Serialize()
			layer.SHA256 = layers.ComputeSHA256(layerData)
			layer.Content = string(layerData)
		}
		layerSHAs = append(layerSHAs, layer.SHA256)
		if err := b.layerManager.StoreLayer(layer); err != nil {
			return nil, err
		}
	}

	if rf.Install != "" {
		installLayer, _, err := b.runInstallStep(layerSHAs, rf.Install, contextPath)
		if err != nil {
			return nil, fmt.Errorf("install step failed: %w", err)
		}
		layerSHAs = append(layerSHAs, installLayer.SHA256)
		if err := b.layerManager.StoreLayer(installLayer); err != nil {
			return nil, err
		}
	}

	entrypointLayer := &layers.Layer{
		Type:    layers.LayerTypeConfig,
		Content: rf.Entrypoint,
	}
	entrypointLayer.SHA256 = layers.ComputeSHA256([]byte(entrypointLayer.Content))
	layerSHAs = append(layerSHAs, entrypointLayer.SHA256)
	if err := b.layerManager.StoreLayer(entrypointLayer); err != nil {
		return nil, err
	}

	m := &manifest.Manifest{
		Name:       imageName,
		Tag:        imageTag,
		Runtime:    rf.BaseImage.Name,
		BaseImage:  rf.BaseImage.Name,
		Layers:     layerSHAs,
		Entrypoint: rf.Entrypoint,
		WorkingDir: rf.WorkingDir,
		Install:    rf.Install,
	}

	baseline, err := b.computeBaselineFromLayers(layerSHAs)
	if err == nil && len(baseline) > 0 {
		m.BaselineSHA = baseline
	}

	manifestPath := filepath.Join(imagePath, "manifest.json")
	if err := m.Save(manifestPath); err != nil {
		return nil, err
	}

	return m, nil
}

func (b *Builder) runInstallStep(layerSHAs []string, installCmd string, contextPath string) (*layers.Layer, string, error) {
	workDir, err := os.MkdirTemp("", "retro-install-")
	if err != nil {
		return nil, "", err
	}

	for _, sha := range layerSHAs {
		layerPath := b.layerManager.GetLayerPath(sha)
		data, err := os.ReadFile(layerPath)
		if err != nil {
			continue
		}

		if isTarArchive(data) {
			if err := b.extractTar(workDir, data); err != nil {
				return nil, "", err
			}
			continue
		}

		fl, err := layers.DeserializeFileLayer(data)
		if err != nil {
			continue
		}

		destPath := filepath.Join(workDir, fl.Name)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return nil, "", err
		}
		if err := os.WriteFile(destPath, fl.Content, 0644); err != nil {
			return nil, "", err
		}
	}

	installDir := installDirFromCmd(installCmd)

	configPath := "/tmp/retro-dosbox-install.conf"
	if envPath := os.Getenv("RETRO_INSTALL_CONF"); envPath != "" {
		configPath = envPath
	}
	config := fmt.Sprintf(`[autoexec]
mount C %s
C:
CD %s
%s
%s
`, workDir, installDir, installCmd, pauseCmd())

	if err := os.WriteFile(configPath, []byte(config), 0644); err != nil {
		return nil, "", err
	}

	fmt.Printf("Running installer: %s\n", installCmd)
	fmt.Printf("Config file: %s\n", configPath)
	cmd := exec.Command("dosbox", "-conf", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Printf("Installer exited with error (may be normal): %v\n", err)
	}

	modifiedData, err := b.tarDirectoryWithPrefix(workDir, "")
	if err != nil {
		return nil, "", err
	}

	return &layers.Layer{
		Type:    layers.LayerTypeCopy,
		SHA256:  layers.ComputeSHA256(modifiedData),
		Content: string(modifiedData),
	}, workDir, nil
}

func (b *Builder) computeBaselineSHAs(installedPath string) (map[string]string, error) {
	baseline := make(map[string]string)

	err := filepath.Walk(installedPath, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, err := filepath.Rel(installedPath, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		relPath = unixPath(relPath)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		baseline[relPath] = layers.ComputeSHA256(data)
		return nil
	})

	return baseline, err
}

func copyDir(src, dest string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		target := filepath.Join(dest, relPath)
		if info.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
}

func dosDir(path string) string {
	if strings.Contains(path, ":") {
		parts := strings.Split(path, ":")
		path = parts[1]
	}
	if strings.HasPrefix(path, "/") {
		return ""
	}
	lastSlash := -1
	secondLastSlash := -1
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '\\' || path[i] == '/' {
			if lastSlash == -1 {
				lastSlash = i
			} else {
				secondLastSlash = i
				break
			}
		}
	}
	if lastSlash == -1 {
		return ""
	}
	if secondLastSlash == -1 {
		if lastSlash == 0 {
			return ""
		}
		return path[:lastSlash]
	}
	return path[secondLastSlash+1 : lastSlash]
}

func dosPath(path string) string {
	path = strings.ReplaceAll(path, "/", "\\")
	return path
}

func dosBase(path string) string {
	if strings.Contains(path, ":") {
		colonIdx := strings.Index(path, ":")
		if colonIdx == len(path)-1 {
			return path
		}
		path = path[colonIdx+1:]
	}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '\\' || path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

func dosName(path string) string {
	if strings.Contains(path, ":") {
		colonIdx := strings.Index(path, ":")
		path = path[colonIdx+1:]
		path = strings.TrimLeft(path, "\\/")
	}
	path = strings.TrimRight(path, "/\\")
	if path == "" {
		return ""
	}
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '\\' || path[i] == '/' {
			return path[i+1:]
		}
	}
	return path
}

func unixPath(path string) string {
	path = strings.ReplaceAll(path, "\\", "/")
	return path
}

func dosToUnixPath(path string) string {
	return unixPath(path)
}

func unixToDosPath(path string) string {
	return dosPath(path)
}

func installDirFromCmd(cmd string) string {
	return dosDir(cmd)
}

func pauseCmd() string {
	if os.Getenv("RETRO_PAUSE") == "1" {
		return "pause\nexit"
	}
	return "exit"
}

func isTarArchive(data []byte) bool {
	return len(data) > 262 && data[257] == 'u' && data[258] == 's' && data[259] == 't' && data[260] == 'a' && data[261] == 'r'
}

func (b *Builder) extractTar(destDir string, data []byte) error {
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

func (b *Builder) computeBaselineFromLayers(layerSHAs []string) (map[string]string, error) {
	workDir, err := os.MkdirTemp("", "retro-baseline-")
	if err != nil {
		return nil, err
	}
	defer os.RemoveAll(workDir)

	for _, sha := range layerSHAs {
		layerPath := b.layerManager.GetLayerPath(sha)
		data, err := os.ReadFile(layerPath)
		if err != nil {
			continue
		}

		if isTarArchive(data) {
			if err := b.extractTar(workDir, data); err != nil {
				return nil, err
			}
			continue
		}

		fl, err := layers.DeserializeFileLayer(data)
		if err != nil {
			continue
		}

		destPath := filepath.Join(workDir, fl.Name)
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return nil, err
		}
		if err := os.WriteFile(destPath, fl.Content, 0644); err != nil {
			return nil, err
		}
	}

	baseline := make(map[string]string)
	err = filepath.Walk(workDir, func(path string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return err
		}
		relPath, err := filepath.Rel(workDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		relPath = unixPath(relPath)
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		baseline[relPath] = layers.ComputeSHA256(data)
		return nil
	})

	return baseline, err
}

func parseName(ref string) string {
	parts := strings.Split(ref, ":")
	return parts[0]
}

func parseTag(ref string) string {
	parts := strings.Split(ref, ":")
	if len(parts) > 1 {
		return parts[1]
	}
	return "1.0"
}
