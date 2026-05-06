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
}

func NewBuilder(registryPath string, platform platforms.Platform) *Builder {
	return &Builder{
		registryPath: registryPath,
		layerManager: layers.NewManager(registryPath),
		platform:     platform,
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
		layer, err := b.createCopyLayer(copyInstr, contextPath)
		if err != nil {
			return nil, err
		}
		layerSHAs = append(layerSHAs, layer.SHA256)
		if err := b.layerManager.StoreLayer(layer); err != nil {
			return nil, err
		}
	}

	if rf.Install != "" {
		installLayer, err := b.runInstallStep(layerSHAs, rf.Install)
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

	manifestPath := filepath.Join(imagePath, "manifest.json")
	if err := m.Save(manifestPath); err != nil {
		return nil, err
	}

	return m, nil
}

func (b *Builder) runInstallStep(layerSHAs []string, installCmd string) (*layers.Layer, error) {
	configPath, err := b.platform.PrepareInstall(layerSHAs, installCmd)
	if err != nil {
		return nil, err
	}

	fmt.Printf("Running installer: %s\n", installCmd)
	cmd := exec.Command("dosbox", "-conf", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	if err := cmd.Run(); err != nil {
		fmt.Printf("Installer exited with error (may be normal): %v\n", err)
	}

	workDir := filepath.Dir(configPath)
	modifiedData, err := b.tarDirectoryWithPrefix(workDir, "")
	if err != nil {
		return nil, err
	}

	return &layers.Layer{
		Type:    layers.LayerTypeCopy,
		SHA256:  layers.ComputeSHA256(modifiedData),
		Content: string(modifiedData),
	}, nil
}

func (b *Builder) createCopyLayer(instr parser.Instruction, contextPath string) (*layers.Layer, error) {
	srcPath := filepath.Join(contextPath, instr.Source)

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return nil, fmt.Errorf("copy source not found: %s", srcPath)
	}

	var data []byte
	if srcInfo.IsDir() {
		destDir := filepath.Base(instr.Dest)
		data, err = b.tarDirectoryWithPrefix(srcPath, destDir)
		if err != nil {
			return nil, err
		}
		layer := &layers.Layer{
			Type:    layers.LayerTypeCopy,
			SHA256:  layers.ComputeSHA256(data),
			Content: string(data),
		}
		return layer, nil
	} else {
		data, err = os.ReadFile(srcPath)
		if err != nil {
			return nil, err
		}
		fl := &layers.FileLayer{
			Name:    filepath.Base(srcPath),
			Content: data,
		}
		layerData, _ := fl.Serialize()
		layer := &layers.Layer{
			Type:    layers.LayerTypeCopy,
			Path:    srcPath,
			Content: string(layerData),
			SHA256:  layers.ComputeSHA256(layerData),
		}
		return layer, nil
	}
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
