// Copyright 2026 ldmonster
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/ldmonster/flac-splitter/internal/cueparser"
	"github.com/ldmonster/flac-splitter/internal/flacsplitter"
	"github.com/spf13/cobra"
)

var (
	// Global flags
	externalMode bool
	hybridMode   bool
	useFFmpeg    bool
	outputDir    string
	quiet        bool
	verbose      bool
)

const (
	defaultOutputDir = "split"
)

var rootCmd = &cobra.Command{
	Use:   "flac-splitter [flags]",
	Short: "Split FLAC files based on CUE sheets with comprehensive metadata tagging",
	Long: `FLAC Splitter - A powerful tool for splitting large FLAC audio files into individual tracks

This tool recursively searches for CUE sheet files in the current directory and splits 
associated FLAC files into individual tracks. It preserves all metadata including album 
information, track titles, artists, and more using the go-flac library.

Features:
  • Automatic CUE file discovery
  • Support for both shnsplit and ffmpeg backends
  • Comprehensive metadata tagging with go-flac
  • Default FLAC validation with pure Go libraries (detailed track information)
  • Organized output directory structure: split/relative-path/album-name/

Modes:
  • Pure Go (default): Decode, split, and re-encode FLAC with pure Go libraries
  • Hybrid (--hybrid): Validate with Go, split with external tools (fast + safe)
  • External (--external): Use only external tools - shnsplit or ffmpeg

Supported Tools:
  External modes can use either shnsplit or ffmpeg. Use --ffmpeg to prefer ffmpeg.`,
	Example: `  # Pure Go splitting (default - no external tools needed)
  flac-splitter

  # Hybrid mode: Go validation + external tools (recommended for speed)
  flac-splitter --hybrid

  # External tools only (fastest, requires shnsplit or ffmpeg)
  flac-splitter --external

  # External mode with ffmpeg preference
  flac-splitter --external --ffmpeg

  # Specify custom output directory
  flac-splitter --output /path/to/output

  # Quiet mode (minimal output)
  flac-splitter --quiet

  # Verbose mode (detailed progress)
  flac-splitter --verbose`,
	Run: runSplitter,
}

func init() {
	rootCmd.Flags().BoolVar(&externalMode, "external", false,
		"Use external tools only (shnsplit/ffmpeg) - fastest")
	rootCmd.Flags().BoolVar(&hybridMode, "hybrid", false,
		"Hybrid mode: Go validation + external splitting (fast + safe)")
	rootCmd.Flags().BoolVar(&useFFmpeg, "ffmpeg", false,
		"Prefer ffmpeg over shnsplit (for external/hybrid modes)")
	rootCmd.Flags().StringVarP(&outputDir, "output", "o", defaultOutputDir,
		"Output directory for split files")
	rootCmd.Flags().BoolVarP(&quiet, "quiet", "q", false,
		"Quiet mode - only show errors and summary")
	rootCmd.Flags().BoolVarP(&verbose, "verbose", "v", false,
		"Verbose mode - show detailed processing information")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func runSplitter(cmd *cobra.Command, args []string) {
	if !quiet {
		log.Println("=== FLAC Splitter from CUE files ===")
	}

	// Determine splitting mode
	var mode flacsplitter.SplitMode
	var modeDesc string

	if externalMode && hybridMode {
		log.Fatal("Error: Cannot use both --external and --hybrid flags")
	}

	if externalMode {
		mode = flacsplitter.ModeExternalTools
		modeDesc = "External tools only (shnsplit/ffmpeg)"
	} else if hybridMode {
		mode = flacsplitter.ModeGoAudio
		modeDesc = "Hybrid (Go validation + external tools)"
	} else {
		mode = flacsplitter.ModeGoAudioFull
		modeDesc = "Pure Go (decode + split + encode)"
	}

	if !quiet {
		log.Printf("Mode: %s", modeDesc)
	}

	// Step 1: Find all CUE files
	if !quiet {
		log.Println("Step 1: Finding all CUE files...")
	}
	cueFiles, err := cueparser.FindAll(".", outputDir)
	if err != nil {
		log.Fatalf("Error finding CUE files: %v", err)
	}

	if len(cueFiles) == 0 {
		log.Println("No CUE files found in current directory")
		return
	}

	if !quiet {
		log.Printf("Found %d CUE file(s)\n", len(cueFiles))
	}

	// Step 2: Create output directory
	if verbose {
		log.Printf("Creating output directory: %s", outputDir)
	}
	if err := os.MkdirAll(outputDir, 0755); err != nil {
		log.Fatalf("Error creating output directory: %v", err)
	}

	// Step 3 & 4: Process each CUE file
	if !quiet {
		log.Println("Processing CUE files and splitting FLAC files...")
	}

	successCount := 0
	failureCount := 0
	skippedCount := 0

	for i, cue := range cueFiles {
		if !quiet {
			fmt.Printf("\n[%d/%d] Processing: %s\n", i+1, len(cueFiles), cue.Path)
		}

		// Parse CUE file
		if err := cueparser.Parse(&cue); err != nil {
			log.Printf("  ✗ Error parsing CUE file: %v", err)
			failureCount++
			continue
		}

		// Check if FLAC file exists
		flacPath := cue.GetAudioFilePath()
		if _, err := os.Stat(flacPath); os.IsNotExist(err) {
			if verbose || !quiet {
				log.Printf("  ⊘ Skipped: FLAC file not found: %s", flacPath)
			}
			skippedCount++
			continue
		}

		// Create output directory structure
		trackOutputDir, err := createOutputDirectory(cue, outputDir)
		if err != nil {
			log.Printf("  ✗ Error creating output directory: %v", err)
			failureCount++
			continue
		}

		// Split FLAC file using the splitter package
		if verbose {
			log.Printf("  Splitting FLAC file: %s", cue.AudioFile)
			log.Printf("  Output directory: %s", trackOutputDir)
			log.Printf("  Number of tracks: %d", cue.TrackCount())
		}

		opts := flacsplitter.DefaultOptions(trackOutputDir)
		opts.Mode = mode
		opts.UseFFmpeg = useFFmpeg

		if err := flacsplitter.Split(cue, flacPath, opts); err != nil {
			log.Printf("  ✗ Error splitting FLAC file: %v", err)
			failureCount++
			continue
		}

		if !quiet {
			log.Printf("  ✓ Successfully processed")
		}
		successCount++
	}

	// Summary
	fmt.Println("\n=== Summary ===")
	fmt.Printf("Total CUE files found: %d\n", len(cueFiles))
	fmt.Printf("Successfully processed: %d\n", successCount)
	if skippedCount > 0 {
		fmt.Printf("Skipped (no FLAC file): %d\n", skippedCount)
	}
	if failureCount > 0 {
		fmt.Printf("Failed (errors): %d\n", failureCount)
	}
	fmt.Printf("\nOutput directory: %s\n", outputDir)

	if failureCount > 0 {
		os.Exit(1)
	}
}

// createOutputDirectory creates the output directory structure
func createOutputDirectory(cue cueparser.CueFile, baseOutputDir string) (string, error) {
	// Get the parent directory of the CUE file (relative to current dir)
	relDir := filepath.Dir(cue.RelativePath)

	// Create base path: baseOutputDir/RelativeDir/CueName
	cueName := strings.TrimSuffix(cue.FileName, filepath.Ext(cue.FileName))
	trackOutputDir := filepath.Join(baseOutputDir, relDir, cueName)

	if err := os.MkdirAll(trackOutputDir, 0755); err != nil {
		return "", err
	}

	return trackOutputDir, nil
}
