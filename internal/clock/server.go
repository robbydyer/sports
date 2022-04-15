package clock

import (
	"context"
	"net/http"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/twitchtv/twirp"

	pb "github.com/robbydyer/sports/internal/proto/basicboard"
)

// Server ...
type Server struct {
	board *Clock
}

// GetRPCHandler ...
func (c *Clock) GetRPCHandler() (string, http.Handler) {
	return c.rpcServer.PathPrefix(), c.rpcServer
}

// SetStatus ...
func (s *Server) SetStatus(ctx context.Context, req *pb.SetStatusReq) (*emptypb.Empty, error) {
	if req.Status == nil {
		return &emptypb.Empty{}, twirp.NewError(twirp.InvalidArgument, "nil status sent")
	}

	s.board.config.ScrollMode.Store(req.Status.ScrollEnabled)

	if req.Status.Enabled {
		s.board.Enabler().Enable()
	} else {
		s.board.Enabler().Disable()
	}

	return &emptypb.Empty{}, nil
}

// GetStatus ...
func (s *Server) GetStatus(ctx context.Context, req *emptypb.Empty) (*pb.StatusResp, error) {
	return &pb.StatusResp{
		Status: &pb.Status{
			Enabled:       s.board.Enabler().Enabled(),
			ScrollEnabled: s.board.config.ScrollMode.Load(),
		},
	}, nil
}
