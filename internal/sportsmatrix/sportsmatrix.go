package sportsmatrix

import (
	"context"
	"fmt"
	"image"
	_ "image/png"
	"time"

	rgb "github.com/robbydyer/rgbmatrix-rpi"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/pkg/nhl"
)

type SportsMatrix struct {
	nhlAPI *nhl.Nhl
	cfg    *Config
	matrix rgb.Matrix
	boards []board.Board
	done   chan bool
}

type Config struct {
	RotationDelay  time.Duration
	EnableNHL      bool
	HardwareConfig *rgb.HardwareConfig
}

func DefaultConfig() Config {
	dCfg := rgb.DefaultConfig
	dCfg.Rows = 64
	dCfg.Cols = 32
	dCfg.Brightness = 60
	return Config{
		RotationDelay:  5 * time.Second,
		HardwareConfig: &dCfg,
	}
}

func New(ctx context.Context, cfg Config, boards ...board.Board) (*SportsMatrix, error) {
	s := &SportsMatrix{
		boards: boards,
		cfg:    &cfg,
		done:   make(chan bool, 1),
	}

	var err error

	rt := &rgb.DefaultRuntimeOptions
	s.matrix, err = rgb.NewRGBLedMatrix(s.cfg.HardwareConfig, rt)
	if err != nil {
		return nil, err
	}

	return s, nil
}

// MatrixBounds returns an image.Rectangle of the matrix bounds
func (s *SportsMatrix) MatrixBounds() image.Rectangle {
	w, h := s.matrix.Geometry()
	return image.Rect(0, 0, w-1, h-1)
}

func (s *SportsMatrix) Done() chan bool {
	return s.done
}

func (s *SportsMatrix) Serve(ctx context.Context) error {
	if len(s.boards) < 1 {
		return fmt.Errorf("no boards configured")
	}
	fmt.Println("Serving boards...")
	for {
		select {
		case <-ctx.Done():
			fmt.Println("Got context cancel, cleaning up boards")
			go func() {
				for _, b := range s.boards {
					b.Cleanup()
				}
			}()
			time.Sleep(5 * time.Second)
			s.done <- true
			return nil
		default:
		}
	INNER:
		for _, b := range s.boards {
			if s.anyPriorities() && !b.HasPriority() {
				continue
			}
			if b.HasPriority() {
				fmt.Printf("Rendering board '%s' as priority\n", b.Name())
				err := b.Render(ctx, s.matrix)
				if err != nil {
					fmt.Printf("Error: %s", err.Error())
				}
				break INNER
			}
			fmt.Printf("Rendering board '%s'\n", b.Name())
			err := b.Render(ctx, s.matrix)
			if err != nil {
				fmt.Printf("Error: %s", err.Error())
			}
			b.Cleanup()
		}
	}
}

func (s *SportsMatrix) anyPriorities() bool {
	for _, b := range s.boards {
		if b.HasPriority() {
			return true
		}
	}

	return false
}

func (s *SportsMatrix) Close() {
	if len(s.boards) > 1 {
		fmt.Println("Waiting for boards to clean up")
		<-s.done
	}
	if s.matrix != nil {
		fmt.Println("Closing matrix")
		_ = s.matrix.Close()
	}
}
