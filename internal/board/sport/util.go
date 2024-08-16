package sportboard

import (
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"math"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/board"
	"github.com/robbydyer/sports/internal/rgbrender"
)

func (s *SportBoard) getTeamInfoWidth(league string, teamID string) (int, error) {
	s.teamInfoLock.RLock()
	defer s.teamInfoLock.RUnlock()

	_, lOk := s.teamInfoWidths[league]
	if !lOk {
		return 0, fmt.Errorf("no info for league %s", league)
	}
	i, ok := s.teamInfoWidths[league][teamID]
	if !ok {
		return 0, fmt.Errorf("no info for %s %s", league, teamID)
	}

	return i, nil
}

func (s *SportBoard) setTeamInfoWidth(league string, teamID string, width int) {
	s.teamInfoLock.Lock()
	defer s.teamInfoLock.Unlock()

	_, ok := s.teamInfoWidths[league]
	if !ok {
		s.teamInfoWidths[league] = make(map[string]int)
	}

	s.teamInfoWidths[league][teamID] = width
}

func (s *SportBoard) logCanvas(canvas board.Canvas, msg string) {
	s.log.Debug(msg,
		zap.Int("minX", canvas.Bounds().Min.X),
		zap.Int("minY", canvas.Bounds().Min.Y),
		zap.Int("maxX", canvas.Bounds().Max.X),
		zap.Int("maxY", canvas.Bounds().Max.Y),
	)
}

func (s *SportBoard) getTimeWriter(canvasBounds image.Rectangle) (*rgbrender.TextWriter, error) {
	bounds := rgbrender.ZeroedBounds(canvasBounds)

	s.log.Debug("time writer bounds",
		zap.Int("minX", bounds.Min.X),
		zap.Int("minY", bounds.Min.Y),
		zap.Int("maxX", bounds.Max.X),
		zap.Int("maxY", bounds.Max.Y),
		zap.Int("starting minX", canvasBounds.Min.X),
		zap.Int("starting minY", canvasBounds.Min.Y),
		zap.Int("starting maxX", canvasBounds.Max.X),
		zap.Int("starting maxY", canvasBounds.Max.Y),
	)

	k := fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy())
	w, ok := s.timeWriters[k]
	if ok {
		s.log.Debug("using cached time writer")
		return w, nil
	}

	timeWriter, err := rgbrender.DefaultTextWriter()
	if err != nil {
		return nil, err
	}

	if s.config.TimeFont != nil {
		s.log.Warn("using configured font size for time",
			zap.Float64("configured", s.config.TimeFont.Size),
		)
		timeWriter.FontSize = s.config.TimeFont.Size
	} else {
		if bounds.Dy() <= 256 {
			timeWriter.FontSize = 8.0
		} else {
			timeWriter.FontSize = 0.25 * float64(bounds.Dy())
		}
	}

	if bounds.Dy() <= 256 {
		timeWriter.YStartCorrection = -2
	} else {
		timeWriter.YStartCorrection = -1 * ((bounds.Dy() / 32) + 1)
	}

	s.log.Debug("time writer font",
		zap.Float64("size", timeWriter.FontSize),
		zap.Int("Y correction", timeWriter.YStartCorrection),
	)

	s.Lock()
	defer s.Unlock()
	s.timeWriters[k] = timeWriter

	return timeWriter, nil
}

