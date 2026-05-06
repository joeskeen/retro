# Image Inheritance (Derived Images)

## Problem

Currently, the only true base image is `dosbox`. Some games require pre-configured environments like `dosbox+soundblaster` or `dosbox+gravis ultrasound`. Creating these from scratch for each game is inefficient.

## Requirements

- Base images can be derived from other images
- Inheritance chain is traceable and auditable
- Images can override or extend parent configurations
- Tags should reflect inheritance (e.g., `dosbox:1.0-soundblaster`)

## Implementation Plan

1. Add `FROM` instruction already exists in Retrofile spec - ensure it's properly implemented
2. Implement image layering in registry:
   - Base images stored once
   - Derived images store only delta
3. Create preset images:
   - `dosbox:soundblaster` - DOSBox with Sound Blaster 16 config
   - `dosbox:gravis` - DOSBox with Gravis Ultrasound config
   - `dosbox:general-midi` - DOSBox with General MIDI
4. Add `inherit` command to inspect image ancestry

## Notes

- Content-addressable layers make this storage-efficient
- Inheritance should be limited to 2-3 levels to avoid complexity
- Presets should be well-documented with their configurations

## Dependencies

Depends on: `overlayfs` - Efficient layer storage and merge operations

Future