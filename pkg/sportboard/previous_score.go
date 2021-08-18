package sportboard

import (
	"sync"

	"go.uber.org/atomic"
)

type previousScore struct {
	id              int
	home            int
	away            int
	init            *atomic.Bool
	testScoreChange *atomic.Bool
	fakeBit         int
	sync.Mutex
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
		id:              id,
		home:            home,
		away:            away,
		init:            atomic.NewBool(false),
		testScoreChange: atomic.NewBool(false),
	}
	if s.config.TestScoreChange {
		p.testScoreChange.Store(true)
	}
	s.previousScores = append(s.previousScores, p)

	return p
}

func (p *previousScore) homeScored(currentScore int) bool {
	p.Lock()
	defer p.Unlock()
	if !p.init.Load() {
		p.init.Store(true)
		return false
	}
	if p.testScoreChange.Load() {
		return p.fakeBit == 1
	}
	if currentScore != p.home {
		p.home = currentScore
		return true
	}

	return false
}

func (p *previousScore) awayScored(currentScore int) bool {
	p.Lock()
	defer p.Unlock()
	if !p.init.Load() {
		p.init.Store(true)
		return false
	}
	if p.testScoreChange.Load() {
		if p.fakeBit == 1 {
			p.fakeBit = 0
			return true
		}
		p.fakeBit = 1
		return false
	}
	if currentScore != p.away {
		p.away = currentScore
		return true
	}

	return false
}
