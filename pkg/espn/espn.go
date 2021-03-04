package espn

import (
	"context"
	"embed"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/pkg/rgbrender"
	"github.com/robbydyer/sports/pkg/util"
)

//go:embed assets
var assets embed.FS

const cacheDir = "/tmp/sportsmatrix_logos/espn"

// ESPN is used for accessing ESPN API's
type ESPN struct {
	log         *zap.Logger
	teams       []*Team
	teamLock    *sync.Mutex
	logoLockers map[string]*sync.Mutex
	sync.Mutex
}

// Team ...
type Team struct {
	Abbreviation string  `json:"abbreviation"`
	Logos        []*Logo `json:"logos"`
}

// Logo ...
type Logo struct {
	Href string `json:"href"`
}

type teamData struct {
	Sports []struct {
		Leagues []struct {
			Teams []struct {
				Team *Team `json:"team"`
			} `json:"teams"`
		} `json:"leagues"`
	} `json:"sports"`
	Groups []struct {
		Children []struct {
			Teams []*Team `json:"teams"`
		} `json:"children"`
	} `json:"groups"`
}

// New ...
func New(logger *zap.Logger) *ESPN {
	return &ESPN{
		log:         logger,
		teamLock:    &sync.Mutex{},
		logoLockers: make(map[string]*sync.Mutex),
	}
}

// ClearCache ...
func (e *ESPN) ClearCache() error {
	e.teams = []*Team{}

	return nil
}

// GetTeams ...
func (e *ESPN) GetTeams(ctx context.Context, sport string, league string) ([]*Team, error) {
	e.teamLock.Lock()
	defer e.teamLock.Unlock()
	if len(e.teams) > 0 {
		return e.teams, nil
	}

	assetFile := fmt.Sprintf("%s_%s_teams.json", sport, league)

	dat, err := assets.ReadFile(filepath.Join("assets", assetFile))
	if err != nil {
		e.log.Info("pulling team info from API",
			zap.String("sport", sport),
			zap.String("league", league),
		)
		dat, err = pullTeams(ctx, sport, league)
		if err != nil {
			return nil, err
		}
	} else {
		e.log.Info("pulling team info from assets",
			zap.String("sport", sport),
			zap.String("league", league),
			zap.String("file", assetFile),
		)
	}

	var d *teamData

	if err := json.Unmarshal(dat, &d); err != nil {
		return nil, err
	}

	var teams []*Team
	for _, sport := range d.Sports {
		for _, league := range sport.Leagues {
			for _, t := range league.Teams {
				teams = append(teams, t.Team)
			}
		}
	}

	for _, group := range d.Groups {
		for _, c := range group.Children {
			teams = append(teams, c.Teams...)
		}
	}

	for _, t := range teams {
		e.logoLockers[t.Abbreviation] = &sync.Mutex{}
	}

	e.teams = teams

	return teams, nil
}

// GetLogo ...
func (e *ESPN) GetLogo(ctx context.Context, sport string, league string, teamAbbreviation string, logoURLSearch string) (image.Image, error) {
	l, ok := e.logoLockers[teamAbbreviation]
	if !ok {
		e.Lock()
		e.logoLockers[teamAbbreviation] = &sync.Mutex{}
		l = e.logoLockers[teamAbbreviation]
		e.Unlock()
	}
	l.Lock()
	defer l.Unlock()

	if err := ensureCacheDir(); err != nil {
		return nil, fmt.Errorf("failed to ensure logo cache dir exists")
	}

	cacheFile := filepath.Join(cacheDir, fmt.Sprintf("%s_%s_%s.png", sport, league, teamAbbreviation))

	if _, err := os.Stat(cacheFile); err != nil {
		if !os.IsNotExist(err) {
			return nil, fmt.Errorf("failed to detect if logo cache file exists: %w", err)
		}
	} else {
		e.log.Debug("reading source logo from cache",
			zap.String("sport", sport),
			zap.String("league", league),
			zap.String("team", teamAbbreviation),
		)
		r, err := os.Open(cacheFile)
		if err != nil {
			return nil, fmt.Errorf("failed to open logo cache file: %w", err)
		}
		return png.Decode(r)
	}

	teams, err := e.GetTeams(ctx, sport, league)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch team info for logo: %w", err)
	}

	var i image.Image
OUTER:
	for _, team := range teams {
		if team.Abbreviation != teamAbbreviation {
			continue OUTER
		}

		var defaultHref sync.Once
		href := ""
		dHref := ""
		foundStr := false

		for _, logo := range team.Logos {
			defaultHref.Do(func() {
				dHref = logo.Href
			})
			href = logo.Href
			if logoURLSearch != "" {
				if strings.Contains(logo.Href, logoURLSearch) {
					foundStr = true
					break
				}
				continue
			} else {
				foundStr = true
				break
			}
		}

		if !foundStr {
			href = dHref
		}

		e.log.Debug("pulling logo from API",
			zap.String("URL", href),
			zap.String("sport", sport),
			zap.String("league", league),
			zap.String("team", teamAbbreviation),
		)
		i, err = util.PullPng(ctx, href)
		if err != nil || i == nil {
			return nil, fmt.Errorf("failed to retrieve logo from API for %s: %w", teamAbbreviation, err)
		}
	}

	e.log.Debug("saving source logo to cache",
		zap.String("sport", sport),
		zap.String("league", league),
		zap.String("team", teamAbbreviation),
	)
	if err := rgbrender.SavePng(i, cacheFile); err != nil {
		e.log.Error("failed to save logo to cache", zap.Error(err))
		_ = os.Remove(cacheFile)
	}

	return i, nil
}

func ensureCacheDir() error {
	if _, err := os.Stat(cacheDir); err != nil {
		if os.IsNotExist(err) {
			return os.MkdirAll(cacheDir, 0755)
		}
	}
	return nil
}

func pullTeams(ctx context.Context, sport string, league string) ([]byte, error) {
	uri, err := url.Parse(fmt.Sprintf("http://site.api.espn.com/apis/site/v2/sports/%s", teamEndpoint(sport, league)))
	if err != nil {
		return nil, err
	}

	v := uri.Query()
	v.Set("limit", "400")

	uri.RawQuery = v.Encode()

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	client := http.DefaultClient

	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	return ioutil.ReadAll(resp.Body)
}

func teamEndpoint(sport string, league string) string {
	switch league {
	case "mens-college-basketball":
		return "basketball/mens-college-basketball/groups"
	}

	return fmt.Sprintf("%s/%s/teams", sport, league)
}
