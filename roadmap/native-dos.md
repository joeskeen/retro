# Native DOS Support

## Problem

Some games have bugs that only manifest in DosBox but work correctly in real DOS. For example, "Worlds of Billy" has a bug that only occurs in DosBox's emulation but is avoided when running on actual DOS within DosBox (booting into DOS from DosBox and then running the game).

## Requirements

- Ability to boot into actual DOS (not DosBox's emulation layer) for specific games
- Maintain compatibility with existing DosBox-based games
- Allow per-game configuration to specify which DOS mode to use

## Implementation Plan

1. Add `boot-method` field to platform configuration (`dosbox` vs `native-dos`)
2. Create DOS layer images that can be derived from base dosbox image
3. Support for common DOS versions (MS-DOS 6.22, etc.)
4. Handle the boot sequence: DOSBox BIOS → Boot into real DOS image → Run game

## Notes

- This is distinct from DosBox-X's emulated DOS
- May require different base images with actual DOS installed
- Games using this mode should be clearly marked in manifests

## Dependencies

Depends on: `derived-images` - Native DOS environments will be distributed as preset base images (e.g., `dosbox:msdos-6.22`)

## Status

Future