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

package cueparser

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// CueFile represents a CUE file with its metadata
type CueFile struct {
	Path         string
	RelativePath string
	FileName     string

	// Audio file information
	AudioFile     string // Main audio file (FLAC, WAV, etc.)
	AudioFileType string // WAVE, MP3, FLAC, etc.

	// Album metadata
	Album      string
	Performer  string
	Composer   string
	Songwriter string

	// Additional metadata
	Date    string
	Year    string
	Genre   string
	Comment string

	// Disc information
	Catalog    string
	DiscID     string
	DiscNumber string
	TotalDiscs string

	// Tracks
	Tracks []Track

	// Custom fields for any other metadata
	CustomFields map[string]string
}

// Track represents a single track in a CUE file
type Track struct {
	Number     int
	Title      string
	Performer  string
	Composer   string
	Songwriter string
	ISRC       string
	Index      string // Main index (01)
	PreGap     string // Index 00 if exists

	// Custom fields for track-specific metadata
	CustomFields map[string]string
}

// ParserConfig holds configuration for the CUE parser
type ParserConfig struct {
	// StrictMode causes parser to fail on malformed data
	StrictMode bool

	// PreserveEmptyFields keeps empty string values instead of nil
	PreserveEmptyFields bool

	// ParseCustomREM enables parsing of custom REM fields
	ParseCustomREM bool
}

// DefaultConfig returns a default parser configuration
func DefaultConfig() *ParserConfig {
	return &ParserConfig{
		StrictMode:          false,
		PreserveEmptyFields: false,
		ParseCustomREM:      true,
	}
}

// patterns holds compiled regex patterns for parsing CUE files
type patterns struct {
	file       *regexp.Regexp
	performer  *regexp.Regexp
	title      *regexp.Regexp
	composer   *regexp.Regexp
	songwriter *regexp.Regexp
	track      *regexp.Regexp
	index      *regexp.Regexp
	pregap     *regexp.Regexp
	isrc       *regexp.Regexp
	catalog    *regexp.Regexp

	// REM fields
	remDate       *regexp.Regexp
	remYear       *regexp.Regexp
	remGenre      *regexp.Regexp
	remComment    *regexp.Regexp
	remDiscID     *regexp.Regexp
	remDiscNumber *regexp.Regexp
	remCustom     *regexp.Regexp
}

// initPatterns initializes and compiles all regex patterns
func initPatterns() *patterns {
	return &patterns{
		file:       regexp.MustCompile(`FILE\s+"([^"]+)"\s+(\w+)`),
		performer:  regexp.MustCompile(`^\s*PERFORMER\s+"([^"]+)"`),
		title:      regexp.MustCompile(`^\s*TITLE\s+"([^"]+)"`),
		composer:   regexp.MustCompile(`^\s*COMPOSER\s+"([^"]+)"`),
		songwriter: regexp.MustCompile(`^\s*SONGWRITER\s+"([^"]+)"`),
		track:      regexp.MustCompile(`^\s*TRACK\s+(\d+)\s+AUDIO`),
		index:      regexp.MustCompile(`^\s*INDEX\s+01\s+(\d+:\d+:\d+)`),
		pregap:     regexp.MustCompile(`^\s*INDEX\s+00\s+(\d+:\d+:\d+)`),
		isrc:       regexp.MustCompile(`^\s*ISRC\s+([A-Z0-9]+)`),
		catalog:    regexp.MustCompile(`^\s*CATALOG\s+(\d+)`),

		remDate:       regexp.MustCompile(`^\s*REM\s+DATE\s+(\d{4}(?:-\d{2}-\d{2})?)`),
		remYear:       regexp.MustCompile(`^\s*REM\s+YEAR\s+(\d{4})`),
		remGenre:      regexp.MustCompile(`^\s*REM\s+GENRE\s+(.+)$`),
		remComment:    regexp.MustCompile(`^\s*REM\s+COMMENT\s+(.+)$`),
		remDiscID:     regexp.MustCompile(`^\s*REM\s+DISCID\s+([A-Fa-f0-9]+)`),
		remDiscNumber: regexp.MustCompile(`^\s*REM\s+DISC(?:NUMBER)?\s+(\d+)(?:/(\d+))?`),
		remCustom:     regexp.MustCompile(`^\s*REM\s+([A-Z_][A-Z0-9_]*)\s+(.+)$`),
	}
}

