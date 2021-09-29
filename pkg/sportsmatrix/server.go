package sportsmatrix

import (
	"context"

	pb "github.com/robbydyer/sports/internal/proto/github.com/robbyder/sports/sportsmatrix"
)

func (s *SportsMatrix) Version(ctx context.Context) (*pb.VersionResp, error) {
	return &pb.VersionResp{
		Version: version,
	}, nil
}
