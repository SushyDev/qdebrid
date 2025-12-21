package torrent

import (
	"context"
	"fmt"

	"github.com/sushydev/real_debrid_go/api"
	"go.uber.org/zap"
	"qdebrid/internal/config"
	"qdebrid/internal/debrid"
	"qdebrid/internal/mediavalidation"
)

// Validator handles media validation for torrents
type Validator struct {
	debridClient   *debrid.Client
	mediaValidator *mediavalidation.Validator
	config         *config.MediaValidationConfig
	logger         *zap.Logger
}

// NewValidator creates a new torrent validator
func NewValidator(
	debridClient *debrid.Client,
	mediaValidator *mediavalidation.Validator,
	config *config.MediaValidationConfig,
	logger *zap.Logger,
) *Validator {
	return &Validator{
		debridClient:   debridClient,
		mediaValidator: mediaValidator,
		config:         config,
		logger:         logger,
	}
}

// ValidateTorrent validates media files in a torrent
// Returns the torrent info to avoid multiple API calls
// If torrentInfo is provided (not nil), it will be used instead of fetching it again
func (v *Validator) ValidateTorrent(ctx context.Context, torrentID string, torrentInfo *api.TorrentInfo) (*api.TorrentInfo, error) {
	// Get torrent info if not provided
	if torrentInfo == nil {
		var err error
		torrentInfo, err = v.debridClient.GetTorrentInfo(ctx, torrentID)
		if err != nil {
			return nil, fmt.Errorf("failed to get torrent info: %w", err)
		}
	}

	// Skip if validation is disabled
	if !v.config.Enabled {
		return torrentInfo, nil
	}

	v.logger.Info("validating torrent media", zap.String("torrent_id", torrentID))

	// Fail-fast: Check if torrent status is 'downloaded' (if required)
	if v.config.RequireDownloaded && torrentInfo.Status != "downloaded" {
		v.logger.Warn("rejecting torrent: status not downloaded",
			zap.String("torrent_id", torrentID),
			zap.String("status", torrentInfo.Status))
		return nil, fmt.Errorf("torrent status is '%s', not 'downloaded' - rejecting", torrentInfo.Status)
	}

	// Filter files to only streamable extensions
	streamableFiles := v.filterStreamableFiles(torrentInfo.Files)

	if len(streamableFiles) == 0 {
		v.logger.Info("no streamable files found in torrent", zap.String("torrent_id", torrentID))
		return torrentInfo, nil // No streamable files, skip validation
	}

	v.logger.Info("found streamable files to validate",
		zap.String("torrent_id", torrentID),
		zap.Int("count", len(streamableFiles)))

	// Validate files (fail-fast on first failure)
	if err := v.validateFiles(ctx, torrentID, torrentInfo, streamableFiles); err != nil {
		return nil, err
	}

	return torrentInfo, nil
}

// CountValidVideoFiles counts the number of non-sample video files in a torrent
// Accepts torrentInfo to avoid additional API calls
func (v *Validator) CountValidVideoFiles(torrentInfo *api.TorrentInfo) int {
	v.logger.Debug("counting valid video files", zap.String("torrent_id", torrentInfo.ID))

	count := 0
	for _, file := range torrentInfo.Files {
		// Only count selected files
		if file.Selected != 1 {
			continue
		}

		// Only count streamable video files
		if !v.mediaValidator.IsStreamableExtension(file.Path) {
			continue
		}

		// Skip sample files
		if v.config.RejectSampleFiles && v.mediaValidator.IsSampleFile(file.Path) {
			v.logger.Debug("skipping sample file",
				zap.String("torrent_id", torrentInfo.ID),
				zap.String("file", file.Path))
			continue
		}

		count++
	}

	v.logger.Debug("counted valid video files",
		zap.String("torrent_id", torrentInfo.ID),
		zap.Int("count", count))

	return count
}

