package commands

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"retrogame/pkg/builder"
	"retrogame/pkg/config"
	"retrogame/pkg/parser"
	"retrogame/pkg/platforms/dosbox"
	"retrogame/pkg/registry"
	"retrogame/pkg/runtime"
	"retrogame/pkg/transport/git"
)

func getRegistryPath() string {
	if usr, err := user.Current(); err == nil {
		return filepath.Join(usr.HomeDir, ".retro", "registry")
	}
	return ".retro"
}

func getConfigPath() string {
	if usr, err := user.Current(); err == nil {
		return filepath.Join(usr.HomeDir, ".retro", "config.toml")
	}
	return ".retro/config.toml"
}

func RunBuild() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: retro build <path>")
	}

	contextPath := os.Args[2]
	retrofilePath := filepath.Join(contextPath, "Retrofile")

	rf, err := parser.ParseRetrofile(retrofilePath)
	if err != nil {
		return fmt.Errorf("failed to parse Retrofile: %w", err)
	}

	registryPath := getRegistryPath()
	p := dosbox.New(registryPath)
	b := builder.NewBuilder(registryPath, p)

	m, err := b.Build(rf, contextPath)
	if err != nil {
		return fmt.Errorf("build failed: %w", err)
	}

	fmt.Printf("Built %s:%s\n", m.Name, m.Tag)
	return nil
}

func RunRun() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: retro run <image> [--remote <remote>]")
	}

	imageRef := os.Args[2]
	remoteName := ""

	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--remote" && i+1 < len(os.Args) {
			remoteName = os.Args[i+1]
			i++
		}
	}

	name, tag := parseImageRef(imageRef)
	registryPath := getRegistryPath()
	reg := registry.New(registryPath)

	m, err := reg.GetManifest(name, tag)
	if err != nil {
		if remoteName != "" {
			if pullErr := runPullForImage(name, tag, remoteName); pullErr == nil {
				m, err = reg.GetManifest(name, tag)
			} else {
				return fmt.Errorf("image not found locally and pull failed: %w", pullErr)
			}
		}
		if m == nil {
			return fmt.Errorf("image not found %s:%s (and no remote specified)", name, tag)
		}
	}

	rt := runtime.New(registryPath)
	if err := rt.Run(m); err != nil {
		return fmt.Errorf("runtime error: %w", err)
	}

	return nil
}

func runPullForImage(name, tag, remoteName string) error {
	cfgPath := getConfigPath()
	registryPath := getRegistryPath()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return err
	}

	remote, err := cfg.GetRemote(remoteName)
	if err != nil {
		return err
	}

	gitPath := filepath.Join(getRegistryPath(), "git", remote.Name)
	t := git.New(remote.URL, gitPath)

	if err := t.CloneOrOpen(); err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to open repo: %w", err)
		}
	} else {
		cfg.Remotes = append(cfg.Remotes, config.Remote{Name: remote.Name, URL: remote.URL})
		if cfg.DefaultRemote == "" {
			cfg.DefaultRemote = remote.Name
		}
		config.Save(cfg, cfgPath)
	}

	if err := t.Pull(); err != nil {
		fmt.Printf("Warning: pull failed: %v\n", err)
	}

	data, err := t.GetManifest(name, tag)
	if err != nil {
		return fmt.Errorf("manifest not found: %w", err)
	}

	manifestPath := filepath.Join(registryPath, name, tag, "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return fmt.Errorf("failed to create manifest dir: %w", err)
	}
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	fmt.Printf("Pulled %s:%s from %s\n", name, tag, remote.Name)
	return nil
}

func RunImages() error {
	registryPath := getRegistryPath()
	reg := registry.New(registryPath)

	images, err := reg.ListImages()
	if err != nil {
		return fmt.Errorf("failed to list images: %w", err)
	}

	if len(images) == 0 {
		fmt.Println("No images found")
		return nil
	}

	for _, img := range images {
		fmt.Println(img)
	}
	return nil
}

func RunRm() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: retro rm <image>")
	}

	imageRef := os.Args[2]
	name, tag := parseImageRef(imageRef)
	registryPath := getRegistryPath()
	manifestPath := filepath.Join(registryPath, name, tag, "manifest.json")

	if _, err := os.Stat(manifestPath); os.IsNotExist(err) {
		return fmt.Errorf("image not found: %s:%s", name, tag)
	}

	if err := os.RemoveAll(filepath.Join(registryPath, name, tag)); err != nil {
		return fmt.Errorf("failed to remove image: %w", err)
	}

	fmt.Printf("Removed %s:%s\n", name, tag)
	return nil
}

