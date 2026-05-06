package manifest

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Manifest struct {
	Name       string   `json:"name"`
	Tag        string   `json:"tag"`
	Runtime    string   `json:"runtime"`
	BaseImage  string   `json:"baseImage"`
	Layers     []string `json:"layers"`
	Entrypoint string   `json:"entrypoint"`
	WorkingDir string   `json:"workingDir,omitempty"`
}

func (m *Manifest) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func Load(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *Manifest) ImagePath(registryPath string) string {
	return filepath.Join(registryPath, m.Name, m.Tag)
}

func (m *Manifest) ManifestPath(registryPath string) string {
	return filepath.Join(m.ImagePath(registryPath), "manifest.json")
}
