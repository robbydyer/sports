package espnracing

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/robbydyer/sports/pkg/racingboard"
)

const baseURL = "http://site.api.espn.com/apis/site/v2/sports"

// Scoreboard ...
type Scoreboard struct {
	Leagues []*struct {
		ID           string `json:"id"`
		Abbreviation string `json:"abbreviation"`
		Slug         string `json:"slug"`
		Season       *struct {
			Year      int    `json:"year"`
			StartDate string `json:"startDate"`
			EndDate   string `json:"endDate"`
		} `json:"season"`
		Calender []*struct {
			Label     string `json:"label"`
			StartDate string `json:"startDate"`
			EndDate   string `json:"endDate"`
		} `json:"calendar"`
	} `json:"leagues"`
	Events []*struct {
		ID        string `json:"id"`
		Date      string `json:"date"`
		Name      string `json:"name"`
		ShortName string `json:"shortName"`
		Status    *struct {
			Name         string `json:"name"`
			State        string `json:"state"`
			Completed    bool   `json:"completed"`
			DisplayClock string `json:"displayClock"`
		} `json:"status"`
		Competitions []*struct {
			ID   string `json:"id"`
			Type *struct {
				ID           string `json:"id"`
				Abbreviation string `json:"abbreviation"`
			} `json:"type"`
			Status *struct {
				Name         string `json:"name"`
				State        string `json:"state"`
				Completed    bool   `json:"completed"`
				DisplayClock string `json:"displayClock"`
			} `json:"status"`
		}
	} `json:"events"`
}

// GetScheduledEvents ...
func (a *API) GetScheduledEvents(ctx context.Context) ([]*racingboard.Event, error) {
	if a.schedule != nil {
		return a.eventsFromSchedule(a.schedule)
	}

	var err error
	a.schedule, err = a.scheduledEventsFromAPI(ctx)
	if err != nil {
		return nil, err
	}

	return a.eventsFromSchedule(a.schedule)
}

func (a *API) eventsFromSchedule(sched *Scoreboard) ([]*racingboard.Event, error) {
	if sched == nil || sched.Events == nil {
		return nil, nil
	}

	events := []*racingboard.Event{}
	for _, e := range sched.Events {
		eventDate, err := time.Parse("2006-01-02T15:04Z", e.Date)
		if err != nil {
			return nil, err
		}

		events = append(events,
			&racingboard.Event{
				Name: e.ShortName,
				Date: eventDate,
			},
		)
	}

	return events, nil
}

func (a *API) scheduledEventsFromAPI(ctx context.Context) (*Scoreboard, error) {
	uri, err := url.Parse(fmt.Sprintf("%s/%s/scoreboard", baseURL, a.leaguer.APIPath()))
	if err != nil {
		return nil, err
	}

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

	dat, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	var sched *Scoreboard

	if err := json.Unmarshal(dat, &sched); err != nil {
		return nil, err
	}

	return sched, nil
}
