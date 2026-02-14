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
	"io"
	"log"
	"os"
	"path/filepath"

	"github.com/ldmonster/flac-splitter/internal/cueparser"
	"github.com/mewkiz/flac"
	"github.com/mewkiz/flac/frame"
	"github.com/mewkiz/flac/meta"
)

// SplitWithGoAudio splits FLAC files using pure Go libraries only
// This implementation decodes, extracts samples, and re-encodes without external tools
func SplitWithGoAudio(cue cueparser.CueFile, flacPath string, opts *SplitOptions) error {
	log.Printf("  Using pure Go audio libraries for splitting (no external tools)...")

	// Open the source FLAC file for decoding
	stream, err := flac.Open(flacPath)
	if err != nil {
		return fmt.Errorf("failed to open FLAC file: %v", err)
	}
	defer stream.Close()

	// Get stream info
	info := stream.Info
	log.Printf("  FLAC Info - Sample Rate: %d Hz, Channels: %d, Bits/Sample: %d",
		info.SampleRate, info.NChannels, info.BitsPerSample)

	// Read all audio samples into memory first
	log.Printf("  Reading and decoding FLAC audio data...")
	samples, err := readAllSamples(stream)
	if err != nil {
		return fmt.Errorf("failed to read FLAC samples: %v", err)
	}

	totalSamples := uint64(len(samples[0]))
	log.Printf("  Decoded %d samples per channel", totalSamples)

	// Process each track
	for i, track := range cue.Tracks {
		startSample := cueTimeToSample(track.Index, info.SampleRate)

		var endSample uint64
		if i < len(cue.Tracks)-1 {
			endSample = cueTimeToSample(cue.Tracks[i+1].Index, info.SampleRate)
		} else {
			endSample = totalSamples
		}

		// Validate sample range
		if startSample >= totalSamples {
			log.Printf("  Warning: Track %d start sample %d exceeds total samples %d, skipping",
				track.Number, startSample, totalSamples)
			continue
		}
		if endSample > totalSamples {
			endSample = totalSamples
		}

		outputFile := filepath.Join(opts.OutputDir,
			fmt.Sprintf(opts.FilenamePattern, track.Number, sanitizeFilename(track.Title)))

		log.Printf("  Encoding track %d: %s (samples %d-%d)",
			track.Number, track.Title, startSample, endSample)

		// Extract samples for this track
		trackSamples := extractSampleRange(samples, startSample, endSample)

		// Encode to FLAC
		if err := encodeFlac(outputFile, trackSamples, info); err != nil {
			log.Printf("  Warning: Failed to encode track %d: %v", track.Number, err)
			continue
		}

		// Write metadata tags
		if err := writeFlacTags(outputFile, cue, track, track.Number); err != nil {
			log.Printf("  Warning: Failed to write tags for track %d: %v", track.Number, err)
		}
	}

	log.Printf("  Split complete with pure Go audio libraries")
	return nil
}

// readAllSamples decodes all FLAC frames into sample arrays
func readAllSamples(stream *flac.Stream) ([][]int32, error) {
	info := stream.Info
	numChannels := int(info.NChannels)

	// Initialize sample arrays for each channel
	samples := make([][]int32, numChannels)
	for i := range samples {
		samples[i] = make([]int32, 0, info.NSamples)
	}

	// Parse all frames
	for {
		frame, err := stream.ParseNext()
		if err != nil {
			if err == io.EOF {
				break
			}
			return nil, fmt.Errorf("failed to parse frame: %w", err)
		}

		// Append samples from each subframe (channel)
		for ch := 0; ch < numChannels; ch++ {
			samples[ch] = append(samples[ch], frame.Subframes[ch].Samples...)
		}
	}

	return samples, nil
}

// extractSampleRange extracts a range of samples from multi-channel sample arrays
func extractSampleRange(samples [][]int32, start, end uint64) [][]int32 {
	numChannels := len(samples)
	extracted := make([][]int32, numChannels)

	for ch := 0; ch < numChannels; ch++ {
		if end > uint64(len(samples[ch])) {
			end = uint64(len(samples[ch]))
		}
		extracted[ch] = samples[ch][start:end]
	}

	return extracted
}

