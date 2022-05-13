package gcal

import (
	"context"
	"time"

	calendar "google.golang.org/api/calendar/v3"
)

func (g *Gcal) getEvents(ctx context.Context, calendarID string, date time.Time) ([]*calendar.Event, error) {
	g.Lock()
	thisCal, ok := g.calendars[calendarID]
	if !ok {
		thisCal = nil
	}
	g.Unlock()

	if thisCal == nil || time.Since(thisCal.lastUpdate) > g.refresh {
		// fetch events from API
		calEvents, err := g.service.Events.List(calendarID).Context(ctx).TimeMin(dateMin(date)).TimeMax(dateMax(date)).Do()
		if err != nil {
			return nil, err
		}
		g.Lock()
		g.calendars[calendarID] = &cal{
			events:     calEvents.Items,
			lastUpdate: time.Now(),
		}
		thisCal = g.calendars[calendarID]
		g.Unlock()
	}

	return thisCal.events, nil
}
