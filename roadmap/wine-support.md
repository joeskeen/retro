# Wine Support (Windows Games)

## Problem

Many classic Windows games (1990s-2000s) can run via Wine. While DosBox handles DOS games, we need a path for Windows games that don't require the full DosBox-X Windows 98SE environment.

## Requirements

- Run Windows games via Wine
- Handle Windows DLL dependencies where needed
- Configure Wine prefix per-game
- Support for Windows games that work well in Wine
- Fallback documentation for games that don't work

## Implementation Plan

1. Create `wine` platform in platforms abstraction:
   ```dockerfile
   TAG windows-game:1.0
   FROM wine:latest
   COPY installer C:\temp
   RUN wine C:\temp\setup.exe
   COPY saves C:\saves
   ENTRYPOINT wine C:\games\game.exe
   ```
2. Implement Wine layer with:
   - Wineprefix management per game
   - Built-in Windows DLLs (optional)
   - Graphics/audio configuration
3. Add `wine` commands:
   - `retro wine init` - Create new Wine prefix
   - `retro wine configure` - Configure for specific game
   - `retro wine dxvnc` - Enable DXVA/null drivers

## Notes

- Start with games known to work well in Wine
- Wine configs can get complex - document good defaults
- Consider using `winetricks` for dependency management
- Some games may need `windowsxp` or `windows7` prefix mode

## Status

Future