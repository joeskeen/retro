package runtime

import (
	"fmt"

	"retrogame/pkg/manifest"
	"retrogame/pkg/platforms"
	"retrogame/pkg/platforms/dosbox"
)

type Runtime struct {
	registryPath string
	platforms    map[string]platforms.Platform
}

func New(registryPath string) *Runtime {
	r := &Runtime{
		registryPath: registryPath,
		platforms:    make(map[string]platforms.Platform),
	}

	r.platforms["dosbox"] = dosbox.New(registryPath)

	return r
}

func (r *Runtime) Run(m *manifest.Manifest) error {
	platform, ok := r.platforms[m.Runtime]
	if !ok {
		return fmt.Errorf("unsupported runtime: %s", m.Runtime)
	}

	return platform.Run(m)
}
