package torrent

import (
	"context"

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
func (s *Service) AddAndValidate(ctx context.Context, torrentID string) error {
	return s.validator.ValidateTorrent(ctx, torrentID)
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
