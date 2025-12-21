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

// Service handles torrent operations and validation
type Service struct {
	debridClient *debrid.Client
	validator    *Validator
	logger       *zap.Logger
}

// NewService creates a new torrent service
func NewService(
	debridClient *debrid.Client,
	cfg *config.MediaValidationConfig,
	logger *zap.Logger,
) *Service {
	mediaValidator := mediavalidation.NewValidator(cfg, logger.Named("media"))
	validator := NewValidator(debridClient, mediaValidator, cfg, logger.Named("validator"))

	return &Service{
		debridClient: debridClient,
		validator:    validator,
		logger:       logger,
	}
}

// AddAndValidate adds a torrent and validates it if configured
// Returns the torrent info to avoid multiple API calls
// If torrentInfo is provided, it will be used instead of fetching it again
func (s *Service) AddAndValidate(ctx context.Context, torrentID string, torrentInfo *api.TorrentInfo) (*api.TorrentInfo, error) {
	return s.validator.ValidateTorrent(ctx, torrentID, torrentInfo)
}

// ValidateFileCount validates that a torrent has the expected number of video files
// Accepts torrentInfo to avoid additional API calls
func (s *Service) ValidateFileCount(ctx context.Context, torrentInfo *api.TorrentInfo, expectedCount int) error {
	// Count actual video files in torrent
	actualCount := s.validator.CountValidVideoFiles(torrentInfo)

	// Check if we have at least the expected number of files
	if actualCount < expectedCount {
		s.logger.Warn("torrent file count validation failed",
			zap.String("torrent_id", torrentInfo.ID),
			zap.Int("expected", expectedCount),
			zap.Int("actual", actualCount))
		return fmt.Errorf("insufficient video files: expected at least %d, found %d", expectedCount, actualCount)
	}

	s.logger.Info("torrent file count validation passed",
		zap.String("torrent_id", torrentInfo.ID),
		zap.Int("expected", expectedCount),
		zap.Int("actual", actualCount))

	return nil
}

// GetInfo retrieves torrent info
func (s *Service) GetInfo(ctx context.Context, torrentID string) (*api.TorrentInfo, error) {
	return s.debridClient.GetTorrentInfo(ctx, torrentID)
}

// GetInfoByHash retrieves torrent info by hash
func (s *Service) GetInfoByHash(ctx context.Context, hash string) (*api.TorrentInfo, error) {
	return s.debridClient.GetTorrentInfoByHash(ctx, hash)
}

// Delete deletes a torrent
func (s *Service) Delete(ctx context.Context, torrentID string) error {
	return s.debridClient.DeleteTorrent(ctx, torrentID)
}

// DeleteByHash deletes a torrent by hash
func (s *Service) DeleteByHash(ctx context.Context, hash string) error {
	return s.debridClient.DeleteTorrentByHash(ctx, hash)
}