func RunPush() error {
	if len(os.Args) < 4 {
		return fmt.Errorf("usage: retro push <image> [--remote <remote>]")
	}

	imageRef := os.Args[2]
	remoteName := ""

	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--remote" && i+1 < len(os.Args) {
			remoteName = os.Args[i+1]
			i++
		}
	}

	name, tag := parseImageRef(imageRef)
	cfgPath := getConfigPath()

	var remote *config.Remote
	if remoteName != "" {
		cfg, err := config.Load(cfgPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if cfg != nil {
			remote, err = cfg.GetRemote(remoteName)
			if err != nil {
				return fmt.Errorf("remote not found: %s", remoteName)
			}
		}
	} else {
		cfg, err := config.Load(cfgPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if cfg != nil {
			remote, err = cfg.GetDefaultRemote()
			if err != nil {
				return fmt.Errorf("no remote specified and no default set")
			}
		}
	}

	if remote == nil {
		return fmt.Errorf("no remote specified")
	}

	registryPath := getRegistryPath()
	reg := registry.New(registryPath)

	m, err := reg.GetManifest(name, tag)
	if err != nil {
		return fmt.Errorf("image not found: %w", err)
	}

	gitPath := filepath.Join(getRegistryPath(), "git", remote.Name)
	t := git.New(remote.URL, gitPath)

	if err := t.CloneOrOpen(); err != nil {
		return fmt.Errorf("failed to clone/open repo: %w", err)
	}

	data, err := os.ReadFile(filepath.Join(registryPath, name, tag, "manifest.json"))
	if err != nil {
		manifestPath := filepath.Join(registryPath, name, tag, "manifest.json")
		if err := m.Save(manifestPath); err != nil {
			return fmt.Errorf("failed to get manifest: %w", err)
		}
		data, _ = os.ReadFile(manifestPath)
	}

	if err := t.AddManifest(name, tag, data); err != nil {
		return fmt.Errorf("failed to add manifest: %w", err)
	}

	for _, layerSHA := range m.Layers {
		layerPath := filepath.Join(registryPath, "layers", layerSHA+".layer")
		data, err := os.ReadFile(layerPath)
		if err != nil {
			continue
		}
		if err := t.AddLayer(layerSHA, data); err != nil {
			return fmt.Errorf("failed to add layer %s: %v", layerSHA[:8], err)
		}
	}

	if err := t.Push(); err != nil {
		return fmt.Errorf("push failed: %w", err)
	}

	fmt.Printf("Pushed %s:%s to %s\n", name, tag, remote.Name)
	return nil
}

func RunPull() error {
	if len(os.Args) < 4 {
		return fmt.Errorf("usage: retro pull <image> [--remote <remote>]")
	}

	imageRef := os.Args[2]
	remoteName := ""

	for i := 3; i < len(os.Args); i++ {
		if os.Args[i] == "--remote" && i+1 < len(os.Args) {
			remoteName = os.Args[i+1]
			i++
		}
	}

	name, tag := parseImageRef(imageRef)
	cfgPath := getConfigPath()

	var remote *config.Remote
	if remoteName != "" {
		cfg, err := config.Load(cfgPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if cfg != nil {
			remote, err = cfg.GetRemote(remoteName)
			if err != nil {
				return fmt.Errorf("remote not found: %s", remoteName)
			}
		}
	} else {
		cfg, err := config.Load(cfgPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if cfg != nil {
			remote, err = cfg.GetDefaultRemote()
			if err != nil {
				return fmt.Errorf("no remote specified and no default set")
			}
		}
	}

	if remote == nil {
		return fmt.Errorf("no remote specified")
	}

	gitPath := filepath.Join(getRegistryPath(), "git", remote.Name)
	t := git.New(remote.URL, gitPath)

	if err := t.CloneOrOpen(); err != nil {
		return fmt.Errorf("failed to clone/open repo: %w", err)
	}

	if err := t.Pull(); err != nil {
		fmt.Printf("Warning: pull failed: %v\n", err)
	}

	if tag == "1.0" {
		if availableTag := t.FindAvailableTag(name); availableTag != "" {
			tag = availableTag
			fmt.Printf("[pull] Using available tag: %s\n", tag)
		}
	}

	data, err := t.GetManifest(name, tag)
	if err != nil {
		return fmt.Errorf("manifest not found: %w", err)
	}

	registryPath := getRegistryPath()
	manifestPath := filepath.Join(registryPath, name, tag, "manifest.json")
	if err := os.MkdirAll(filepath.Dir(manifestPath), 0755); err != nil {
		return fmt.Errorf("failed to create manifest dir: %w", err)
	}
	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("failed to save manifest: %w", err)
	}

	fmt.Printf("Pulled %s:%s from %s\n", name, tag, remote.Name)
	return nil
}

func RunRemoteAdd() error {
	if len(os.Args) < 5 {
		return fmt.Errorf("usage: retro remote add <name> <url>")
	}

	name := os.Args[3]
	url := os.Args[4]

	cfgPath := getConfigPath()
	cfg := &config.Config{}

	existingCfg, err := config.Load(cfgPath)
	if err == nil {
		cfg = existingCfg
	}

	for _, r := range cfg.Remotes {
		if r.Name == name {
			return fmt.Errorf("remote %q already exists", name)
		}
	}

	cfg.Remotes = append(cfg.Remotes, config.Remote{Name: name, URL: url})

	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Added remote %s\n", name)
	return nil
}

func RunRemoteList() error {
	cfgPath := getConfigPath()
	cfg, err := config.Load(cfgPath)
	if err != nil {
		if os.IsNotExist(err) {
			fmt.Println("No remotes configured")
			return nil
		}
		return fmt.Errorf("failed to load config: %w", err)
	}

	if len(cfg.Remotes) == 0 {
		fmt.Println("No remotes configured")
		return nil
	}

	for _, r := range cfg.Remotes {
		marker := " "
		if cfg.DefaultRemote == r.Name {
			marker = "*"
		}
		fmt.Printf("%s %s - %s\n", marker, r.Name, r.URL)
	}
	return nil
}

func RunRemoteRemove() error {
	if len(os.Args) < 4 {
		return fmt.Errorf("usage: retro remote remove <name>")
	}

	name := os.Args[3]
	cfgPath := getConfigPath()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	found := false
	newRemotes := []config.Remote{}
	for _, r := range cfg.Remotes {
		if r.Name == name {
			found = true
		} else {
			newRemotes = append(newRemotes, r)
		}
	}

	if !found {
		return fmt.Errorf("remote not found: %s", name)
	}

	cfg.Remotes = newRemotes
	if cfg.DefaultRemote == name {
		cfg.DefaultRemote = ""
	}

	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Removed remote %s\n", name)
	return nil
}

func RunRemoteDefault() error {
	if len(os.Args) < 4 {
		return fmt.Errorf("usage: retro remote default <name>")
	}

	name := os.Args[3]
	cfgPath := getConfigPath()

	cfg, err := config.Load(cfgPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	found := false
	for _, r := range cfg.Remotes {
		if r.Name == name {
			found = true
			break
		}
	}

	if !found {
		return fmt.Errorf("remote not found: %s", name)
	}

	cfg.DefaultRemote = name

	if err := config.Save(cfg, cfgPath); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Printf("Default remote set to %s\n", name)
	return nil
}

func parseImageRef(ref string) (string, string) {
	name := ref
	tag := "1.0"

	for i := len(ref) - 1; i >= 0; i-- {
		if ref[i] == ':' {
			name = ref[:i]
			tag = ref[i+1:]
			break
		}
	}
	return name, tag
}

func RunPrune() error {
	if len(os.Args) < 3 {
		return fmt.Errorf("usage: retro prune [--remote <remote>]")
	}

	remoteName := ""
	for i := 2; i < len(os.Args); i++ {
		if os.Args[i] == "--remote" && i+1 < len(os.Args) {
			remoteName = os.Args[i+1]
			i++
		}
	}

	cfgPath := getConfigPath()
	var remote *config.Remote

	if remoteName == "" {
		cfg, err := config.Load(cfgPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if cfg != nil {
			remote, err = cfg.GetDefaultRemote()
			if err != nil {
				return fmt.Errorf("no remote specified and no default set")
			}
		}
	} else {
		cfg, err := config.Load(cfgPath)
		if err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("failed to load config: %w", err)
		}
		if cfg != nil {
			remote, err = cfg.GetRemote(remoteName)
			if err != nil {
				return fmt.Errorf("remote not found: %s", remoteName)
			}
		}
	}

	if remote == nil {
		return fmt.Errorf("no remote specified")
	}

	gitPath := filepath.Join(getRegistryPath(), "git", remote.Name)
	t := git.New(remote.URL, gitPath)

	if err := t.CloneOrOpen(); err != nil {
		return fmt.Errorf("failed to clone/open repo: %w", err)
	}

	pruned, err := t.Prune()
	if err != nil {
		return fmt.Errorf("prune failed: %w", err)
	}

	fmt.Printf("Pruned %d unreferenced layers from %s\n", pruned, remote.Name)
	return nil
}
