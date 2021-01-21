package sportsmatrix

import (
	"context"
	"fmt"
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
	return Config{
		RotationDelay: 5 * time.Second,
		HardwareConfig: &rgb.HardwareConfig{
			Rows:        64,
			Cols:        32,
			Brightness:  60,
			ChainLength: 1,
			Parallel:    1,
			PWMBits:     11,
		},
	}
}

func New(ctx context.Context, cfg Config, boards ...board.Board) (*SportsMatrix, error) {
	s := &SportsMatrix{
		boards: boards,
		cfg:    &cfg,
		done:   make(chan bool),
	}

	var err error

	rt := &rgb.DefaultRuntimeOptions
	s.matrix, err = rgb.NewRGBLedMatrix(s.cfg.HardwareConfig, rt)
	if err != nil {
		return nil, err
	}

	return s, nil
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
			for _, b := range s.boards {
				b.Cleanup()
			}
			s.done <- true
			return nil
		default:
		}
	INNER:
		for _, b := range s.boards {
			if b.HasPriority() {
				fmt.Printf("Rendering board '%s' as priority\n", b.Name())
				err := b.Render(ctx, s.matrix, s.cfg.RotationDelay)
				if err != nil {
					fmt.Printf("Error: %s", err.Error())
				}
				break INNER
			}
			fmt.Printf("Rendering board '%s'\n", b.Name())
			err := b.Render(ctx, s.matrix, s.cfg.RotationDelay)
			if err != nil {
				fmt.Printf("Error: %s", err.Error())
			}
			b.Cleanup()
		}
	}
}

func (s *SportsMatrix) Close() {
	fmt.Println("Waiting for boards to clean up")
	<-s.done
	if s.matrix != nil {
		fmt.Println("Closing matrix")
		_ = s.matrix.Close()
	}
}
