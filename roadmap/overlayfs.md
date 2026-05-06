# OverlayFS Layer System

## Problem

As we add more platforms and complex image building, we need a clean way to layer filesystem changes. OverlayFS provides copy-on-write layering that aligns well with our content-addressable storage model.

## Requirements

- Use overlayfs for combining image layers
- Support read-only base layers with writable upper layers
- Handle permission and ownership mapping
- Work across different filesystems and container environments
- Provide atomic layer composition

## Use Cases

1. **Image Building**: Each build step creates a new overlay layer
2. **Runtime Merging**: Game image layers are merged at runtime
3. **Savegames**: Writable overlay captures user changes separately
4. **Disc Mounts**: Insert disc images as additional layers
5. **Configuration Override**: Per-game config layers on top of base

## Dependencies

Required by:
- `dosbox-x-windows` - Handles mount-to-image conversion during boot
- `disc-mounting` - Layer-based disc image storage
- `derived-images` - Storage-efficient delta layers

## Implementation Plan

1. Create `pkg/layers/overlay/` module with:
   ```go
   type OverlayFS struct {
       lower  []string  // read-only base layers
       upper  string    // writable layer
       work   string    // work dir for atomic ops
       merged string    // view of merged filesystem
   }
   ```
2. Implement layer operations:
   - `Merge(base, upper) → merged`
   - `Split(merged) → [base, delta]`
   - `Commit(delta) → new-layer`
3. Add build integration:
   - Each RUN command creates new upper layer
   - Final image is lower+upper merged
4. Add runtime integration:
   - Game runs with base + savegame overlay

## Implementation

### Core Module (`pkg/layers/overlay/overlay.go`)

- `OverlayFS` struct: lowerPaths, upperPath, workPath, mergedPath
- `Mount()` / `Unmount()` - mount/unmount overlay filesystem
- `LayerMerger` - handles layer merge and commit operations

### Build Integration (`pkg/builder/builder.go`)

- `createBuildOverlay()` - creates overlay for build process
- `runInstallOverlay()` - runs installer with overlay, commits result
- COPY instructions write directly to overlay upper layer

### Runtime Integration (`pkg/platforms/dosbox/dosbox.go`)

- `createRuntimeOverlay()` - creates overlay at runtime
- Base image (lower) + save directory (upper)
- Game runs with merged view; changes go to saveDir
- On exit, save files visible in `~/.retro/saves/game-name/`

## Notes

- Requires kernel support for overlayfs (Linux)
- May need fallback for macOS/Windows
- Important for Docker integration - images become layers
- Enables snapshot/restore functionality

## Status

In Progress