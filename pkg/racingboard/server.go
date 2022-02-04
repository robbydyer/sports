package racingboard

import (
	"context"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/robbydyer/sports/internal/proto/racingboard"
)

// Server ...
type Server struct {
	board *RacingBoard
}

// SetStatus ...
func (s *Server) SetStatus(ctx context.Context, req *pb.SetStatusReq) (*emptypb.Empty, error) {
	cancelBoard := false
	if req.Status == nil {
		return &emptypb.Empty{}, twirp.NewError(twirp.InvalidArgument, "nil status sent")
	}

	if s.board.config.Enabled.CAS(!req.Status.Enabled, req.Status.Enabled) {
		cancelBoard = true
	}
	if s.board.config.ScrollMode.CAS(!req.Status.ScrollEnabled, req.Status.ScrollEnabled) {
		cancelBoard = true
	}

	if cancelBoard {
		if s.board.boardCancel != nil {
			s.board.boardCancel()
		}
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
