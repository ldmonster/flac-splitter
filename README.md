
# FLAC Splitter from CUE Files

A powerful Go program that automatically splits FLAC audio files based on CUE sheet information with comprehensive metadata tagging.

## Features

- 🔍 **Automatic CUE File Discovery**: Recursively finds all `.cue` files in the workspace
- 📁 **Organized Output Structure**: Creates a well-organized directory structure in `split/` folder
- 🎵 **Three Splitting Modes**: Pure Go (default), Hybrid, or External tools
- 🏷️ **Comprehensive Metadata Tagging**: Preserves all album and track information using go-flac
- 📊 **Detailed Logging**: Provides clear progress updates with track information
- 🌍 **Unicode Support**: Handles CUE files with international characters
- ✅ **Error Handling**: Robust error handling with detailed status reporting
- 🚀 **No External Dependencies**: Default mode works without any external tools

## Splitting Modes

### Pure Go Mode (Default)
- **No external tools required**
- Decodes, splits, and re-encodes FLAC files using pure Go libraries
- Full metadata preservation with go-flac
- Works on any system with Go installed
- Moderate speed, perfect quality

### Hybrid Mode (`--hybrid`)
- Go-based FLAC validation + external tools for splitting
- Fast splitting with comprehensive safety checks
- Displays detailed track timing information
- Requires shnsplit or ffmpeg
- **Recommended for large collections**

### External Mode (`--external`)
- Uses only shnsplit or ffmpeg
- Fastest option available
- Requires external tools
- Traditional approach

## Prerequisites

### For Pure Go Mode (Default)
- **NothingREADME.md.bak README.md && mv QUICKSTART.md.bak QUICKSTART.md* Just build and run
- Works out of the box with Go installed

### For Hybrid/External Modes (Optional)

Choose **one** of the following:

#### Option 1: shnsplit (Recommended for speed)
```sh
# Ubuntu/Debian
sudo apt-get install shntool flac

# macOS with Homebrew
brew install shntool flac

# Arch Linux
sudo pacman -S shntool flac
```

#### Option 2: ffmpeg
```sh
# Ubuntu/Debian
sudo apt-get install ffmpeg

# macOS with Homebrew
brew install ffmpeg

# Arch Linux
sudo pacman -S ffmpeg
```

## Quick Start

```sh
# Build the program
make build

# Run with Pure Go mode (default - no external tools needed)
./flac-splitter

# Run with Hybrid mode (Go validation + external tools)
./flac-splitter --hybrid

# Run with External mode (fastest, requires shnsplit/ffmpeg)
./flac-splitter --external

# Prefer ffmpeg in external/hybrid mode
./flac-splitter --external --ffmpeg

# Custom output directory
./flac-splitter --output /path/to/output

# Or use make commands
make run
```

## How It Works

The program performs the following steps:

1. **Find all CUE files**: Recursively searches the workspace for `.cue` files
2. **Parse metadata**: Extracts album and track information from CUE sheets
3. **Create output structure**: Creates organized directory structure in `split/`
4. **Split FLAC files**: Decodes and splits tracks (method depends on mode)
5. **Apply metadata**: Writes comprehensive tags using go-flac library:
   - Track title, artist, album, performer
   - Track number, total tracks
   - Date, genre, catalog, disc ID
   - All custom CUE fields

## Output Structure

```
split/
├── One/
│   ├── Motoi Sakuraba - Tales of Xillia Original Soundtrack CD2/
│   │   ├── 01 - The Sword That Dances Magnificently.flac
│   │   ├── 02 - A Mecca of Battles.flac
│   │   └── ...
│   └── Motoi Sakuraba - Tales of Xillia Original Soundtrack CD3/
│       └── ...
└── ...
```

## Command-Line Options

```sh
./flac-splitter [flags]

Flags:
  --external        Use external tools only (shnsplit/ffmpeg) - fastest
  --hybrid          Hybrid mode: Go validation + external splitting
  --ffmpeg          Prefer ffmpeg over shnsplit (for external/hybrid)
  -o, --output      Output directory (default: "split")
  -q, --quiet       Quiet mode - only errors and summary
  -v, --verbose     Verbose mode - detailed progress
  -h, --help        Show help message
```

## Makefile Commands

```sh
make build         # Build the FLAC splitter
make run           # Build and run the FLAC splitter
make clean         # Remove build artifacts and output directory
make clean-output  # Remove only the split output directory
make install-deps  # Install external dependencies (for hybrid/external modes)
make help          # Show help message
```

## Troubleshooting

### "FLAC file not found" error
- Ensure the FLAC file referenced in the CUE file exists in the same directory
- Check that the filename matches (case-sensitive on Linux)

### "Neither shnsplit nor ffmpeg found" error (Hybrid/External mode)
**Solution:** Use Pure Go mode (default) or install external tools:
```sh
# Use Pure Go mode (no external tools needed)
./flac-splitter

# Or install external tools for hybrid/external mode
make install-deps
```

### Failed splitting
- Check FLAC file integrity: `flac -t yourfile.flac`
- Ensure sufficient disk space
- Verify CUE file format is valid

## Technical Details

### Splitting Modes Comparison

| Mode | Speed | Dependencies | Quality | Use Case |
|------|-------|--------------|---------|----------|
| **Pure Go** | Moderate | None | Perfect | Default, no external tools |
| **Hybrid** | Fast | shnsplit/ffmpeg | Perfect | Large collections, with validation |
| **External** | Fastest | shnsplit/ffmpeg | Perfect | Maximum speed |

### Metadata Tagging

All modes use **go-flac** library for comprehensive metadata:
- Standard tags: TITLE, ARTIST, ALBUM, PERFORMER, DATE, GENRE
- Track numbering: TRACKNUMBER, TOTALTRACKS
- Extended: CATALOG, DISCID, DESCRIPTION
- Custom: Any additional CUE fields preserved

### Architecture

The codebase is organized into three main components:

- `internal/cueparser/` - Generic CUE sheet parser
- `internal/flacsplitter/`
  - `splitter.go` - Core types and mode router
  - `splitter_audio.go` - Pure Go and hybrid implementations
  - `splitter_external.go` - External tools implementation
- `cmd/flac-splitter/` - CLI with Cobra framework

## Performance

Typical speeds per album:
- **Pure Go**: 1-2 minutes (moderate, no dependencies)
- **Hybrid**: 10-30 seconds (fast + validation)
- **External**: 5-15 seconds (fastest)

## License

This project is licensed under the Apache License 2.0 - see the [LICENSE](LICENSE) file for details.