func (s *SportBoard) getScoreWriter(canvasBounds image.Rectangle) (*rgbrender.TextWriter, error) {
	bounds := rgbrender.ZeroedBounds(canvasBounds)

	k := fmt.Sprintf("%dx%d", bounds.Dx(), bounds.Dy())
	w, ok := s.scoreWriters[k]
	if ok {
		s.log.Debug("using cached score writer")
		return w, nil
	}

	var scoreWriter *rgbrender.TextWriter

	if (bounds.Dx() == bounds.Dy()) && bounds.Dx() <= 32 {
		var err error
		scoreWriter, err = rgbrender.DefaultTextWriter()
		if err != nil {
			return nil, err
		}
		scoreWriter.FontSize = 0.25 * float64(bounds.Dy())
		scoreWriter.YStartCorrection = -1 * ((bounds.Dy() / 32) + 1)
	} else {
		fnt, err := rgbrender.GetFont("score.ttf")
		if err != nil {
			return nil, fmt.Errorf("failed to load font for score: %w", err)
		}
		size := 0.5 * float64(bounds.Dy())
		if s.config.ScoreFont != nil {
			s.log.Warn("Using configured font for Scores",
				zap.Float64("configured", s.config.ScoreFont.Size),
				zap.Float64("default", size),
			)
			size = s.config.ScoreFont.Size
		}
		scoreWriter = rgbrender.NewTextWriter(fnt, size)
		yCorrect := math.Ceil(float64(3.0/32.0) * float64(bounds.Dy()))
		scoreWriter.YStartCorrection = int(yCorrect * -1)
	}

	s.log.Debug("score writer font",
		zap.Float64("size", scoreWriter.FontSize),
		zap.Int("Y correction", scoreWriter.YStartCorrection),
	)

	s.Lock()
	defer s.Unlock()
	s.scoreWriters[k] = scoreWriter
	return scoreWriter, nil
}

func (s *SportBoard) isFavoriteGame(game Game) (bool, error) {
	homeTeam, err := game.HomeTeam()
	if err != nil {
		return false, err
	}
	awayTeam, err := game.AwayTeam()
	if err != nil {
		return false, err
	}

	return (s.isFavorite(awayTeam.GetAbbreviation()) || s.isFavorite(homeTeam.GetAbbreviation())), nil
}

func (s *SportBoard) isFavorite(abbrev string) bool {
	for _, a := range s.config.FavoriteTeams {
		if abbrev == a {
			return true
		}
	}

	return false
}

func (s *SportBoard) textAreaWidth(bounds image.Rectangle) int {
	bounds = rgbrender.ZeroedBounds(bounds)
	if bounds.Dx() == bounds.Dy() {
		return bounds.Dx() / 8
	}

	if s.config.UseGradient.Load() {
		if bounds.Dx() >= 64 && bounds.Dy() <= 64 {
			return 10
		}
		return int(math.Floor(float64(bounds.Dx()) / 5.0))
	}

	if bounds.Dx() >= 64 && bounds.Dy() <= 64 {
		return 16
	}
	return bounds.Dx() / 4
}

func (s *SportBoard) calculateTeamInfoWidth(canvas draw.Image, writer *rgbrender.TextWriter, strs []string) (int, error) {
	lengths, err := writer.MeasureStrings(canvas, strs)
	if err != nil {
		return defaultTeamInfoArea, err
	}

	maxLen := 0
	for _, l := range lengths {
		if l > maxLen {
			maxLen = l
		}
	}

	return maxLen, nil
}

func (s *SportBoard) writeBoxColor() color.Color {
	if s.config.UseGradient.Load() {
		return color.NRGBA{255, 255, 255, 0}
	}

	return color.Black
}

func scoreStr(g Game, homeSide side) (string, error) {
	a, err := g.AwayTeam()
	if err != nil {
		return "", err
	}
	h, err := g.HomeTeam()
	if err != nil {
		return "", err
	}

	if homeSide == left {
		return fmt.Sprintf("%d-%d", h.Score(), a.Score()), nil
	}
	return fmt.Sprintf("%d-%d", a.Score(), h.Score()), nil
}

func (s *SportBoard) season() string {
	todays := s.config.TodayFunc()
	if len(todays) < 1 {
		return ""
	}
	return fmt.Sprintf("%d", todays[0].Year())
}

func rankShift(bounds image.Rectangle) int {
	return int(math.Ceil(float64(bounds.Dy()) * 3.0 / 32.0))
}
