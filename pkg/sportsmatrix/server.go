package sportsmatrix

import (
	"context"
	"time"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/robbydyer/sports/internal/proto/sportsmatrix"
)

// Server ...
type Server struct {
	sm *SportsMatrix
}

// Version ...
func (s *Server) Version(ctx context.Context, req *emptypb.Empty) (*pb.VersionResp, error) {
	return &pb.VersionResp{
		Version: version,
	}, nil
}

// ScreenOn ...
func (s *Server) ScreenOn(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.sm.ScreenOn(ctx); err != nil {
		return &emptypb.Empty{}, twirp.NewError(twirp.Internal, "failed to turn screen on")
	}
	return &emptypb.Empty{}, nil
}

// ScreenOff ...
func (s *Server) ScreenOff(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	if err := s.sm.ScreenOn(ctx); err != nil {
		return &emptypb.Empty{}, twirp.NewError(twirp.Internal, "failed to turn screen off")
	}
	return &emptypb.Empty{}, nil
}

// SetAll ...
func (s *Server) SetAll(ctx context.Context, req *pb.SetAllReq) (*emptypb.Empty, error) {
	s.sm.Lock()
	defer s.sm.Unlock()

	if req.Enabled {
		for _, board := range s.sm.boards {
			board.Enable()
		}
	} else {
		for _, board := range s.sm.boards {
			board.Disable()
		}
	}

	return &emptypb.Empty{}, nil
}

// Jump ...
func (s *Server) Jump(ctx context.Context, req *pb.JumpReq) (*emptypb.Empty, error) {
	if s.sm.jumping.Load() {
		return &emptypb.Empty{}, nil
	}

	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if err := s.sm.JumpTo(c, req.Board); err != nil {
		return nil, twirp.NewError(twirp.Internal, err.Error())
	}

	return &emptypb.Empty{}, nil
}

// Status ...
func (s *Server) Status(ctx context.Context, req *emptypb.Empty) (*pb.ScreenStatusResp, error) {
	return &pb.ScreenStatusResp{
		ScreenOn:   s.sm.screenIsOn.Load(),
		WebboardOn: s.sm.webBoardIsOn.Load(),
	}, nil
}

// NextBoard jumps to the next board in the sequence
func (s *Server) NextBoard(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	s.sm.Lock()
	defer s.sm.Unlock()
	s.sm.currentBoardCancel()
	return &emptypb.Empty{}, nil
}