// filterStreamableFiles filters files to only those with streamable extensions
func (v *Validator) filterStreamableFiles(files []api.TorrentFile) []int {
	var streamableFiles []int
	for _, file := range files {
		// Only validate selected files
		if file.Selected == 1 && v.mediaValidator.IsStreamableExtension(file.Path) {
			streamableFiles = append(streamableFiles, file.ID)
		}
	}
	return streamableFiles
}

// validateFiles validates each streamable file, failing fast on first error
func (v *Validator) validateFiles(
	ctx context.Context,
	torrentID string,
	torrentInfo *api.TorrentInfo,
	streamableFileIDs []int,
) error {
	// The Links array corresponds to selected files
	// We need to match files with their links
	linkIndex := 0
	for _, file := range torrentInfo.Files {
		if file.Selected != 1 {
			continue
		}

		// Check if this file should be validated
		if !v.shouldValidateFile(file.ID, streamableFileIDs) {
			linkIndex++
			continue
		}

		// Validate this file
		if err := v.validateFile(ctx, torrentID, torrentInfo, file, linkIndex); err != nil {
			return err
		}

		linkIndex++
	}

	v.logger.Info("torrent validation completed successfully",
		zap.String("torrent_id", torrentID),
		zap.Int("validated_files", len(streamableFileIDs)))

	return nil
}

// shouldValidateFile checks if a file ID should be validated
func (v *Validator) shouldValidateFile(fileID int, streamableFileIDs []int) bool {
	for _, id := range streamableFileIDs {
		if fileID == id {
			return true
		}
	}
	return false
}

// validateFile validates a single file
func (v *Validator) validateFile(
	ctx context.Context,
	torrentID string,
	torrentInfo *api.TorrentInfo,
	file api.TorrentFile,
	linkIndex int,
) error {
	// Fail-fast: Make sure we have a corresponding link
	if linkIndex >= len(torrentInfo.Links) {
		v.logger.Warn("rejecting torrent: no link available for file",
			zap.String("torrent_id", torrentID),
			zap.String("file", file.Path),
			zap.Int("link_index", linkIndex),
			zap.Int("total_links", len(torrentInfo.Links)))
		return fmt.Errorf("no download link available for file: %s", file.Path)
	}

	restrictedLink := torrentInfo.Links[linkIndex]

	// Fail-fast: Unrestrict the link to get the actual download URL
	unrestrictedResp, err := v.debridClient.UnrestrictLink(ctx, restrictedLink)
	if err != nil {
		v.logger.Warn("rejecting torrent: failed to unrestrict link",
			zap.String("torrent_id", torrentID),
			zap.String("file", file.Path),
			zap.Error(err))
		return fmt.Errorf("failed to unrestrict link for file %s: %w", file.Path, err)
	}

	// Fail-fast: Validate the file with ffprobe
	result, err := v.mediaValidator.ValidateURL(ctx, unrestrictedResp.Download)
	if err != nil {
		v.logger.Error("rejecting torrent: validation error",
			zap.String("torrent_id", torrentID),
			zap.String("file", file.Path),
			zap.Error(err))
		return fmt.Errorf("validation error for file %s: %w", file.Path, err)
	}

	// Fail-fast: Reject immediately if validation fails
	if !result.Valid {
		v.logger.Warn("rejecting torrent: file validation failed",
			zap.String("torrent_id", torrentID),
			zap.String("file", file.Path),
			zap.String("reason", result.Reason),
			zap.Bool("has_video", result.HasVideo),
			zap.Bool("has_audio", result.HasAudio),
			zap.Float64("duration", result.DurationSeconds))
		return fmt.Errorf("file validation failed for %s: %s", file.Path, result.Reason)
	}

	v.logger.Info("file validation passed",
		zap.String("torrent_id", torrentID),
		zap.String("file", file.Path),
		zap.Float64("duration", result.DurationSeconds),
		zap.String("video_codec", result.VideoCodec),
		zap.String("audio_codec", result.AudioCodec))

	return nil
}
