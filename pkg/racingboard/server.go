package racingboard

import (
	"context"

	pb "github.com/robbydyer/sports/internal/proto/racingboard"
	"github.com/twitchtv/twirp"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"
)

type Server struct {
	board *RacingBoard
}

// SetStatus ...
func (s *Server) SetStatus(ctx context.Context, req *pb.SetStatusReq) (*emptypb.Empty, error) {
	if req.Status == nil {
		return &emptypb.Empty{}, twirp.NewError(twirp.InvalidArgument, "nil status sent")
	}

	if s.board.config.Enabled.CAS(!req.Status.Enabled, req.Status.Enabled) {
		s.board.log.Debug("racing board status change",
			zap.Bool("enabled", req.Status.Enabled),
		)
	}
	if s.board.config.ScrollMode.CAS(!req.Status.ScrollEnabled, req.Status.ScrollEnabled) {
		s.board.log.Debug("racing board status change",
			zap.Bool("scroll", req.Status.ScrollEnabled),
		)
	}

	return &emptypb.Empty{}, nil
}

// GetStatus ...
func (s *Server) GetStatus(ctx context.Context, req *emptypb.Empty) (*pb.StatusResp, error) {
	return &pb.StatusResp{
		Status: &pb.Status{
			Enabled:       s.board.config.Enabled.Load(),
			ScrollEnabled: s.board.config.ScrollMode.Load(),
		},
	}, nil
}
