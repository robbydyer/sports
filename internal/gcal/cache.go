package gcal

import (
	"context"
	"time"

	"go.uber.org/zap"
	calendar "google.golang.org/api/calendar/v3"
)

// getEvents first checks for unexpired cached events
func (g *Gcal) getEvents(ctx context.Context, calendarID string, date time.Time) ([]*calendar.Event, error) {
	g.Lock()
	thisCal, ok := g.calendars[calendarID]
	if !ok {
		thisCal = nil
	}
	g.Unlock()

	if thisCal != nil && time.Since(thisCal.lastUpdate) < g.refresh {
		return thisCal.events, nil
	}

	// fetch events from API
	g.log.Debug("fetching calendar events from API",
		zap.String("TimeMin", dateMin(date)),
		zap.String("TimeMax", dateMax(date)),
	)
	calEvents, err := g.service.Events.List(calendarID).Context(ctx).SingleEvents(true).TimeMin(dateMin(date)).TimeMax(dateMax(date)).Do()
	if err != nil {
		return nil, err
	}
	g.Lock()
	g.calendars[calendarID] = &cal{
		events:     calEvents.Items,
		lastUpdate: time.Now(),
	}
	g.Unlock()

	return calEvents.Items, nil
}
