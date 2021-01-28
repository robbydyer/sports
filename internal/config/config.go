package config

import (
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
)

type Config struct {
	EnableNHL          bool                 `json:"enableNHL,omitempty"`
	NHLConfig          *sportboard.Config   `json:"nhlConfig,omitempty"`
	SportsMatrixConfig *sportsmatrix.Config `json:"sportsMatrixConfig,omitempty"`
}
