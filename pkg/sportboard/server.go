package sportboard

import (
	"context"
	"net/http"

	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/types/known/emptypb"

	pb "github.com/robbydyer/sports/internal/proto/sportboard"
)

// Server ...
type Server struct {
	board *SportBoard
}

// GetRPCHandler ...
func (s *SportBoard) GetRPCHandler() (string, http.Handler) {
	return s.rpcServer.PathPrefix(), s.rpcServer
}

// SetStatus ...
func (s *Server) SetStatus(ctx context.Context, req *pb.SetStatusReq) (*emptypb.Empty, error) {
	cancelBoard := false

	if req.Status == nil {
		return &emptypb.Empty{}, twirp.NewError(twirp.InvalidArgument, "nil status sent")
	}

	if s.board.config.HideFavoriteScore.CAS(!req.Status.FavoriteHidden, req.Status.FavoriteHidden) {
		cancelBoard = true
	}
	if s.board.config.Enabled.CAS(!req.Status.Enabled, req.Status.Enabled) {
		cancelBoard = true
	}
	if s.board.config.FavoriteSticky.CAS(!req.Status.FavoriteSticky, req.Status.FavoriteSticky) {
		cancelBoard = true
	}
	if s.board.config.GamblingSpread.CAS(!req.Status.OddsEnabled, req.Status.OddsEnabled) {
		cancelBoard = true
	}
	if s.board.config.ScrollMode.CAS(!req.Status.ScrollEnabled, req.Status.ScrollEnabled) {
		cancelBoard = true
	}
	if s.board.config.TightScroll.CAS(!req.Status.TightScrollEnabled, req.Status.TightScrollEnabled) {
		cancelBoard = true
	}
	if s.board.config.ShowRecord.CAS(!req.Status.RecordRankEnabled, req.Status.RecordRankEnabled) {
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
			Enabled:            s.board.config.Enabled.Load(),
			FavoriteHidden:     s.board.config.HideFavoriteScore.Load(),
			FavoriteSticky:     s.board.config.FavoriteSticky.Load(),
			ScrollEnabled:      s.board.config.ScrollMode.Load(),
			TightScrollEnabled: s.board.config.TightScroll.Load(),
			RecordRankEnabled:  s.board.config.ShowRecord.Load(),
			OddsEnabled:        s.board.config.GamblingSpread.Load(),
		},
	}, nil
}
