# Apple ][ Emulation

## Problem

Apple ][ has a rich library of classic games that deserve preservation and easy running. AppleWin on Windows and other emulators on other platforms can provide excellent compatibility.

## Requirements

- Support Apple ][ platform via AppleWin (or cross-platform equivalent)
- Handle disk swap for multi-disk games
- Support both 5.25" and 3.5" disk images
- Support cassette-based games
- Joystick/keyboard support

## Implementation Plan

1. Add `applewin` (or `apple2`) platform:
   ```dockerfile
   TAG apple2-game:1.0
   FROM apple2:latest
   COPY game.po C:\games
   ENTRYPOINT C:\games\game.po
   ```
2. Implement disk management:
   - .dsk, .do, .po disk image support
   - Disk swap commands
   - Create disk set management for multi-disk games
3. Add Apple II specific features:
   - Joystick emulation
   - Keyboard input handling
   - Color monitor emulation

## Notes

- AppleWin is Windows-only; cross-platform needs evaluation
- MAME supports Apple II but may be overkill
- Consider `复仇者` (Avengers) and other favorites as test cases

## Status

Future