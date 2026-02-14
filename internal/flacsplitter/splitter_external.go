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
	"bufio"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"

	"github.com/go-flac/flacvorbis"
	"github.com/go-flac/go-flac"
	"github.com/ldmonster/flac-splitter/internal/cueparser"
)

// splitWithExternalTools uses shnsplit or ffmpeg for splitting
func splitWithExternalTools(cue cueparser.CueFile, flacPath string, opts *SplitOptions) error {
	// Check which tool is available
	hasShnsplit := executableExists("shnsplit")
	hasFFmpeg := executableExists("ffmpeg")

	if !hasShnsplit && !hasFFmpeg {
		return fmt.Errorf("neither shnsplit nor ffmpeg found - please install one of them")
	}

	// Choose splitter
	if opts.UseFFmpeg && hasFFmpeg {
		return splitWithFFmpeg(cue, flacPath, opts)
	} else if hasShnsplit {
		return splitWithShnsplit(cue, flacPath, opts)
	} else if hasFFmpeg {
		return splitWithFFmpeg(cue, flacPath, opts)
	}

	return fmt.Errorf("no suitable audio splitter found")
}

// executableExists checks if a command is available in PATH
func executableExists(name string) bool {
	_, err := exec.LookPath(name)
	return err == nil
}

// splitWithShnsplit uses shnsplit to split the FLAC file
func splitWithShnsplit(cue cueparser.CueFile, flacPath string, opts *SplitOptions) error {
	// Create temporary CUE file with absolute path
	tempCuePath := filepath.Join(opts.OutputDir, "temp.cue")
	if err := copyCueFile(cue.Path, tempCuePath, flacPath); err != nil {
		return fmt.Errorf("failed to create temporary CUE file: %v", err)
	}
	defer os.Remove(tempCuePath)

	// Run shnsplit
	cmd := exec.Command("shnsplit",
		"-f", tempCuePath,
		"-t", "%n - %t",
		"-o", "flac",
		"-d", opts.OutputDir,
		flacPath,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("shnsplit failed: %v\nOutput: %s", err, string(output))
	}

	log.Printf("  Split complete with shnsplit")

	// Apply metadata tags using go-flac
	return applyMetadataTags(cue, opts)
}

// splitWithFFmpeg uses ffmpeg to split the FLAC file
func splitWithFFmpeg(cue cueparser.CueFile, flacPath string, opts *SplitOptions) error {
	for i, track := range cue.Tracks {
		// Calculate start time
		startTime := convertCueTimeToSeconds(track.Index)

		// Calculate duration
		var duration string
		if i < len(cue.Tracks)-1 {
			duration = calculateDuration(track.Index, cue.Tracks[i+1].Index)
		}

		// Output filename
		outputFile := filepath.Join(opts.OutputDir,
			fmt.Sprintf(opts.FilenamePattern, track.Number, sanitizeFilename(track.Title)))

		// Build ffmpeg command - use copy codec for speed
		args := []string{
			"-i", flacPath,
			"-ss", startTime,
		}

		if duration != "" {
			args = append(args, "-t", duration)
		}

		args = append(args, "-acodec", "copy")

		if opts.OverwriteFiles {
			args = append(args, "-y")
		}

		args = append(args, outputFile)

		cmd := exec.Command("ffmpeg", args...)
		if output, err := cmd.CombinedOutput(); err != nil {
			log.Printf("  Warning: Failed to extract track %d: %v", track.Number, err)
			log.Printf("  FFmpeg output: %s", string(output))
			continue
		}
	}

	log.Printf("  Split complete with ffmpeg")

	// Apply metadata tags using go-flac
	return applyMetadataTags(cue, opts)
}