// encodeFlac encodes samples to a FLAC file
func encodeFlac(outputPath string, samples [][]int32, info *meta.StreamInfo) error {
	if len(samples) == 0 || len(samples[0]) == 0 {
		return fmt.Errorf("no samples to encode")
	}

	// Create output file
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("failed to create output file: %w", err)
	}
	defer outFile.Close()

	// Create a new stream info for the output file
	outputInfo := &meta.StreamInfo{
		SampleRate:    info.SampleRate,
		BitsPerSample: info.BitsPerSample,
		NChannels:     info.NChannels,
		NSamples:      uint64(len(samples[0])),
	}

	// Create encoder
	enc, err := flac.NewEncoder(outFile, outputInfo)
	if err != nil {
		return fmt.Errorf("failed to create encoder: %w", err)
	}
	defer enc.Close()

	// Determine channel mode for header
	channelMode := frame.ChannelsMono
	if info.NChannels == 2 {
		channelMode = frame.ChannelsLR
	}

	// Write samples in frames
	numChannels := int(info.NChannels)
	numSamples := len(samples[0])
	blockSize := 4096 // Standard FLAC block size

	for offset := 0; offset < numSamples; offset += blockSize {
		// Determine how many samples to write in this frame
		frameSamples := blockSize
		if offset+frameSamples > numSamples {
			frameSamples = numSamples - offset
		}

		// Create frame with header
		f := &frame.Frame{
			Header: frame.Header{
				HasFixedBlockSize: true,
				BlockSize:         uint16(frameSamples),
				SampleRate:        info.SampleRate,
				Channels:          channelMode,
				BitsPerSample:     info.BitsPerSample,
			},
		}

		// Create subframes for each channel
		f.Subframes = make([]*frame.Subframe, numChannels)
		for ch := 0; ch < numChannels; ch++ {
			// Extract samples for this channel and frame
			channelSamples := samples[ch][offset : offset+frameSamples]

			// Create verbatim subframe (uncompressed)
			f.Subframes[ch] = &frame.Subframe{
				SubHeader: frame.SubHeader{
					Pred: frame.PredVerbatim,
				},
				Samples:  channelSamples,
				NSamples: frameSamples,
			}
		}

		// Write the frame
		if err := enc.WriteFrame(f); err != nil {
			return fmt.Errorf("failed to write frame: %w", err)
		}
	}

	return nil
}

// cueTimeToSample converts CUE time format (MM:SS:FF) to sample number
func cueTimeToSample(cueTime string, sampleRate uint32) uint64 {
	seconds := parseFloat(convertCueTimeToSeconds(cueTime))
	return uint64(seconds * float64(sampleRate))
}

// parseFloat safely parses a float64 from a string
func parseFloat(s string) float64 {
	var f float64
	fmt.Sscanf(s, "%f", &f)
	return f
}

// SplitWithGoAudioSimple is a hybrid approach that uses go-audio for validation
// but still uses external tools for actual splitting
func SplitWithGoAudioSimple(cue cueparser.CueFile, flacPath string, opts *SplitOptions) error {
	log.Printf("  Validating FLAC file with go-audio libraries...")

	// Open and validate the FLAC file
	stream, err := flac.Open(flacPath)
	if err != nil {
		return fmt.Errorf("failed to open/validate FLAC file: %v", err)
	}
	defer stream.Close()

	info := stream.Info
	log.Printf("  FLAC validated - Sample Rate: %d Hz, Channels: %d, Duration: %.2f seconds",
		info.SampleRate, info.NChannels, float64(info.NSamples)/float64(info.SampleRate))

	// Validate that all tracks fit within the audio duration
	totalDuration := float64(info.NSamples) / float64(info.SampleRate)
	for i, track := range cue.Tracks {
		trackStart := parseFloat(convertCueTimeToSeconds(track.Index))

		if trackStart > totalDuration {
			return fmt.Errorf("track %d starts at %.2fs but audio is only %.2fs long",
				track.Number, trackStart, totalDuration)
		}

		if i < len(cue.Tracks)-1 {
			trackEnd := parseFloat(convertCueTimeToSeconds(cue.Tracks[i+1].Index))
			log.Printf("  Track %d: %.2fs - %.2fs (%.2fs)",
				track.Number, trackStart, trackEnd, trackEnd-trackStart)
		} else {
			log.Printf("  Track %d: %.2fs - %.2fs (%.2fs)",
				track.Number, trackStart, totalDuration, totalDuration-trackStart)
		}
	}

	// Use external tools for actual splitting (validated approach)
	log.Printf("  Using external tools for actual splitting (after validation)...")

	if executableExists("ffmpeg") {
		return splitWithFFmpeg(cue, flacPath, opts)
	} else if executableExists("shnsplit") {
		return splitWithShnsplit(cue, flacPath, opts)
	}

	return fmt.Errorf("no external tools available for splitting")
}
