package main

import (
	"context"
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"time"

	yaml "github.com/ghodss/yaml"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

	"github.com/robbydyer/sports/internal/board"
	calendarboard "github.com/robbydyer/sports/internal/board/calendar"
	"github.com/robbydyer/sports/internal/board/clock"
	imageboard "github.com/robbydyer/sports/internal/board/image"
	racingboard "github.com/robbydyer/sports/internal/board/racing"
	sportboard "github.com/robbydyer/sports/internal/board/sport"
	statboard "github.com/robbydyer/sports/internal/board/stat"
	stockboard "github.com/robbydyer/sports/internal/board/stocks"
	sysboard "github.com/robbydyer/sports/internal/board/sys"
	textboard "github.com/robbydyer/sports/internal/board/text"
	weatherboard "github.com/robbydyer/sports/internal/board/weather"
	"github.com/robbydyer/sports/internal/config"
	"github.com/robbydyer/sports/internal/espnboard"
	"github.com/robbydyer/sports/internal/espnracing"
	"github.com/robbydyer/sports/internal/matrix"
	"github.com/robbydyer/sports/internal/mlb"
	"github.com/robbydyer/sports/internal/nhl"
	"github.com/robbydyer/sports/internal/openweather"
	"github.com/robbydyer/sports/internal/pga"
	rgb "github.com/robbydyer/sports/internal/rgbmatrix-rpi"
	"github.com/robbydyer/sports/internal/sportsmatrix"
	"github.com/robbydyer/sports/internal/util"
	"github.com/robbydyer/sports/internal/yahoo"
)

var defaultPGAUpdateInterval = 2 * time.Minute

const defaultConfigFile = "/etc/sportsmatrix.conf"

type rootArgs struct {
	level        string
	logLevel     zapcore.Level
	configFile   string
	config       *config.Config
	test         bool
	today        string
	logFile      string
	writer       *os.File
	alternateAPI bool
}

func main() {
	args := &rootArgs{}

	rootCmd := newRootCmd(args)

	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err.Error())

		os.Exit(1)
	}

	os.Exit(0)
}

func newRootCmd(args *rootArgs) *cobra.Command {
	rootCmd := &cobra.Command{
		Use:   "sports",
		Short: "Sports info",
		PersistentPreRunE: func(cmd *cobra.Command, a []string) error {
			configFile := viper.GetString("config")

			if configFile == defaultConfigFile {
				if _, err := os.Stat(configFile); err != nil && os.IsNotExist(err) {
					fmt.Println("Using default config")
					args.config = &config.Config{}
				} else {
					fmt.Printf("Loading config from file %s\n", configFile)
					if err := args.setConfig(configFile); err != nil {
						return fmt.Errorf("failed to load config file: %w", err)
					}
				}
			} else if configFile != "" {
				fmt.Printf("Loading config from file %s\n", configFile)
				if err := args.setConfig(configFile); err != nil {
					return fmt.Errorf("failed to load config file: %w", err)
				}
			} else {
				fmt.Println("Using default config")
				args.config = &config.Config{}
			}

			lvl := viper.GetString("log-level")

			if lvl == "" {
				args.logLevel = zapcore.InfoLevel
			} else {
				var l zapcore.Level
				if err := l.Set(lvl); err != nil {
					return err
				}
				args.logLevel = l
			}

			args.setConfigDefaults()

			return args.setTodayFuncs(viper.GetString("date-str"))
		},
	}

	f := rootCmd.PersistentFlags()

	f.StringVarP(&args.configFile, "config", "c", defaultConfigFile, "Config filename")
	f.StringVarP(&args.level, "log-level", "l", "info", "Log level. 'info', 'warn', 'debug'")
	f.BoolVarP(&args.test, "test", "t", false, "uses a test console matrix")
	f.StringVar(&args.today, "date-str", "", "Set the date of 'Today' for testing past days. Format 2020-01-30")
	f.StringVarP(&args.logFile, "log-file", "f", "", "Write logs to given file instead of STDOUT")
	f.BoolVarP(&args.alternateAPI, "alt-api", "a", false, "Use alternative API's where available")

	_ = viper.BindPFlags(f)

	rootCmd.AddCommand(newMlbCmd(args))
	rootCmd.AddCommand(newNhlCmd(args))
	rootCmd.AddCommand(newRunCmd(args))
	rootCmd.AddCommand(newNcaaMCmd(args))
	rootCmd.AddCommand(newAbbrevCmd(args))
	rootCmd.AddCommand(newStockCmd(args))
	rootCmd.AddCommand(newWeatherCmd(args))
	rootCmd.AddCommand(newCalCmd(args))

	return rootCmd
}

