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

package flacsplitter

import (
	"fmt"
	"strings"

	"github.com/ldmonster/flac-splitter/internal/cueparser"
)

// SplitMode defines the splitting implementation to use
type SplitMode int

const (
	// ModeExternalTools uses shnsplit or ffmpeg (fast, reliable)
	ModeExternalTools SplitMode = iota
	// ModeGoAudio uses pure Go libraries with validation (hybrid approach)
	ModeGoAudio
	// ModeGoAudioFull uses pure Go FLAC encoding (no external tools)
	ModeGoAudioFull
)

// SplitOptions holds configuration for FLAC splitting
type SplitOptions struct {
	OutputDir       string
	FilenamePattern string // e.g., "%02d - %s.flac"
	OverwriteFiles  bool
	UseFFmpeg       bool      // Prefer ffmpeg over shnsplit (only for external mode)
	Mode            SplitMode // Which splitter implementation to use
}

// DefaultOptions returns default split options
func DefaultOptions(outputDir string) *SplitOptions {
	return &SplitOptions{
		OutputDir:       outputDir,
		FilenamePattern: "%02d - %s.flac",
		OverwriteFiles:  true,
		UseFFmpeg:       false,
		Mode:            ModeGoAudioFull,
	}
}

// Split splits a FLAC file based on CUE sheet using the configured mode
func Split(cue cueparser.CueFile, flacPath string, opts *SplitOptions) error {
	switch opts.Mode {
	case ModeGoAudio:
		// Hybrid: Go validation + external tools for splitting
		return SplitWithGoAudioSimple(cue, flacPath, opts)

	case ModeGoAudioFull:
		// Pure Go: decode, split, and re-encode with Go libraries
		return SplitWithGoAudio(cue, flacPath, opts)

	case ModeExternalTools:
		// External tools only (shnsplit or ffmpeg)
		return splitWithExternalTools(cue, flacPath, opts)

	default:
		return fmt.Errorf("unknown split mode: %d", opts.Mode)
	}
}

// sanitizeFilename removes or replaces invalid characters from filenames
func sanitizeFilename(name string) string {
	// Replace invalid characters
	invalid := []string{"/", "\\", ":", "*", "?", "\"", "<", ">", "|"}
	result := name
	for _, char := range invalid {
		result = strings.ReplaceAll(result, char, "_")
	}
	return strings.TrimSpace(result)
}

// convertCueTimeToSeconds converts CUE time format (MM:SS:FF) to seconds string
func convertCueTimeToSeconds(cueTime string) string {
	parts := strings.Split(cueTime, ":")
	if len(parts) != 3 {
		return "0"
	}

	var minutes, seconds, frames int
	fmt.Sscanf(parts[0], "%d", &minutes)
	fmt.Sscanf(parts[1], "%d", &seconds)
	fmt.Sscanf(parts[2], "%d", &frames)

	totalSeconds := float64(minutes*60) + float64(seconds) + float64(frames)/75.0
	return fmt.Sprintf("%.3f", totalSeconds)
}

// calculateDuration calculates the duration between two CUE timestamps
func calculateDuration(start, end string) string {
	startSec := convertCueTimeToSeconds(start)
	endSec := convertCueTimeToSeconds(end)

	var startFloat, endFloat float64
	fmt.Sscanf(startSec, "%f", &startFloat)
	fmt.Sscanf(endSec, "%f", &endFloat)

	duration := endFloat - startFloat
	if duration <= 0 {
		return ""
	}

	return fmt.Sprintf("%.3f", duration)
}
