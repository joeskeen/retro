package overlay

import (
	"archive/tar"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"retrogame/pkg/layers"
)

type OverlayFS struct {
	lowerPaths []string
	upperPath  string
	workPath   string
	mergedPath string
}

func NewOverlayFS(lowerPaths []string, upperPath, workPath, mergedPath string) *OverlayFS {
	return &OverlayFS{
		lowerPaths: lowerPaths,
		upperPath:  upperPath,
		workPath:   workPath,
		mergedPath: mergedPath,
	}
}

func (o *OverlayFS) Mount() error {
	if err := os.MkdirAll(o.upperPath, 0755); err != nil {
		return fmt.Errorf("failed to create upper dir: %w", err)
	}
	if err := os.MkdirAll(o.workPath, 0755); err != nil {
		return fmt.Errorf("failed to create work dir: %w", err)
	}
	if err := os.MkdirAll(o.mergedPath, 0755); err != nil {
		return fmt.Errorf("failed to create merged dir: %w", err)
	}

	lower := ""
	for _, p := range o.lowerPaths {
		if lower != "" {
			lower += ":"
		}
		lower += p
	}

	args := []string{
		"-t", "overlay",
		"-o", fmt.Sprintf("lowerdir=%s,upperdir=%s,workdir=%s", lower, o.upperPath, o.workPath),
		"overlay",
		o.mergedPath,
	}

	cmd := exec.Command("mount", args...)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (o *OverlayFS) Unmount() error {
	cmd := exec.Command("umount", o.mergedPath)
	return cmd.Run()
}

func (o *OverlayFS) UpperPath() string {
	return o.upperPath
}

func (o *OverlayFS) MergedPath() string {
	return o.mergedPath
}

type LayerMerger struct {
	registryPath string
	workDir      string
}

func NewLayerMerger(registryPath string) *LayerMerger {
	workDir := filepath.Join(registryPath, "overlay-work")
	os.MkdirAll(workDir, 0755)
	return &LayerMerger{
		registryPath: registryPath,
		workDir:      workDir,
	}
}

func (m *LayerMerger) MergeLayers(layerSHAs []string) (string, error) {
	if len(layerSHAs) == 0 {
		return "", fmt.Errorf("no layers to merge")
	}

	layerDir := filepath.Join(m.workDir, "layers")
	if err := os.MkdirAll(layerDir, 0755); err != nil {
		return "", err
	}

	lowerDirs := []string{}
	for i, sha := range layerSHAs[:len(layerSHAs)-1] {
		layerPath := m.getLayerPath(sha)
		linkPath := filepath.Join(layerDir, fmt.Sprintf("lower-%d", i))
		if err := os.Symlink(layerPath, linkPath); err != nil {
			return "", fmt.Errorf("failed to create symlink for layer %s: %w", sha, err)
		}
		lowerDirs = append(lowerDirs, linkPath)
	}

	upperDir := filepath.Join(m.workDir, "upper")
	workDir := filepath.Join(m.workDir, "work")
	mergedDir := filepath.Join(m.workDir, "merged")

	os.MkdirAll(upperDir, 0755)
	os.MkdirAll(workDir, 0755)
	os.MkdirAll(mergedDir, 0755)

	ov := NewOverlayFS(lowerDirs, upperDir, workDir, mergedDir)
	if err := ov.Mount(); err != nil {
		return "", fmt.Errorf("failed to mount overlay: %w", err)
	}

	return mergedDir, nil
}

func (m *LayerMerger) Commit(upperDir string) (*layers.Layer, error) {
	data, err := os.ReadDir(upperDir)
	if err != nil {
		return nil, err
	}

	if len(data) == 0 {
		return &layers.Layer{
			Type:   layers.LayerTypeCopy,
			SHA256: layers.ComputeSHA256([]byte{}),
		}, nil
	}

	mergedData, err := m.tarUpperLayer(upperDir)
	if err != nil {
		return nil, fmt.Errorf("failed to tar upper layer: %w", err)
	}

	return &layers.Layer{
		Type:    layers.LayerTypeCopy,
		SHA256:  layers.ComputeSHA256(mergedData),
		Content: string(mergedData),
	}, nil
}

func (m *LayerMerger) tarUpperLayer(upperDir string) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "retro-layer-*.tar")
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()

	tw := &tarWriter{writer: tmpFile}
	if err := filepath.Walk(upperDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(upperDir, path)
		if err != nil {
			return err
		}
		if relPath == "." {
			return nil
		}
		relPath = strings.ReplaceAll(relPath, "\\", "/")
		return tw.writeFile(path, relPath, info)
	}); err != nil {
		return nil, err
	}

	if err := tw.Close(); err != nil {
		return nil, err
	}

	return os.ReadFile(tmpFile.Name())
}

func (m *LayerMerger) getLayerPath(sha string) string {
	return filepath.Join(m.registryPath, "layers", sha+".layer")
}

type tarWriter struct {
	writer *os.File
	tw     *tar.Writer
}

func (tw *tarWriter) writeFile(path, name string, info os.FileInfo) error {
	if tw.tw == nil {
		tw.tw = tar.NewWriter(tw.writer)
	}
	header, err := tar.FileInfoHeader(info, "")
	if err != nil {
		return err
	}
	header.Name = name
	if err := tw.tw.WriteHeader(header); err != nil {
		return err
	}
	if !info.IsDir() {
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tw.tw.Write(data)
		return err
	}
	return nil
}

func (tw *tarWriter) Close() error {
	if tw.tw != nil {
		return tw.tw.Close()
	}
	return nil
}
