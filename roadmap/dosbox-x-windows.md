# DosBox-X Support (Windows 98SE)

## Problem

DOSBox-X can emulate Windows 98SE, opening the door to Windows games from that era. However, DosBox-X works differently from standard DosBox - it converts folder mounts to image files once booted, which breaks the expected workflow for savegames and data persistence.

## Requirements

- Boot into Windows 98SE environment via DosBox-X
- Maintain external storage for savegames (outside the booted image)
- Handle the mount-to-image conversion that occurs during boot
- Provide seamless access to game files before and after boot
- Support save state backup and transfer

## Challenges

1. **Mount Conversion**: When DosBox-X boots, it converts folder mounts to internal image files. After boot, external folders are no longer accessible at the same paths.

2. **Savegame Management**: Savegames created inside the Windows 98SE environment need to be extracted/exported to remain accessible.

3. **State Persistence**: The booted image contains all changes made during session - need to track which changes are "owned" by the user vs the base image.

## Implementation Plan

1. Add `dosbox-x` platform variant with Windows 98SE support:
   ```dockerfile
   TAG windows-game:1.0
   FROM dosbox-x:98se
   COPY game-installer C:\temp
   ENTRYPOINT C:\temp\setup.exe
   ```
2. Implement overlay filesystem for external folders:
   - Before boot: normal folder mount
   - After boot: overlay layer maps external folder to internal path
3. Add savegame export commands
4. Create `retro snapshot` command to capture post-boot state
5. Implement `retro export-saves` to extract savegame directories

## Notes

- This is a significant architectural change
- DosBox-X can also run older DOS games with better compatibility
- May want separate image tags: `dosbox-x:dos`, `dosbox-x:98se`

## Status

Future