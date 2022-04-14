package config

import (
	"github.com/robbydyer/sports/internal/calendarboard"
	"github.com/robbydyer/sports/internal/clock"
	"github.com/robbydyer/sports/internal/imageboard"
	"github.com/robbydyer/sports/internal/racingboard"
	"github.com/robbydyer/sports/internal/sportboard"
	"github.com/robbydyer/sports/internal/sportsmatrix"
	"github.com/robbydyer/sports/internal/statboard"
	"github.com/robbydyer/sports/internal/stockboard"
	"github.com/robbydyer/sports/internal/sysboard"
	"github.com/robbydyer/sports/internal/weatherboard"
)

// Config holds configuration for the RGB matrix and all of its supported Boards
type Config struct {
	EnableNHL          bool                  `json:"enableNHL,omitempty"`
	NHLConfig          *sportboard.Config    `json:"nhlConfig,omitempty"`
	MLBConfig          *sportboard.Config    `json:"mlbConfig,omitempty"`
	NCAAMConfig        *sportboard.Config    `json:"ncaamConfig,omitempty"`
	NCAAFConfig        *sportboard.Config    `json:"ncaafConfig,omitempty"`
	NBAConfig          *sportboard.Config    `json:"nbaConfig,omitempty"`
	NFLConfig          *sportboard.Config    `json:"nflConfig,omitempty"`
	MLSConfig          *sportboard.Config    `json:"mlsConfig,omitempty"`
	EPLConfig          *sportboard.Config    `json:"eplConfig,omitempty"`
	ImageConfig        *imageboard.Config    `json:"imageConfig"`
	ClockConfig        *clock.Config         `json:"clockConfig"`
	SysConfig          *sysboard.Config      `json:"sysConfig"`
	PGA                *statboard.Config     `json:"pga"`
	SportsMatrixConfig *sportsmatrix.Config  `json:"sportsMatrixConfig,omitempty"`
	StocksConfig       *stockboard.Config    `json:"stocksConfig"`
	WeatherConfig      *weatherboard.Config  `json:"weatherConfig"`
	F1Config           *racingboard.Config   `json:"f1Config"`
	IRLConfig          *racingboard.Config   `json:"irlConfig"`
	CalenderConfig     *calendarboard.Config `json:"calendarConfig"`
}
