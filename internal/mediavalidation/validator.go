package mediavalidation

import (
	"context"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"go.uber.org/zap"
	"qdebrid/internal/config"
)

// Validator validates media files using ffprobe
type Validator struct {
	config *config.MediaValidationConfig
	logger *zap.Logger
}

// NewValidator creates a new media validator
func NewValidator(cfg *config.MediaValidationConfig, logger *zap.Logger) *Validator {
	return &Validator{
		config: cfg,
		logger: logger,
	}
}

// ValidationResult contains the result of media validation
type ValidationResult struct {
	Valid           bool
	Reason          string
	HasVideo        bool
	HasAudio        bool
	DurationSeconds float64
	VideoCodec      string
	AudioCodec      string
	ContainerFormat string
	Width           int
	Height          int
	BitRate         int64
	FFProbeOutput   string
}

// FFProbeStream represents a stream from ffprobe output
type FFProbeStream struct {
	Index         int    `json:"index"`
	CodecName     string `json:"codec_name"`
	CodecLongName string `json:"codec_long_name"`
	CodecType     string `json:"codec_type"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	BitRate       string `json:"bit_rate"`
	Duration      string `json:"duration"`
	Channels      int    `json:"channels"`
}

// FFProbeFormat represents format information from ffprobe
type FFProbeFormat struct {
	Filename       string `json:"filename"`
	FormatName     string `json:"format_name"`
	FormatLongName string `json:"format_long_name"`
	Duration       string `json:"duration"`
	Size           string `json:"size"`
	BitRate        string `json:"bit_rate"`
}

// FFProbeOutput represents the full ffprobe JSON output
type FFProbeOutput struct {
	Streams []FFProbeStream `json:"streams"`
	Format  FFProbeFormat   `json:"format"`
}

// ValidateURL validates a media file from a URL
func (v *Validator) ValidateURL(ctx context.Context, url string) (*ValidationResult, error) {
	if !v.config.Enabled {
		return &ValidationResult{Valid: true, Reason: "validation disabled"}, nil
	}

	v.logger.Debug("validating media URL", zap.String("url", url))

	// Create timeout context for ffprobe
	timeoutCtx, cancel := context.WithTimeout(ctx, time.Duration(v.config.FFProbeTimeout)*time.Second)
	defer cancel()

	// Run ffprobe with JSON output
	// Using -v quiet to suppress ffprobe's own output
	// Using -print_format json for structured output
	// Using -show_format and -show_streams to get all relevant info
	cmd := exec.CommandContext(timeoutCtx, "ffprobe",
		"-v", "quiet",
		"-print_format", "json",
		"-show_format",
		"-show_streams",
		url,
	)

	output, err := cmd.CombinedOutput()
	if err != nil {
		v.logger.Warn("ffprobe failed",
			zap.String("url", url),
			zap.Error(err),
			zap.String("output", string(output)))
		return &ValidationResult{
			Valid:  false,
			Reason: fmt.Sprintf("ffprobe failed: %v", err),
		}, nil
	}

	// Parse ffprobe output
	var probeData FFProbeOutput
	if err := json.Unmarshal(output, &probeData); err != nil {
		v.logger.Error("failed to parse ffprobe output",
			zap.Error(err),
			zap.String("output", string(output)))
		return &ValidationResult{
			Valid:  false,
			Reason: fmt.Sprintf("failed to parse ffprobe output: %v", err),
		}, nil
	}

	// Analyze the results
	result := v.analyzeProbeData(&probeData, string(output))

	// Fail-fast: Apply validation rules and reject immediately on failure
	if v.config.RequireVideoStream && !result.HasVideo {
		result.Valid = false
		result.Reason = "no video stream found"
		v.logger.Warn("validation failed: no video stream", zap.String("url", url))
		return result, nil
	}

	if v.config.RequireAudioStream && !result.HasAudio {
		result.Valid = false
		result.Reason = "no audio stream found"
		v.logger.Warn("validation failed: no audio stream", zap.String("url", url))
		return result, nil
	}

	if v.config.MinDurationSeconds > 0 && result.DurationSeconds < float64(v.config.MinDurationSeconds) {
		result.Valid = false
		result.Reason = fmt.Sprintf("duration too short: %.1fs < %ds", result.DurationSeconds, v.config.MinDurationSeconds)
		v.logger.Warn("validation failed: duration too short",
			zap.String("url", url),
			zap.Float64("duration", result.DurationSeconds),
			zap.Int("min_duration", v.config.MinDurationSeconds))
		return result, nil
	}

	if v.config.RejectSampleFiles && result.DurationSeconds > 0 && result.DurationSeconds < float64(v.config.SampleMinRuntime) {
		result.Valid = false
		result.Reason = fmt.Sprintf("file appears to be a sample: %.1fs < %ds", result.DurationSeconds, v.config.SampleMinRuntime)
		v.logger.Warn("validation failed: sample file detected",
			zap.String("url", url),
			zap.Float64("duration", result.DurationSeconds),
			zap.Int("sample_min_runtime", v.config.SampleMinRuntime))
		return result, nil
	}

	// All checks passed
	result.Valid = true
	result.Reason = "validation passed"
	v.logger.Info("media validation passed",
		zap.String("url", url),
		zap.Bool("has_video", result.HasVideo),
		zap.Bool("has_audio", result.HasAudio),
		zap.Float64("duration", result.DurationSeconds),
		zap.String("video_codec", result.VideoCodec),
		zap.String("audio_codec", result.AudioCodec))

	return result, nil
}

// analyzeProbeData analyzes ffprobe output and extracts relevant information
func (v *Validator) analyzeProbeData(data *FFProbeOutput, rawOutput string) *ValidationResult {
	result := &ValidationResult{
		FFProbeOutput: rawOutput,
	}

	// Extract format information
	result.ContainerFormat = data.Format.FormatName

	// Parse duration from format (most reliable source)
	if data.Format.Duration != "" {
		fmt.Sscanf(data.Format.Duration, "%f", &result.DurationSeconds)
	}

	// Parse bit rate
	if data.Format.BitRate != "" {
		fmt.Sscanf(data.Format.BitRate, "%d", &result.BitRate)
	}

	// Analyze streams
	for _, stream := range data.Streams {
		switch stream.CodecType {
		case "video":
			// Filter out motion image codecs (like Radarr does)
			motionImageCodecs := []string{"mjpeg", "png", "gif"}
			isMotionImage := false
			for _, codec := range motionImageCodecs {
				if stream.CodecName == codec {
					isMotionImage = true
					break
				}
			}

			// Only consider non-motion-image video streams as the primary video
			if !isMotionImage && !result.HasVideo {
				result.HasVideo = true
				result.VideoCodec = stream.CodecName
				result.Width = stream.Width
				result.Height = stream.Height

				// Use stream duration if format duration is not available
				if result.DurationSeconds == 0 && stream.Duration != "" {
					fmt.Sscanf(stream.Duration, "%f", &result.DurationSeconds)
				}
			}

		case "audio":
			// Take the first audio stream as primary
			if !result.HasAudio {
				result.HasAudio = true
				result.AudioCodec = stream.CodecName

				// Use audio stream duration if we still don't have one
				if result.DurationSeconds == 0 && stream.Duration != "" {
					fmt.Sscanf(stream.Duration, "%f", &result.DurationSeconds)
				}
			}
		}
	}

	return result
}

// IsStreamableExtension checks if a file extension is streamable
func (v *Validator) IsStreamableExtension(filename string) bool {
	if len(v.config.StreamableExtensions) == 0 {
		return true // No filter, allow all
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if ext != "" && ext[0] == '.' {
		ext = ext[1:] // Remove leading dot
	}

	for _, allowedExt := range v.config.StreamableExtensions {
		if strings.ToLower(allowedExt) == ext {
			return true
		}
	}

	return false
}

// IsSampleFile checks if a file appears to be a sample based on its name
func (v *Validator) IsSampleFile(filename string) bool {
	lowerName := strings.ToLower(filename)
	sampleIndicators := []string{"sample", "trailer", "preview", "demo"}

	for _, indicator := range sampleIndicators {
		if strings.Contains(lowerName, indicator) {
			return true
		}
	}

	return false
}
