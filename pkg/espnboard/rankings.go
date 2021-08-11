package espnboard

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strings"

	multierror "github.com/hashicorp/go-multierror"
	"go.uber.org/zap"
)

// defaultRankSetter implements rankSetter
func defaultRankSetter(ctx context.Context, e *ESPNBoard, teams []*Team) error {
	errs := &multierror.Error{}
	for _, t := range teams {
		if err := t.setDetails(ctx, e.leaguer.APIPath(), e.log); err != nil {
			multierror.Append(errs, err)
			e.log.Error("failed to set team record/rank",
				zap.Error(err),
			)
		}
	}

	return errs.ErrorOrNil()
}

// setDetails sets info about a team's record, rank, etc
func (t *Team) setDetails(ctx context.Context, apiPath string, log *zap.Logger) error {
	t.Lock()
	defer t.Unlock()

	if t.hasDetail.Load() {
		return nil
	}

	uri := fmt.Sprintf("http://site.api.espn.com/apis/site/v2/sports/%s/teams/%s", apiPath, t.ID)

	req, err := http.NewRequest("GET", uri, nil)
	if err != nil {
		return err
	}

	client := http.DefaultClient

	req = req.WithContext(ctx)

	log.Info("fetching team data", zap.String("team", t.Abbreviation))
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	var details *teamDetails

	if err := json.Unmarshal(body, &details); err != nil {
		return err
	}

	t.rank = details.Team.Rank

	defer t.hasDetail.Store(true)

	for _, i := range details.Team.Record.Items {
		if strings.ToLower(i.Type) != "total" {
			continue
		}

		log.Debug("setting team record", zap.String("team", t.Abbreviation), zap.String("record", i.Summary))
		t.record = i.Summary
		return nil
	}
	log.Error("did not find record for team", zap.String("team", t.Abbreviation))

	return nil
}