// applyMetadataTags applies metadata to all split tracks
func applyMetadataTags(cue cueparser.CueFile, opts *SplitOptions) error {
	log.Printf("  Writing metadata tags with go-flac...")
	tagErrors := 0

	for _, track := range cue.Tracks {
		trackFile := filepath.Join(opts.OutputDir,
			fmt.Sprintf(opts.FilenamePattern, track.Number, sanitizeFilename(track.Title)))

		if err := writeFlacTags(trackFile, cue, track, track.Number); err != nil {
			log.Printf("  Warning: Failed to write tags for track %d: %v", track.Number, err)
			tagErrors++
		}
	}

	if tagErrors == 0 {
		log.Printf("  Metadata tags written successfully for all %d tracks", len(cue.Tracks))
	} else {
		log.Printf("  Metadata tags written (%d/%d tracks had errors)", tagErrors, len(cue.Tracks))
	}

	return nil
}

// writeFlacTags writes metadata tags to a FLAC file
func writeFlacTags(flacPath string, cue cueparser.CueFile, track cueparser.Track, trackNum int) error {
	// Open the FLAC file
	f, err := flac.ParseFile(flacPath)
	if err != nil {
		return fmt.Errorf("failed to parse FLAC file: %v", err)
	}

	// Get or create VorbisComment metadata block
	var cmtsmeta *flac.MetaDataBlock
	for _, meta := range f.Meta {
		if meta.Type == flac.VorbisComment {
			cmtsmeta = meta
			break
		}
	}

	var cmts *flacvorbis.MetaDataBlockVorbisComment
	if cmtsmeta != nil {
		cmts, err = flacvorbis.ParseFromMetaDataBlock(*cmtsmeta)
		if err != nil {
			return fmt.Errorf("failed to parse vorbis comment: %v", err)
		}
	} else {
		cmts = flacvorbis.New()
	}

	// Clear existing comments to avoid duplicates
	cmts.Comments = nil

	// Add standard tags
	cmts.Add(flacvorbis.FIELD_TITLE, track.Title)
	cmts.Add(flacvorbis.FIELD_ARTIST, track.Performer)
	cmts.Add(flacvorbis.FIELD_ALBUM, cue.Album)
	cmts.Add(flacvorbis.FIELD_PERFORMER, cue.Performer)
	cmts.Add(flacvorbis.FIELD_TRACKNUMBER, strconv.Itoa(trackNum))
	cmts.Add("TOTALTRACKS", strconv.Itoa(len(cue.Tracks)))

	// Add optional tags
	if cue.Date != "" {
		cmts.Add(flacvorbis.FIELD_DATE, cue.Date)
	}
	if cue.Genre != "" {
		cmts.Add(flacvorbis.FIELD_GENRE, cue.Genre)
	}
	if cue.Comment != "" {
		cmts.Add(flacvorbis.FIELD_DESCRIPTION, cue.Comment)
	}
	if cue.Catalog != "" {
		cmts.Add("CATALOG", cue.Catalog)
	}
	if cue.DiscID != "" {
		cmts.Add("DISCID", cue.DiscID)
	}

	// Marshal to metadata block
	res := cmts.Marshal()

	// Update or add the VorbisComment block
	if cmtsmeta != nil {
		*cmtsmeta = res
	} else {
		f.Meta = append(f.Meta, &res)
	}

	// Save the file
	if err := f.Save(flacPath); err != nil {
		return fmt.Errorf("failed to save FLAC file: %v", err)
	}

	return nil
}

// copyCueFile copies a CUE file and adjusts the FILE path to be absolute
func copyCueFile(srcPath, dstPath, flacPath string) error {
	input, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer input.Close()

	output, err := os.Create(dstPath)
	if err != nil {
		return err
	}
	defer output.Close()

	scanner := bufio.NewScanner(input)
	writer := bufio.NewWriter(output)
	defer writer.Flush()

	filePattern := regexp.MustCompile(`FILE\s+"([^"]+)"\s+WAVE`)

	for scanner.Scan() {
		line := scanner.Text()

		// Replace FILE path with absolute path
		if filePattern.MatchString(line) {
			absFlacPath, _ := filepath.Abs(flacPath)
			line = fmt.Sprintf(`FILE "%s" WAVE`, absFlacPath)
		}

		fmt.Fprintln(writer, line)
	}

	return scanner.Err()
}
