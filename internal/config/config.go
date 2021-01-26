package config

import (
	"github.com/robbydyer/sports/pkg/nhlboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
)

type Config struct {
	EnableNHL          bool                 `json:"enableNHL,omitempty"`
	NHLConfig          *nhlboard.Config     `json:"NHLConfig,omitempty"`
	SportsMatrixConfig *sportsmatrix.Config `json:"sportsMatrixConfig,omitempty"`
}
