package config

import (
	"github.com/robbydyer/sports/pkg/imageboard"
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
)

// Config holds configuration for the RGB matrix and all of its supported Boards
type Config struct {
	EnableNHL          bool                 `json:"enableNHL,omitempty"`
	NHLConfig          *sportboard.Config   `json:"nhlConfig,omitempty"`
	MLBConfig          *sportboard.Config   `json:"mlbConfig,omitempty"`
	ImageConfig        *imageboard.Config   `json:"imageConfig"`
	SportsMatrixConfig *sportsmatrix.Config `json:"sportsMatrixConfig,omitempty"`
}
