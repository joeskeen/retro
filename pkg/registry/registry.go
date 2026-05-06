package registry

import (
	"os"
	"path/filepath"

	"retrogame/pkg/manifest"
)

type Registry struct {
	Path string
}

func New(registryPath string) *Registry {
	return &Registry{Path: registryPath}
}

func (r *Registry) PathForImage(name, tag string) string {
	return filepath.Join(r.Path, name, tag)
}

func (r *Registry) ManifestPath(name, tag string) string {
	return filepath.Join(r.PathForImage(name, tag), "manifest.json")
}

func (r *Registry) GetManifest(name, tag string) (*manifest.Manifest, error) {
	return manifest.Load(r.ManifestPath(name, tag))
}

func (r *Registry) ListImages() ([]string, error) {
	entries, err := os.ReadDir(r.Path)
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

func (r *Registry) EnsureDir(name, tag string) error {
	return os.MkdirAll(r.PathForImage(name, tag), 0755)
}
