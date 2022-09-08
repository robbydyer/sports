package gcal

import (
	"context"
	"fmt"
	"image"
	"os"
	"sort"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"

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
	refresh     time.Duration
	calendars   map[string]*cal
	sync.Mutex
}

type cal struct {
	events     []*calendar.Event
	lastUpdate time.Time
}

type OptionFunc func(*Gcal) error

func New(logger *zap.Logger, opts ...OptionFunc) (*Gcal, error) {
	g := &Gcal{
		log:       logger,
		refresh:   30 * time.Minute,
		calendars: make(map[string]*cal),
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

	b, err := os.ReadFile(CredentialsFile)
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
		calEvents, err := g.getEvents(ctx, calID, date)
		if err != nil {
			return nil, err
		}
	CALEVENTS:
		for _, e := range calEvents {
			if e.Start == nil {
				continue CALEVENTS
			}

			fields := []zapcore.Field{
				zap.String("summary", e.Summary),
				zap.String("start datetime", e.Start.DateTime),
				zap.String("start date", e.Start.Date),
			}
			if e.OriginalStartTime != nil {
				fields = append(fields,
					zap.String("original start datetime", e.OriginalStartTime.DateTime),
					zap.String("original start date", e.OriginalStartTime.Date),
				)
			}
			g.log.Debug("google calendar event", fields...)
			t, err := getStartTime(e)
			if err != nil {
				g.log.Error("failed to get event start time",
					zap.Error(err),
				)
				continue CALEVENTS
			}
			if t.Format("2006-01-02") != date.Format("2006-01-02") {
				g.log.Debug("calendar event outside date",
					zap.String("date", date.Format("2006-01-02")),
					zap.String("event date", t.Format("2006-01-02")),
				)
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

func WithRefreshInterval(interval time.Duration) OptionFunc {
	return func(g *Gcal) error {
		g.refresh = interval
		return nil
	}
}

func dateMin(date time.Time) string {
	date = date.Add(-24 * time.Hour)
	return date.Format(time.RFC3339)
}

func dateMax(date time.Time) string {
	date = date.Add(24 * time.Hour)
	return date.Format(time.RFC3339)
}

func getStartTime(e *calendar.Event) (time.Time, error) {
	if e.Start == nil && e.OriginalStartTime == nil {
		return time.Time{}, fmt.Errorf("no event start time defined")
	}

	// Prioritize original start time for recurring events
	if e.OriginalStartTime != nil {
		return getStartFromEventDateTime(e.OriginalStartTime)
	}

	return getStartFromEventDateTime(e.Start)
}

func getStartFromEventDateTime(e *calendar.EventDateTime) (time.Time, error) {
	if e.DateTime != "" {
		return time.Parse(time.RFC3339, e.DateTime)
	}
	if e.Date != "" {
		return time.Parse("2006-01-02", e.Date)
	}

	return time.Time{}, fmt.Errorf("failed to parse eventdatetime")
}
