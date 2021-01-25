package config

import (
	"github.com/robbydyer/sports/pkg/nhlboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
)

type Config struct {
	EnableNHL          bool
	NHLConfig          *nhlboard.Config
	SportsMatrixConfig *sportsmatrix.Config
}
