package layers

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type LayerType string

const (
	LayerTypeCopy   LayerType = "copy"
	LayerTypeConfig LayerType = "config"
	LayerTypeBase   LayerType = "base"
)

type Layer struct {
	Type    LayerType `json:"type"`
	SHA256  string    `json:"sha256"`
	Path    string    `json:"path,omitempty"`
	Content string    `json:"content,omitempty"`
}

type FileLayer struct {
	Name    string `json:"name"`
	Content []byte `json:"content"`
}

func (fl *FileLayer) Serialize() ([]byte, error) {
	return json.Marshal(fl)
}

func DeserializeFileLayer(data []byte) (*FileLayer, error) {
	var fl FileLayer
	if err := json.Unmarshal(data, &fl); err != nil {
		return nil, err
	}
	return &fl, nil
}

type Manager struct {
	RegistryPath string
}

func NewManager(registryPath string) *Manager {
	return &Manager{RegistryPath: registryPath}
}

func (m *Manager) StoreLayer(layer *Layer) error {
	layerPath := filepath.Join(m.RegistryPath, "layers", layer.SHA256+".layer")
	if err := os.MkdirAll(filepath.Dir(layerPath), 0755); err != nil {
		return err
	}

	if layer.Content != "" {
		return os.WriteFile(layerPath, []byte(layer.Content), 0644)
	}

	if layer.Path != "" {
		data, err := os.ReadFile(layer.Path)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("layer source file not found: %s", layer.Path)
			}
			return err
		}
		return os.WriteFile(layerPath, data, 0644)
	}

	return fmt.Errorf("layer has no content or path")
}

func (m *Manager) GetLayerPath(sha256 string) string {
	return filepath.Join(m.RegistryPath, "layers", sha256+".layer")
}

func ComputeSHA256(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func ComputeSHA256String(data []byte) string {
	return fmt.Sprintf("%x", sha256.Sum256(data))
}

func ComputeSHA256Reader(r io.Reader) (string, error) {
	hash := sha256.New()
	if _, err := io.Copy(hash, r); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", hash.Sum(nil)), nil
}
