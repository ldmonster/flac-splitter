# Quick Start Guide

## Installation

### Step 1: Build the Program

```bash
# Clone or navigate to the repository
cd flac-splitter

# Build the program
make build
# OR
go build -o flac-splitter ./cmd/flac-splitter
```

### Step 2: Install External Tools (Optional)

**Only needed for Hybrid or External modes.** Pure Go mode (default) works without any external tools!

Choose **one** of the following options if you want to use `--hybrid` or `--external` flags:

#### Option A: Using Make (Easiest)
```bash
make install-deps
```

#### Option B: Manual Installation

**Ubuntu/Debian:**
```bash
sudo apt-get install shntool flac
# OR
sudo apt-get install ffmpeg
```

**macOS:**
```bash
brew install shntool flac
# OR
brew install ffmpeg
```

**Arch Linux:**
```bash
sudo pacman -S shntool flac
# OR
sudo pacman -S ffmpeg
```

## Running the Splitter

### Method 1: Pure Go Mode (Default - Recommended)
```bash
# No external tools needed!
./flac-splitter

# With custom output directory
./flac-splitter --output /path/to/output

# Quiet mode
./flac-splitter --quiet
```

### Method 2: Hybrid Mode (Fast + Safe)
```bash
# Requires shnsplit or ffmpeg
./flac-splitter --hybrid

# With ffmpeg preference
./flac-splitter --hybrid --ffmpeg
```

### Method 3: External Mode (Fastest)
```bash
# Requires shnsplit or ffmpeg
./flac-splitter --external

# With ffmpeg
./flac-splitter --external --ffmpeg
```

### Method 4: Using Make
```bash
make run
```

## Mode Selection Guide

### When to use Pure Go Mode (Default)
- ✅ You don't have external tools installed
- ✅ You want a simple, dependency-free solution
- ✅ You're okay with moderate processing speed
- ✅ You want guaranteed compatibility

### When to use Hybrid Mode (`--hybrid`)
- ✅ You have large collections to process
- ✅ You want detailed validation and track info
- ✅ You need fast processing with safety checks
- ✅ You have shnsplit or ffmpeg installed

### When to use External Mode (`--external`)
- ✅ You need maximum processing speed
- ✅ You have shnsplit or ffmpeg installed
- ✅ You trust your CUE and FLAC files are valid
- ✅ You're familiar with traditional FLAC splitting

## What To Expect

### Output Structure
```
split/
└── One/
    ├── Motoi Sakuraba - Tales of Xillia Original Soundtrack CD2/
    │   ├── 01 - The Sword That Dances Magnificently.flac
    │   ├── 02 - A Mecca of Battles.flac
    │   ├── 03 - Melee Dance!.flac
    │   └── ... (all tracks)
    │
    └── Motoi Sakuraba - Tales of Xillia Original Soundtrack CD3/
        └── ...
```

### Track Naming Format
```
<track_number> - <track_title>.flac
```

Examples:
- `01 - The Sword That Dances Magnificently.flac`
- `02 - A Mecca of Battles.flac`

## Common Issues & Solutions

### Issue: "FLAC file not found"
**Solution:**
- Make sure FLAC files are in the same directory as CUE files
- Check that filenames match exactly (case-sensitive on Linux)

### Issue: "Neither shnsplit nor ffmpeg found" (Hybrid/External mode)
**Solution:**
```bash
# Option 1: Use Pure Go mode instead (no external tools needed)
./flac-splitter

# Option 2: Install external tools
make install-deps
```

### Issue: "Permission denied"
**Solution:**
```bash
chmod +x flac-splitter
```

### Issue: Slow processing in Pure Go mode
**Solution:**
Use Hybrid mode for faster processing:
```bash
./flac-splitter --hybrid
```

## Performance Comparison

Processing time for a typical album (12 tracks, ~50 MB FLAC):

| Mode | Time | Dependencies | Notes |
|------|------|--------------|-------|
| Pure Go | ~90 seconds | None | Decodes and re-encodes |
| Hybrid | ~15 seconds | shnsplit/ffmpeg | Fast + validated |
| External | ~8 seconds | shnsplit/ffmpeg | Maximum speed |

*Times are approximate and depend on system performance*

## Advanced Usage

### Custom Output Directory
```bash
./flac-splitter --output /mnt/music/split
```

### Quiet Mode (Minimal Output)
```bash
./flac-splitter --quiet
```

###Verbose Mode (Detailed Progress)
```bash
./flac-splitter --verbose
```

### Combine Flags
```bash
./flac-splitter --hybrid --ffmpeg --output /tmp/music --quiet
```

## Cleaning Up

### Remove Only Output Directory
```bash
make clean-output
# Your original files remain untouched
```

### Remove Everything (Output + Executable)
```bash
make clean
```

## Important Notes

✅ Original FLAC and CUE files are **NOT** modified or deleted  
✅ Safe to re-run; will overwrite existing output  
✅ Supports Unicode filenames and CUE files  
✅ All metadata from CUE sheets is preserved  
✅ Pure Go mode works without ANY external dependencies

## Next Steps

1. Choose your mode (Pure Go is default and easiest)
2. Run the splitter: `./flac-splitter`
3. Check the `split/` directory
4. Verify the output tracks have correct metadata
5. Enjoy your split FLAC files!

For more details and technical information, see [README.md](README.md)
