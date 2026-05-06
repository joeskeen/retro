# Retro

A Docker-like system for running classic games. Build once, run anywhere.

## Concept

Retro lets you package and run retro games using declarative **Retrofiles** (like Dockerfiles). Games are built into layered images that can be shared via Git repositories, then run on any platform with the appropriate emulator.

## Quick Start

```bash
# Check platform readiness
retro check

# Add a game registry
retro remote add my-games git@github.com:username/my-games.git

# Run a game (in this example, "cat:1.0" is a DOSBox image)
retro run cat:1.0

# See what you've got
retro images
```

## Retrofile Example

```dockerfile
TAG cat:1.0
FROM dosbox:1.0
COPY CAT.BAT C:\
ENTRYPOINT C:\CAT.BAT
```

## Installation

### Pre-built Binary

```bash
curl -L https://github.com/your-username/retrogame/releases/latest/download/retro -o ~/bin/retro
chmod +x ~/bin/retro
```

### From Source (requires Go)

```bash
git clone https://github.com/your-username/retrogame.git
cd retrogame
go build -o retro ./cmd/cli
```

## Commands

| Command | Description |
|---------|-------------|
| `retro build <path>` | Build image from Retrofile |
| `retro run <image>` | Run an image |
| `retro run <image> --remote <name>` | Pull and run from remote |
| `retro clone <git-url>` | Clone a game registry |
| `retro push <image>` | Push to remote registry |
| `retro pull <image>` | Pull from remote registry |
| `retro images` | List installed images |
| `retro check` | Check platform readiness |
| `retro remote add <name> <url>` | Add a remote registry |
| `retro remote list` | List configured remotes |

## Requirements

- **git** - https://git-scm.com
- **git lfs** - https://git-lfs.github.com (run `git lfs install`)
- **DOSBox** or other platform emulators for running games

## Supported Git Hosts

- GitHub (2 GB free LFS per repo)
- GitLab (10 GiB free LFS per project)
- Gitea and other Git servers (self-hosted)

## Configuration

Remotes are stored in `~/.retro/config.toml`:

```toml
default-remote = "my-games"

[remote "my-games"]
url = "git@github.com:username/my-games.git"

[remote "my-gitlab"]
url = "git@gitlab.com:username/my-games.git"
```

## Supported Platforms

- **dosbox** - DOSBox (DOS games)
- **dosbox-x** - DOSBox-X (enhanced DOSBox)
- **wine** - Wine (Windows games)

## Architecture

```
pkg/
├── parser/          # Retrofile parsing
├── builder/         # Image building
├── manifest/        # Image manifest format
├── layers/          # Content-addressable layers
├── registry/        # Local registry
├── platforms/       # Platform abstraction
│   └── dosbox/     # DOSBox implementation
└── transport/      # Transport layers (git)
    └── git/        # Git transport
```

## Publishing Games

Games are published by pushing to a Git repository. The structure is:

```
my-games/
├── manifests/
│   ├── game1/1.0/manifest.json
│   └── game2/1.0/manifest.json
└── layers/
    ├── abc123.layer
    └── def456.layer
```

Use any Git host (GitHub, GitLab, Gitea) - no special server needed.

## License

MIT