// FindAll recursively finds all .cue files in the given directory
func FindAll(rootPath string, skipDirs ...string) ([]CueFile, error) {
	var cueFiles []CueFile
	skipMap := make(map[string]bool)
	for _, dir := range skipDirs {
		skipMap[dir] = true
	}

	err := filepath.Walk(rootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		// Skip directories in skipMap
		for skipDir := range skipMap {
			if strings.Contains(path, skipDir) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}
		}

		// Skip hidden directories and .dist folder (but not the root path)
		if info.IsDir() && path != rootPath && path != "." {
			if strings.HasPrefix(info.Name(), ".") || info.Name() == ".dist" {
				return filepath.SkipDir
			}
		}

		// Check if it's a CUE file
		if !info.IsDir() && strings.ToLower(filepath.Ext(path)) == ".cue" {
			relPath, _ := filepath.Rel(rootPath, path)
			cueFiles = append(cueFiles, CueFile{
				Path:         path,
				RelativePath: relPath,
				FileName:     info.Name(),
			})
		}

		return nil
	})

	return cueFiles, err
}

// Parse parses a CUE file and extracts metadata using default configuration
func Parse(cue *CueFile) error {
	return ParseWithConfig(cue, DefaultConfig())
}

// ParseWithConfig parses a CUE file with custom configuration
func ParseWithConfig(cue *CueFile, config *ParserConfig) error {
	file, err := os.Open(cue.Path)
	if err != nil {
		return fmt.Errorf("failed to open CUE file: %w", err)
	}
	defer file.Close()

	// Initialize custom fields maps
	if cue.CustomFields == nil {
		cue.CustomFields = make(map[string]string)
	}

	scanner := bufio.NewScanner(file)
	pat := initPatterns()

	var currentTrack *Track
	albumSet := false
	albumPerformer := ""
	lineNum := 0

	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Parse FILE line
		if matches := pat.file.FindStringSubmatch(line); matches != nil {
			cue.AudioFile = matches[1]
			if len(matches) > 2 {
				cue.AudioFileType = matches[2]
			}
			continue
		}

		// Parse CATALOG
		if matches := pat.catalog.FindStringSubmatch(line); matches != nil {
			cue.Catalog = matches[1]
			continue
		}

		// Parse REM fields
		if strings.HasPrefix(strings.TrimSpace(line), "REM") {
			if err := parseREMField(line, cue, config, pat); err != nil && config.StrictMode {
				return fmt.Errorf("line %d: %w", lineNum, err)
			}
			continue
		}

		// Parse PERFORMER (album or track level)
		if matches := pat.performer.FindStringSubmatch(line); matches != nil {
			if currentTrack == nil {
				if albumPerformer == "" {
					albumPerformer = matches[1]
					cue.Performer = matches[1]
				}
			} else {
				currentTrack.Performer = matches[1]
			}
			continue
		}

		// Parse COMPOSER (album or track level)
		if matches := pat.composer.FindStringSubmatch(line); matches != nil {
			if currentTrack == nil {
				cue.Composer = matches[1]
			} else {
				currentTrack.Composer = matches[1]
			}
			continue
		}

		// Parse SONGWRITER (album or track level)
		if matches := pat.songwriter.FindStringSubmatch(line); matches != nil {
			if currentTrack == nil {
				cue.Songwriter = matches[1]
			} else {
				currentTrack.Songwriter = matches[1]
			}
			continue
		}

		// Parse TITLE (album or track title)
		if matches := pat.title.FindStringSubmatch(line); matches != nil {
			if !albumSet && currentTrack == nil {
				cue.Album = matches[1]
				albumSet = true
			} else if currentTrack != nil {
				currentTrack.Title = matches[1]
			}
			continue
		}

		// Parse TRACK
		if matches := pat.track.FindStringSubmatch(line); matches != nil {
			// Save previous track if exists
			if currentTrack != nil {
				fillTrackDefaults(currentTrack, albumPerformer)
				cue.Tracks = append(cue.Tracks, *currentTrack)
			}
			// Create new track
			currentTrack = &Track{
				Number:       len(cue.Tracks) + 1,
				CustomFields: make(map[string]string),
			}
			continue
		}

		// Parse track-specific fields
		if currentTrack != nil {
			// INDEX 01
			if matches := pat.index.FindStringSubmatch(line); matches != nil {
				currentTrack.Index = matches[1]
				continue
			}

			// INDEX 00 (pregap)
			if matches := pat.pregap.FindStringSubmatch(line); matches != nil {
				currentTrack.PreGap = matches[1]
				continue
			}

			// ISRC
			if matches := pat.isrc.FindStringSubmatch(line); matches != nil {
				currentTrack.ISRC = matches[1]
				continue
			}
		}
	}

	// Add the last track
	if currentTrack != nil {
		fillTrackDefaults(currentTrack, albumPerformer)
		cue.Tracks = append(cue.Tracks, *currentTrack)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("error reading CUE file: %w", err)
	}

	// Validate parsed data
	if config.StrictMode {
		if err := cue.Validate(); err != nil {
			return fmt.Errorf("validation failed: %w", err)
		}
	}

	return nil
}

