package config

import (
	"github.com/robbydyer/sports/pkg/clock"
	"github.com/robbydyer/sports/pkg/imageboard"
	"github.com/robbydyer/sports/pkg/sportboard"
	"github.com/robbydyer/sports/pkg/sportsmatrix"
	"github.com/robbydyer/sports/pkg/statboard"
	"github.com/robbydyer/sports/pkg/sysboard"
)

// Config holds configuration for the RGB matrix and all of its supported Boards
type Config struct {
	EnableNHL          bool                 `json:"enableNHL,omitempty"`
	NHLConfig          *sportboard.Config   `json:"nhlConfig,omitempty"`
	MLBConfig          *sportboard.Config   `json:"mlbConfig,omitempty"`
	NCAAMConfig        *sportboard.Config   `json:"ncaamConfig,omitempty"`
	NCAAFConfig        *sportboard.Config   `json:"ncaafConfig,omitempty"`
	NBAConfig          *sportboard.Config   `json:"nbaConfig,omitempty"`
	NFLConfig          *sportboard.Config   `json:"nflConfig,omitempty"`
	MLSConfig          *sportboard.Config   `json:"mlsConfig,omitempty"`
	ImageConfig        *imageboard.Config   `json:"imageConfig"`
	ClockConfig        *clock.Config        `json:"clockConfig"`
	SysConfig          *sysboard.Config     `json:"sysConfig"`
	PGA                *statboard.Config    `json:"pga"`
	SportsMatrixConfig *sportsmatrix.Config `json:"sportsMatrixConfig,omitempty"`
}
