package weatherboard

import (
	"context"
	"net/http"

	"google.golang.org/protobuf/types/known/emptypb"

	"github.com/twitchtv/twirp"

	pb "github.com/robbydyer/sports/internal/proto/weatherboard"
)

// Server ...
type Server struct {
	board *WeatherBoard
}

// GetRPCHandler ...
func (w *WeatherBoard) GetRPCHandler() (string, http.Handler) {
	return w.rpcServer.PathPrefix(), w.rpcServer
}

// SetStatus ...
func (s *Server) SetStatus(ctx context.Context, req *pb.SetStatusReq) (*emptypb.Empty, error) {
	if req.Status == nil {
		return &emptypb.Empty{}, twirp.NewError(twirp.InvalidArgument, "nil status sent")
	}

	cancelBoard := false
	if s.board.Enabler().Store(req.Status.Enabled) {
		cancelBoard = true
	}
	if s.board.config.ScrollMode.CompareAndSwap(!req.Status.ScrollEnabled, req.Status.ScrollEnabled) {
		cancelBoard = true
	}
	if s.board.config.DailyForecast.CompareAndSwap(!req.Status.DailyEnabled, req.Status.DailyEnabled) {
		cancelBoard = true
	}
	if s.board.config.HourlyForecast.CompareAndSwap(!req.Status.HourlyEnabled, req.Status.HourlyEnabled) {
		cancelBoard = true
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
			ScrollEnabled: s.board.config.ScrollMode.Load(),
			DailyEnabled:  s.board.config.DailyForecast.Load(),
			HourlyEnabled: s.board.config.HourlyForecast.Load(),
		},
	}, nil
}
