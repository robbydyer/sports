package sportsmatrix

import (
	"context"
	"math"
	"os"
	"syscall"
	"time"

	"github.com/twitchtv/twirp"
	"go.uber.org/zap"
	"google.golang.org/protobuf/types/known/emptypb"

	sportboard "github.com/robbydyer/sports/internal/board/sport"
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
	if err := s.sm.ScreenOff(ctx); err != nil {
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
			board.Enabler().Enable()
		}
		for _, board := range s.sm.betweenBoards {
			board.Enabler().Enable()
		}
	} else {
		for _, board := range s.sm.boards {
			board.Enabler().Disable()
		}
		for _, board := range s.sm.betweenBoards {
			board.Enabler().Disable()
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

// SetStatus ...
func (s *Server) SetStatus(ctx context.Context, req *pb.Status) (*emptypb.Empty, error) {
	if req.ScreenOn {
		if _, err := s.ScreenOn(ctx, &emptypb.Empty{}); err != nil {
			return nil, twirp.NewError(twirp.Internal, err.Error())
		}
	} else {
		if _, err := s.ScreenOff(ctx, &emptypb.Empty{}); err != nil {
			return nil, twirp.NewError(twirp.Internal, err.Error())
		}
	}

	if req.WebboardOn {
		if !s.sm.webBoardIsOn.Load() {
			s.sm.startWebBoard(ctx)
		}
	} else {
		if s.sm.webBoardIsOn.Load() {
			s.sm.stopWebBoard()
		}
	}

	if s.sm.cfg.CombinedScroll.CAS(!req.CombinedScroll, req.CombinedScroll) {
		if _, err := s.ScreenOff(ctx, &emptypb.Empty{}); err != nil {
			return nil, twirp.NewError(twirp.Internal, err.Error())
		}
		if _, err := s.ScreenOn(ctx, &emptypb.Empty{}); err != nil {
			return nil, twirp.NewError(twirp.Internal, err.Error())
		}
	}

	return &emptypb.Empty{}, nil
}

// GetStatus ...
func (s *Server) GetStatus(ctx context.Context, req *emptypb.Empty) (*pb.Status, error) {
	return &pb.Status{
		ScreenOn:       s.sm.screenIsOn.Load(),
		WebboardOn:     s.sm.webBoardIsOn.Load(),
		CombinedScroll: s.sm.cfg.CombinedScroll.Load(),
	}, nil
}

// NextBoard jumps to the next board in the sequence
func (s *Server) NextBoard(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	s.sm.Lock()
	defer s.sm.Unlock()
	s.sm.currentBoardCancel()
	return &emptypb.Empty{}, nil
}

// RestartService restarts the sportsmatrix service
func (s *Server) RestartService(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	myPid := os.Getpid()
	s.sm.log.Warn("restarting sportsmatrix service",
		zap.Int("pid", myPid),
	)

	proc, err := os.FindProcess(myPid)
	if err != nil {
		return nil, twirp.NewError(twirp.Internal, err.Error())
	}

	go func() {
		time.Sleep(2 * time.Second)
		if err := proc.Signal(syscall.SIGHUP); err != nil {
			s.sm.log.Error("failed to restart service",
				zap.Error(err),
			)
		}
	}()

	return &emptypb.Empty{}, nil
}

// SetLiveOnly sets the LiveOnly setting for SportBoards
func (s *Server) SetLiveOnly(ctx context.Context, req *pb.LiveOnlyReq) (*emptypb.Empty, error) {
	for _, board := range s.sm.boards {
		if sportBoard, ok := board.(*sportboard.SportBoard); ok {
			sportBoard.SetLiveOnly(req.LiveOnly)
		}
	}

	return &emptypb.Empty{}, nil
}

func (s *Server) SpeedUp(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	for _, scrollCanvas := range s.sm.getActiveScrollCanvases() {
		spd := time.Duration(int64(math.Ceil(float64(scrollCanvas.GetScrollSpeed()) - float64(speedUpIncrement))))
		if spd < 0 {
			continue
		}
		s.sm.log.Info("speeding up scroll canvas",
			zap.String("name", scrollCanvas.Name()),
			zap.Duration("new interval", spd),
		)
		scrollCanvas.SetScrollSpeed(spd)
	}
	return &emptypb.Empty{}, nil
}

func (s *Server) SlowDown(ctx context.Context, req *emptypb.Empty) (*emptypb.Empty, error) {
	for _, scrollCanvas := range s.sm.getActiveScrollCanvases() {
		spd := time.Duration(int64(math.Ceil(float64(scrollCanvas.GetScrollSpeed()) + float64(speedUpIncrement))))
		s.sm.log.Info("speeding up scroll canvas",
			zap.String("name", scrollCanvas.Name()),
			zap.Duration("new interval", spd),
		)
		scrollCanvas.SetScrollSpeed(spd)
	}
	return &emptypb.Empty{}, nil
}
