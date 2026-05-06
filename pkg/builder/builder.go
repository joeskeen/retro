package builder

import (
	"archive/tar"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"retrogame/pkg/layers"
	"retrogame/pkg/manifest"
	"retrogame/pkg/parser"
)

type Builder struct {
	registryPath string
	layerManager *layers.Manager
}

func NewBuilder(registryPath string) *Builder {
	return &Builder{
		registryPath: registryPath,
		layerManager: layers.NewManager(registryPath),
	}
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
	}

	manifestPath := filepath.Join(imagePath, "manifest.json")
	if err := m.Save(manifestPath); err != nil {
		return nil, err
	}

	return m, nil
}

func (b *Builder) createCopyLayer(instr parser.Instruction, contextPath string) (*layers.Layer, error) {
	srcPath := filepath.Join(contextPath, instr.Source)

	srcInfo, err := os.Stat(srcPath)
	if err != nil {
		return nil, fmt.Errorf("copy source not found: %s", srcPath)
	}

	var data []byte
	if srcInfo.IsDir() {
		data, err = b.tarDirectory(srcPath)
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

func (b *Builder) tarDirectory(dirPath string) ([]byte, error) {
	tmpFile, err := os.CreateTemp("", "retro-tar-*.tar")
	if err != nil {
		return nil, err
	}
	defer tmpFile.Close()
	defer os.Remove(tmpFile.Name())

	tw := tar.NewWriter(tmpFile)
	dirName := filepath.Base(dirPath)
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
		header.Name = filepath.Join(dirName, relPath)
		header.Name = filepath.ToSlash(header.Name)

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
	tw.Close()

	data, err := os.ReadFile(tmpFile.Name())
	return data, err
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
