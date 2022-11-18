package imageboard

import (
	"context"
	"net/http"
	"time"

	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/twitchtv/twirp"

	pb "github.com/robbydyer/sports/internal/proto/imageboard"
)

// Server ...
type Server struct {
	board *ImageBoard
}

// GetRPCHandler ...
func (i *ImageBoard) GetRPCHandler() (string, http.Handler) {
	return i.rpcServer.PathPrefix(), i.rpcServer
}

// SetStatus ...
func (s *Server) SetStatus(ctx context.Context, req *pb.SetStatusReq) (*emptypb.Empty, error) {
	if req.Status == nil {
		return &emptypb.Empty{}, twirp.NewError(twirp.InvalidArgument, "nil status sent")
	}

	s.board.Enabler().Store(req.Status.Enabled)
	s.board.config.UseDiskCache.Store(req.Status.DiskcacheEnabled)
	s.board.config.UseMemCache.Store(req.Status.MemcacheEnabled)
	s.board.config.ScrollMode.Store(req.Status.ScrollEnabled)

	return &emptypb.Empty{}, nil
}

// GetStatus ...
func (s *Server) GetStatus(ctx context.Context, req *emptypb.Empty) (*pb.StatusResp, error) {
	return &pb.StatusResp{
		Status: &pb.Status{
			Enabled:          s.board.Enabler().Enabled(),
			DiskcacheEnabled: s.board.config.UseDiskCache.Load(),
			MemcacheEnabled:  s.board.config.UseMemCache.Load(),
			ScrollEnabled:    s.board.config.ScrollMode.Load(),
		},
	}, nil
}

// Jump ...
func (s *Server) Jump(ctx context.Context, req *pb.JumpReq) (*emptypb.Empty, error) {
	i := s.board
	i.jumpLock.Lock()
	defer i.jumpLock.Unlock()

	// Clear the channel
	select {
	case <-i.jumpTo:
	default:
	}

	s.board.priorJumpState.Store(s.board.Enabler().Enabled())

	select {
	case i.jumpTo <- req.Name:
	case <-time.After(5 * time.Second):
		return &emptypb.Empty{}, twirp.InternalError("timed out attempting image jump")
	}

	c, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	if i.jumpTo != nil {
		if err := i.jumper(c, i.Name()); err != nil {
			i.log.Error("failed to jump to image board",
				zap.Error(err),
				zap.String("file name", req.Name),
			)
			return &emptypb.Empty{}, twirp.InternalError("failed to jump to image board")
		}
	}

	return &emptypb.Empty{}, nil
}
