package textboard

import (
	"context"
	"net/http"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/twitchtv/twirp"

	pb "github.com/robbydyer/sports/internal/proto/basicboard"
)

// Server ...
type Server struct {
	board *TextBoard
}

// GetRPCHandler ...
func (s *TextBoard) GetRPCHandler() (string, http.Handler) {
	return s.rpcServer.PathPrefix(), s.rpcServer
}

// SetStatus ...
func (s *Server) SetStatus(ctx context.Context, req *pb.SetStatusReq) (*emptypb.Empty, error) {
	if req.Status == nil {
		return &emptypb.Empty{}, twirp.NewError(twirp.InvalidArgument, "nil status sent")
	}

	cancelBoard := false
	if req.Status.Enabled {
		if s.board.Enabler().Enable() {
			cancelBoard = true
		}
	} else {
		if s.board.Enabler().Disable() {
			cancelBoard = true
		}
	}

	if cancelBoard {
		select {
		case s.board.cancelBoard <- struct{}{}:
			s.board.log.Info("sent cancel board signal on status change")
		default:
		}
	}

	return &emptypb.Empty{}, nil
}

// GetStatus ...
func (s *Server) GetStatus(ctx context.Context, req *emptypb.Empty) (*pb.StatusResp, error) {
	return &pb.StatusResp{
		Status: &pb.Status{
			Enabled:       s.board.Enabler().Enabled(),
			ScrollEnabled: true,
		},
	}, nil
}
