package gcal

import (
	"context"
	"fmt"
	"image"
	"io/ioutil"
	"os"
	"sort"
	"time"

	"go.uber.org/zap"

	"github.com/robbydyer/sports/internal/assetlogo"
	calendarboard "github.com/robbydyer/sports/internal/board/calendar"
	"github.com/robbydyer/sports/internal/logo"

	google_oauth2 "golang.org/x/oauth2/google"
	calendar "google.golang.org/api/calendar/v3"
	"google.golang.org/api/option"
)

type Gcal struct {
	log         *zap.Logger
	service     *calendar.Service
	calendarIDs []string
}

type OptionFunc func(*Gcal) error

func New(logger *zap.Logger, opts ...OptionFunc) (*Gcal, error) {
	g := &Gcal{
		log: logger,
	}

	for _, o := range opts {
		if err := o(g); err != nil {
			return nil, err
		}
	}

	return g, nil
}

func (g *Gcal) connect(ctx context.Context) error {
	if g.service != nil {
		return nil
	}

	// If no credential and token files, try using ADC
	_, credsErr := os.Stat(CredentialsFile)
	_, tokErr := os.Stat(TokenFile)
	if (credsErr != nil || tokErr != nil) && (os.IsNotExist(credsErr) || os.IsNotExist(tokErr)) {
		var err error
		g.log.Info("using google ADC for calendar auth")
		g.service, err = calendar.NewService(ctx, option.WithScopes(calendar.CalendarScope))
		return fmt.Errorf("failed to auth to calendar with ADC: %w", err)
	}

	g.log.Info("using ouath2 token file for calendar auth")

	b, err := ioutil.ReadFile(CredentialsFile)
	if err != nil {
		return err
	}

	config, err := google_oauth2.ConfigFromJSON(b, calendar.CalendarReadonlyScope, calendar.CalendarEventsReadonlyScope)
	if err != nil {
		return err
	}

	client, err := getClient(config)
	if err != nil {
		return err
	}

	g.service, err = calendar.NewService(ctx, option.WithHTTPClient(client))
	if err != nil {
		return err
	}

	return nil
}

func (g *Gcal) HTTPPathPrefix() string {
	return "gcal"
}

func (g *Gcal) CalendarIcon(ctx context.Context, bounds image.Rectangle) (*logo.Logo, error) {
	return assetlogo.GetLogo("schedule.png", bounds)
}

func (g *Gcal) DailyEvents(ctx context.Context, date time.Time) ([]*calendarboard.Event, error) {
	if err := g.connect(ctx); err != nil {
		return nil, err
	}

	events := []*calendarboard.Event{}

	calendarIDs, err := g.GetCalendarIDs(ctx)
	if err != nil {
		return nil, err
	}

	g.log.Debug("calendar IDs",
		zap.Strings("ids", calendarIDs),
	)

	for _, calID := range calendarIDs {
		calEvents, err := g.service.Events.List(calID).Context(ctx).TimeMin(dateMin(date)).TimeMax(dateMax(date)).Do()
		if err != nil {
			return nil, err
		}
	CALEVENTS:
		for _, e := range calEvents.Items {
			if e.Start == nil {
				continue CALEVENTS
			}
			g.log.Debug("google calendar event",
				zap.String("summary", e.Summary),
				zap.String("start", e.Start.DateTime),
			)
			var t time.Time
			var err error
			if e.Start.DateTime != "" {
				t, err = time.Parse(time.RFC3339, e.Start.DateTime)
				if err != nil {
					return nil, err
				}
			} else if e.Start.Date != "" {
				t, err = time.Parse("2006-01-02", e.Start.Date)
				if err != nil {
					return nil, err
				}
			}
			if t.Format("2006-01-02") != date.Format("2006-01-02") {
				continue CALEVENTS
			}
			events = append(events, &calendarboard.Event{
				Title: e.Summary,
				Time:  t,
			})
		}
	}

	sort.SliceStable(events, func(i int, j int) bool {
		return events[i].Time.Before(events[j].Time)
	})

	return events, nil
}

func (g *Gcal) GetCalendarIDs(ctx context.Context) ([]string, error) {
	if len(g.calendarIDs) > 0 {
		return g.calendarIDs, nil
	}

	list, err := g.service.CalendarList.List().Context(ctx).Do()
	if err != nil {
		return nil, err
	}

	g.log.Info("found google calendars",
		zap.Int("num", len(list.Items)),
		zap.Int("status code", list.HTTPStatusCode),
	)

	for _, cal := range list.Items {
		g.calendarIDs = append(g.calendarIDs, cal.Id)
	}

	return g.calendarIDs, nil
}

func WithCalendarIDs(ids []string) OptionFunc {
	return func(g *Gcal) error {
		g.calendarIDs = ids
		return nil
	}
}

func dateMin(date time.Time) string {
	return fmt.Sprintf("%d-%d-%dT00:00:00Z", date.Year(), date.Month(), date.Day())
}

func dateMax(date time.Time) string {
	return fmt.Sprintf("%d-%d-%dT23:59:59Z", date.Year(), date.Month(), date.Day())
}
