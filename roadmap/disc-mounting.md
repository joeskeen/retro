# Disc Mounting (Multi-Mode BIN/CUE)

## Problem

Some games require physical CDs to be mounted at install and/or runtime. These CDs may be multi-mode (e.g., Music + Data) stored as BIN/CUE files.

## Requirements

- Mount disc images (ISO, BIN/CUE) at build time (install phase)
- Mount disc images at runtime
- Support BIN/CUE multi-session discs
- Handle disc swap requests during gameplay
- Track which discs are needed for which game phases

## Implementation Plan

1. Add `mounts` section to Retrofile:
   ```dockerfile
   MOUNT_DISC install:C:\games\disc1.cue
   MOUNT_DISC runtime:C:\games\disc2.iso
   ```
2. Create disc mounting layer in platform abstraction
3. Implement disc swap commands that can be triggered by game scripts
4. Store disc images as separate layers (content-addressable)

## Notes

- BIN/CUE files should be kept together as a pair
- Multi-mode discs need special handling to present both Music and Data modes
- Some games need disc swaps at specific points (install vs gameplay)

## Dependencies

Depends on: `overlayfs` - Disc images stored as layers and mounted via overlayfs

Future