// parseREMField parses REM (remark) fields
func parseREMField(line string, cue *CueFile, config *ParserConfig, pat *patterns) error {
	line = strings.TrimSpace(line)

	// DATE
	if matches := pat.remDate.FindStringSubmatch(line); matches != nil {
		cue.Date = matches[1]
		// Extract year from date if it's in YYYY format or YYYY-MM-DD
		if len(matches[1]) >= 4 {
			cue.Year = matches[1][:4]
		}
		return nil
	}

	// YEAR
	if matches := pat.remYear.FindStringSubmatch(line); matches != nil {
		cue.Year = matches[1]
		if cue.Date == "" {
			cue.Date = matches[1]
		}
		return nil
	}

	// GENRE
	if matches := pat.remGenre.FindStringSubmatch(line); matches != nil {
		cue.Genre = strings.TrimSpace(matches[1])
		return nil
	}

	// COMMENT
	if matches := pat.remComment.FindStringSubmatch(line); matches != nil {
		cue.Comment = strings.TrimSpace(matches[1])
		return nil
	}

	// DISCID
	if matches := pat.remDiscID.FindStringSubmatch(line); matches != nil {
		cue.DiscID = strings.TrimSpace(matches[1])
		return nil
	}

	// DISCNUMBER / DISC
	if matches := pat.remDiscNumber.FindStringSubmatch(line); matches != nil {
		cue.DiscNumber = matches[1]
		if len(matches) > 2 && matches[2] != "" {
			cue.TotalDiscs = matches[2]
		}
		return nil
	}

	// Custom REM fields
	if config.ParseCustomREM {
		if matches := pat.remCustom.FindStringSubmatch(line); matches != nil {
			key := strings.ToUpper(strings.TrimSpace(matches[1]))
			value := strings.TrimSpace(matches[2])

			// Skip already parsed fields
			knownFields := map[string]bool{
				"DATE": true, "YEAR": true, "GENRE": true, "COMMENT": true,
				"DISCID": true, "DISCNUMBER": true, "DISC": true,
			}

			if !knownFields[key] {
				cue.CustomFields[key] = value
			}
		}
	}

	return nil
}

// fillTrackDefaults sets default values for track fields
func fillTrackDefaults(track *Track, albumPerformer string) {
	if track.Performer == "" {
		track.Performer = albumPerformer
	}
}

// Validate checks if the CueFile has required fields
func (c *CueFile) Validate() error {
	if c.AudioFile == "" {
		return fmt.Errorf("missing FILE directive")
	}
	if c.Album == "" {
		return fmt.Errorf("missing album TITLE")
	}
	if len(c.Tracks) == 0 {
		return fmt.Errorf("no tracks found")
	}

	for i, track := range c.Tracks {
		if track.Title == "" {
			return fmt.Errorf("track %d missing TITLE", i+1)
		}
		if track.Index == "" {
			return fmt.Errorf("track %d missing INDEX 01", i+1)
		}
	}

	return nil
}

// GetAudioFilePath returns the full path to the audio file
func (c *CueFile) GetAudioFilePath() string {
	if c.AudioFile == "" {
		return ""
	}

	// If AudioFile is absolute, return it
	if filepath.IsAbs(c.AudioFile) {
		return c.AudioFile
	}

	// Otherwise, join with CUE file directory
	return filepath.Join(filepath.Dir(c.Path), c.AudioFile)
}

// HasCustomField checks if a custom field exists
func (c *CueFile) HasCustomField(key string) bool {
	_, exists := c.CustomFields[strings.ToUpper(key)]
	return exists
}

// GetCustomField retrieves a custom field value
func (c *CueFile) GetCustomField(key string) string {
	return c.CustomFields[strings.ToUpper(key)]
}

// TrackCount returns the number of tracks
func (c *CueFile) TrackCount() int {
	return len(c.Tracks)
}

// GetTrack returns a track by number (1-based)
func (c *CueFile) GetTrack(number int) *Track {
	if number < 1 || number > len(c.Tracks) {
		return nil
	}
	return &c.Tracks[number-1]
}

// HasCustomField checks if a track has a custom field
func (t *Track) HasCustomField(key string) bool {
	_, exists := t.CustomFields[strings.ToUpper(key)]
	return exists
}

// GetCustomField retrieves a custom field value from a track
func (t *Track) GetCustomField(key string) string {
	return t.CustomFields[strings.ToUpper(key)]
}
