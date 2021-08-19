package sportboard

import (
	"go.uber.org/atomic"
	"go.uber.org/zap"
)

type previousScore struct {
	id   int
	home *previousTeam
	away *previousTeam
}
type previousTeam struct {
	previous   *atomic.Int32
	repeats    *atomic.Int32
	init       *atomic.Bool
	maxRepeats int32
}

func (s *SportBoard) storeOrGetPreviousScore(id int, away int, home int) *previousScore {
	s.prevScoreLock.Lock()
	defer s.prevScoreLock.Unlock()
	for _, s := range s.previousScores {
		if s.id == id {
			// Already have this score
			return s
		}
	}
	p := &previousScore{
		id: id,
		home: &previousTeam{
			init:       atomic.NewBool(false),
			previous:   atomic.NewInt32(int32(home)),
			repeats:    atomic.NewInt32(0),
			maxRepeats: int32(*s.config.ScoreHighlightRepeat),
		},
		away: &previousTeam{
			init:       atomic.NewBool(false),
			previous:   atomic.NewInt32(int32(away)),
			repeats:    atomic.NewInt32(0),
			maxRepeats: int32(*s.config.ScoreHighlightRepeat),
		},
	}
	s.previousScores = append(s.previousScores, p)

	s.log.Debug("storing game previous score",
		zap.Int("id", id),
		zap.Int32("home repeat", p.home.maxRepeats),
		zap.Int32("away repeat", p.home.maxRepeats),
	)

	return p
}

func (t *previousTeam) hasScored(current int) bool {
	if !t.init.Load() {
		t.previous.Store(int32(current))
		t.init.Store(true)
		return false
	}

	c := int32(current)
	if c != t.previous.Load() {
		t.previous.Store(c)
		t.repeats.Store(0)
		return true
	}

	if t.repeats.Load() < t.maxRepeats {
		t.repeats.Inc()
		return true
	}

	return false
}
