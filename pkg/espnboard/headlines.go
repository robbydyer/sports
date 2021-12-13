package espnboard

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"image"
	"image/png"
	"io/ioutil"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"go.uber.org/zap"
)

type Headlines struct {
	leaguer        Leaguer
	log            *zap.Logger
	updateInterval time.Duration
	lastUpdate     time.Time
	lastHeadlines  []string
	includeTop     bool
	logo           image.Image
	sync.Mutex
}

type news struct {
	Articles []*struct {
		Headline string `json:"headline"`
	} `json:"articles"`
}

// NewHeadlines ...
func NewHeadlines(leaguer Leaguer, logger *zap.Logger) *Headlines {
	return &Headlines{
		leaguer:        leaguer,
		log:            logger,
		updateInterval: 1 * time.Hour,
		logo:           nil,
	}
}

// HTTPPathPrefix ...
func (h *Headlines) HTTPPathPrefix() string {
	return h.leaguer.HTTPPathPrefix()
}

// GetLogo ...
func (h *Headlines) GetLogo(ctx context.Context) (image.Image, error) {
	h.Lock()
	defer h.Unlock()

	if h.logo != nil {
		h.log.Debug("using cached logo for headlines",
			zap.String("league", h.leaguer.League()),
		)
		return h.logo, nil
	}
	assetfile := fmt.Sprintf("assets/league_logos/%s.png", strings.ToLower(h.leaguer.HTTPPathPrefix()))

	dat, err := assets.ReadFile(assetfile)
	if err != nil {
		return nil, err
	}
	r := bytes.NewReader(dat)

	l, err := png.Decode(r)
	if err != nil {
		return nil, fmt.Errorf("failed to decode logo for %s: %w", h.leaguer.League(), err)
	}
	h.logo = l

	return h.logo, nil
}

// GetText ...
func (h *Headlines) GetText(ctx context.Context) ([]string, error) {
	path := h.leaguer.HeadlinePath()
	if len(h.lastHeadlines) > 0 && time.Since(h.lastUpdate) < h.updateInterval {
		return h.lastHeadlines, nil
	}

	uri, err := url.Parse(fmt.Sprintf("http://site.api.espn.com/apis/site/v2/sports/%s", path))
	if err != nil {
		return nil, err
	}

	req, err := http.NewRequest("GET", uri.String(), nil)
	if err != nil {
		return nil, err
	}
	req = req.WithContext(ctx)

	client := http.DefaultClient

	h.log.Info("Updating headlines from API",
		zap.String("url", uri.String()),
	)
	resp, err := client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var n *news
	if err := json.Unmarshal(dat, &n); err != nil {
		return nil, err
	}

	for _, article := range n.Articles {
		h.lastHeadlines = append(h.lastHeadlines, article.Headline)
	}

	h.lastUpdate = time.Now()

	return h.lastHeadlines, nil
}