func (r *rootArgs) setConfig(filename string) error {
	f, err := ioutil.ReadFile(filename)
	if err != nil {
		return err
	}

	var c *config.Config

	if err := yaml.Unmarshal(f, &c); err != nil {
		return fmt.Errorf("failed to parse config file: %w", err)
	}

	r.config = c
	return nil
}

func (r *rootArgs) setConfigDefaults() {
	if r.config.SportsMatrixConfig == nil {
		r.config.SportsMatrixConfig = &sportsmatrix.Config{}
	}
	r.config.SportsMatrixConfig.Defaults()

	if r.config.NHLConfig == nil {
		r.config.NHLConfig = &sportboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	if r.config.NHLConfig.Stats == nil {
		r.config.NHLConfig.Stats = &statboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	if r.config.NHLConfig.Headlines == nil {
		r.config.NHLConfig.Headlines = &textboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}

	r.config.NHLConfig.SetDefaults()
	r.config.NHLConfig.Stats.SetDefaults()
	r.config.NHLConfig.Headlines.SetDefaults()

	if r.config.ImageConfig == nil {
		r.config.ImageConfig = &imageboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.ImageConfig.SetDefaults()

	if r.config.ClockConfig == nil {
		r.config.ClockConfig = &clock.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.ClockConfig.SetDefaults()

	if r.config.MLBConfig == nil {
		r.config.MLBConfig = &sportboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	if r.config.MLBConfig.Stats == nil {
		r.config.MLBConfig.Stats = &statboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	if r.config.MLBConfig.Headlines == nil {
		r.config.MLBConfig.Headlines = &textboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.MLBConfig.SetDefaults()
	r.config.MLBConfig.Stats.SetDefaults()
	r.config.MLBConfig.Headlines.SetDefaults()

	if r.config.NCAAMConfig == nil {
		r.config.NCAAMConfig = &sportboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	if r.config.NCAAMConfig.Headlines == nil {
		r.config.NCAAMConfig.Headlines = &textboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.NCAAMConfig.SetDefaults()
	r.config.NCAAMConfig.Headlines.SetDefaults()

	if r.config.NCAAFConfig == nil {
		r.config.NCAAFConfig = &sportboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	if r.config.NCAAFConfig.Headlines == nil {
		r.config.NCAAFConfig.Headlines = &textboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.NCAAFConfig.SetDefaults()
	r.config.NCAAFConfig.Headlines.SetDefaults()

	if r.config.NBAConfig == nil {
		r.config.NBAConfig = &sportboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	if r.config.NBAConfig.Headlines == nil {
		r.config.NBAConfig.Headlines = &textboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.NBAConfig.SetDefaults()
	r.config.NBAConfig.Headlines.SetDefaults()

	if r.config.NFLConfig == nil {
		r.config.NFLConfig = &sportboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	if r.config.NFLConfig.Headlines == nil {
		r.config.NFLConfig.Headlines = &textboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.NFLConfig.SetDefaults()
	r.config.NFLConfig.Headlines.SetDefaults()

	if r.config.MLSConfig == nil {
		r.config.MLSConfig = &sportboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	if r.config.MLSConfig.Headlines == nil {
		r.config.MLSConfig.Headlines = &textboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.MLSConfig.SetDefaults()
	r.config.MLSConfig.Headlines.SetDefaults()

	if r.config.EPLConfig == nil {
		r.config.EPLConfig = &sportboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	if r.config.EPLConfig.Headlines == nil {
		r.config.EPLConfig.Headlines = &textboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.EPLConfig.SetDefaults()
	r.config.EPLConfig.Headlines.SetDefaults()

	if r.config.SysConfig == nil {
		r.config.SysConfig = &sysboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.SysConfig.SetDefaults()

	if r.config.PGA == nil {
		r.config.PGA = &statboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	if r.config.PGA.UpdateInterval == "" {
		// Set PGA to a lower update interval than the default for Statboard
		r.config.PGA.UpdateInterval = defaultPGAUpdateInterval.String()
	}
	r.config.PGA.SetDefaults()
	r.config.PGA.Teams = append(r.config.PGA.Teams, "players")

	if r.config.StocksConfig == nil {
		r.config.StocksConfig = &stockboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.StocksConfig.SetDefaults()

	if r.config.WeatherConfig == nil {
		r.config.WeatherConfig = &weatherboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.WeatherConfig.SetDefaults()

	if r.config.F1Config == nil {
		r.config.F1Config = &racingboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.F1Config.SetDefaults()

	if r.config.IRLConfig == nil {
		r.config.IRLConfig = &racingboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.IRLConfig.SetDefaults()

	if r.config.CalenderConfig == nil {
		r.config.CalenderConfig = &calendarboard.Config{
			StartEnabled: atomic.NewBool(false),
		}
	}
	r.config.CalenderConfig.SetDefaults()
}

func (r *rootArgs) getRGBMatrix(logger *zap.Logger) (matrix.Matrix, error) {
	var matrix matrix.Matrix
	logger.Info("initializing matrix",
		zap.Int("Cols", r.config.SportsMatrixConfig.HardwareConfig.Cols),
		zap.Int("Rows", r.config.SportsMatrixConfig.HardwareConfig.Rows),
		zap.Int("Brightness", r.config.SportsMatrixConfig.HardwareConfig.Brightness),
		zap.String("Mapping", r.config.SportsMatrixConfig.HardwareConfig.HardwareMapping),
	)

	// If we have configured the http server to listen on a privileged port (like 80),
	// we need to maintain root permissions
	if r.config.SportsMatrixConfig.HTTPListenPort < 1024 {
		r.config.SportsMatrixConfig.RuntimeOptions.DropPrivileges = -1
	}

	var err error
	matrix, err = rgb.NewRGBLedMatrix(r.config.SportsMatrixConfig.HardwareConfig, r.config.SportsMatrixConfig.RuntimeOptions, logger)

	return matrix, err
}

func (r *rootArgs) getTestMatrix(logger *zap.Logger) matrix.Matrix {
	logger.Info("initializing console matrix",
		zap.Int("Cols", r.config.SportsMatrixConfig.HardwareConfig.Cols),
		zap.Int("Rows", r.config.SportsMatrixConfig.HardwareConfig.Rows),
	)
	return matrix.NewConsoleMatrix(r.config.SportsMatrixConfig.HardwareConfig.Cols, r.config.SportsMatrixConfig.HardwareConfig.Rows, os.Stdout, logger)
}

func (r *rootArgs) getBoards(ctx context.Context, logger *zap.Logger) ([]board.Board, error) {
	bounds := image.Rect(0, 0, r.config.SportsMatrixConfig.HardwareConfig.Cols, r.config.SportsMatrixConfig.HardwareConfig.Rows)

	var boards []board.Board

	nhlAPI, err := nhl.New(ctx, logger)
	if err != nil {
		logger.Error("nhl setup failed", zap.Error(err))
	}
	mlbAPI, err := mlb.New(ctx, logger)
	if err != nil {
		logger.Error("mlb setup failed", zap.Error(err))
	}

	if r.config.NHLConfig != nil && nhlAPI != nil {
		var api sportboard.API
		if r.alternateAPI {
			api, err = nhl.New(ctx, logger)
			if err != nil {
				return nil, err
			}
		} else {
			api, err = espnboard.NewNHL(ctx, logger)
			if err != nil {
				return boards, err
			}
		}
		b, err := sportboard.New(ctx, api, bounds, logger, r.config.NHLConfig)
		if err != nil {
			return boards, err
		}

		boards = append(boards, b)
	}

	if r.config.NHLConfig.Stats != nil {
		b, err := statboard.New(ctx, nhlAPI, r.config.NHLConfig.Stats, logger)
		if err != nil {
			return nil, err
		}

		boards = append(boards, b)
	}
	if r.config.NHLConfig.Headlines != nil {
		l, err := espnboard.GetLeaguer("nhl")
		if err != nil {
			return nil, err
		}
		api := espnboard.NewHeadlines(l, logger)
		b, err := textboard.New(api, r.config.NHLConfig.Headlines, logger, textboard.WithHalfSizeLogo())
		if err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	if r.config.MLBConfig != nil {
		var api sportboard.API
		if r.alternateAPI {
			api, err = mlb.New(ctx, logger)
			if err != nil {
				return nil, err
			}
		} else {
			api, err = espnboard.NewMLB(ctx, logger)
			if err != nil {
				return boards, err
			}
		}
		b, err := sportboard.New(ctx, api, bounds, logger, r.config.MLBConfig)
		if err != nil {
			return boards, err
		}

		boards = append(boards, b)
	}
	if r.config.MLBConfig.Stats != nil {
		b, err := statboard.New(ctx, mlbAPI, r.config.MLBConfig.Stats, logger)
		if err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	if r.config.MLBConfig.Headlines != nil {
		l, err := espnboard.GetLeaguer("mlb")
		if err != nil {
			return nil, err
		}
		api := espnboard.NewHeadlines(l, logger)
		b, err := textboard.New(api, r.config.MLBConfig.Headlines, logger, textboard.WithHalfSizeLogo())
		if err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	if r.config.NCAAMConfig != nil {
		api, err := espnboard.NewNCAAMensBasketball(ctx, logger)
		if err != nil {
			return boards, err
		}

		b, err := sportboard.New(ctx, api, bounds, logger, r.config.NCAAMConfig)
		if err != nil {
			return boards, err
		}

		boards = append(boards, b)
	}
	if r.config.NCAAMConfig.Headlines != nil {
		l, err := espnboard.GetLeaguer("ncaam")
		if err != nil {
			return nil, err
		}
		api := espnboard.NewHeadlines(l, logger)
		b, err := textboard.New(api, r.config.NCAAMConfig.Headlines, logger)
		if err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	if r.config.NCAAFConfig != nil {
		api, err := espnboard.NewNCAAF(ctx, logger)
		if err != nil {
			return boards, err
		}

		b, err := sportboard.New(ctx, api, bounds, logger, r.config.NCAAFConfig)
		if err != nil {
			return boards, err
		}

		boards = append(boards, b)
	}
	if r.config.NCAAFConfig.Headlines != nil {
		l, err := espnboard.GetLeaguer("ncaaf")
		if err != nil {
			return nil, err
		}
		api := espnboard.NewHeadlines(l, logger)
		b, err := textboard.New(api, r.config.NCAAFConfig.Headlines, logger)
		if err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	if r.config.NBAConfig != nil {
		api, err := espnboard.NewNBA(ctx, logger)
		if err != nil {
			return nil, err
		}

		b, err := sportboard.New(ctx, api, bounds, logger, r.config.NBAConfig)
		if err != nil {
			return nil, err
		}

		boards = append(boards, b)
	}
	if r.config.NBAConfig.Headlines != nil {
		l, err := espnboard.GetLeaguer("nba")
		if err != nil {
			return nil, err
		}
		api := espnboard.NewHeadlines(l, logger)
		b, err := textboard.New(api, r.config.NBAConfig.Headlines, logger)
		if err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	if r.config.NFLConfig != nil {
		api, err := espnboard.NewNFL(ctx, logger)
		if err != nil {
			return nil, err
		}

		b, err := sportboard.New(ctx, api, bounds, logger, r.config.NFLConfig)
		if err != nil {
			return nil, err
		}

		boards = append(boards, b)
	}
	if r.config.NFLConfig.Headlines != nil {
		l, err := espnboard.GetLeaguer("nfl")
		if err != nil {
			return nil, err
		}
		api := espnboard.NewHeadlines(l, logger)
		b, err := textboard.New(api, r.config.NFLConfig.Headlines, logger, textboard.WithHalfSizeLogo())
		if err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	if r.config.MLSConfig != nil {
		api, err := espnboard.NewMLS(ctx, logger)
		if err != nil {
			return nil, err
		}

		b, err := sportboard.New(ctx, api, bounds, logger, r.config.MLSConfig)
		if err != nil {
			return nil, err
		}

		boards = append(boards, b)
	}
	if r.config.MLSConfig.Headlines != nil {
		l, err := espnboard.GetLeaguer("mls")
		if err != nil {
			return nil, err
		}
		api := espnboard.NewHeadlines(l, logger)
		b, err := textboard.New(api, r.config.MLSConfig.Headlines, logger, textboard.WithHalfSizeLogo())
		if err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}
	if r.config.EPLConfig != nil {
		api, err := espnboard.NewEPL(ctx, logger)
		if err != nil {
			return nil, err
		}

		b, err := sportboard.New(ctx, api, bounds, logger, r.config.EPLConfig)
		if err != nil {
			return nil, err
		}

		boards = append(boards, b)
	}
	if r.config.EPLConfig.Headlines != nil {
		l, err := espnboard.GetLeaguer("epl")
		if err != nil {
			return nil, err
		}
		api := espnboard.NewHeadlines(l, logger)
		b, err := textboard.New(api, r.config.EPLConfig.Headlines, logger)
		if err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}

	if r.config.ImageConfig != nil {
		b, err := imageboard.New(r.config.ImageConfig, logger)
		if err != nil {
			return boards, err
		}
		boards = append(boards, b)
	}

	if r.config.ClockConfig != nil {
		b, err := clock.New(r.config.ClockConfig, logger)
		if err != nil {
			return boards, err
		}
		boards = append(boards, b)
	}

	if r.config.SysConfig != nil {
		b, err := sysboard.New(logger, r.config.SysConfig)
		if err != nil {
			return boards, err
		}
		boards = append(boards, b)
	}

	if r.config.PGA != nil {
		update := defaultPGAUpdateInterval
		if r.config.PGA.UpdateInterval != "" {
			d, err := time.ParseDuration(r.config.PGA.UpdateInterval)
			if err == nil {
				update = d
			}
		}
		api, err := pga.New(logger, update)
		if err != nil {
			return nil, err
		}
		b, err := statboard.New(ctx, api, r.config.PGA, logger,
			statboard.WithSorter(pga.SortByScore),
			statboard.WithTitleRow(false),
			statboard.WithPrefixCol(true),
		)
		if err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}

	if r.config.StocksConfig != nil {
		api, err := yahoo.New(logger)
		if err != nil {
			return nil, err
		}
		b, err := stockboard.New(api, r.config.StocksConfig, logger)
		if err != nil {
			return nil, err
		}

		boards = append(boards, b)
	}

	if r.config.WeatherConfig != nil {
		if r.config.WeatherConfig.APIKey == "" {
			logger.Warn("Missing Weather API key. Weather Board will not be enabled")
		} else {
			api, err := openweather.New(r.config.WeatherConfig.APIKey, 30*time.Minute, logger)
			if err != nil {
				return nil, err
			}
			b, err := weatherboard.New(api, r.config.WeatherConfig, logger)
			if err != nil {
				return nil, err
			}
			boards = append(boards, b)
		}
	}

	if r.config.F1Config != nil {
		api, err := espnracing.New(&espnracing.F1{}, logger)
		if err != nil {
			return nil, err
		}
		b, err := racingboard.New(api, logger, r.config.F1Config)
		if err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}

	if r.config.IRLConfig != nil {
		api, err := espnracing.New(&espnracing.IRL{}, logger)
		if err != nil {
			return nil, err
		}
		b, err := racingboard.New(api, logger, r.config.IRLConfig)
		if err != nil {
			return nil, err
		}
		boards = append(boards, b)
	}

	return boards, nil
}

func (r *rootArgs) setTodayFuncs(today string) error {
	if today == "" {
		return nil
	}

	t, err := time.Parse("2006-01-02T15:04:05", fmt.Sprintf("%sT12:00:00", today))
	if err != nil {
		return err
	}

	f := func() []time.Time {
		return []time.Time{t}
	}

	r.config.NHLConfig.TodayFunc = f
	r.config.MLBConfig.TodayFunc = f
	r.config.NCAAMConfig.TodayFunc = f
	r.config.NBAConfig.TodayFunc = f
	r.config.MLSConfig.TodayFunc = f
	r.config.EPLConfig.TodayFunc = f

	ncaafF := func() []time.Time {
		return util.NCAAFToday(t)
	}
	r.config.NCAAFConfig.TodayFunc = ncaafF

	nflF := func() []time.Time {
		return util.NFLToday(t)
	}
	r.config.NFLConfig.TodayFunc = nflF

	return nil
}
