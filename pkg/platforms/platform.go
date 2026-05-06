package platforms

import (
	"retrogame/pkg/manifest"
)

type Platform interface {
	Name() string
	Run(m *manifest.Manifest) error
}
