package app

import (
	ical "github.com/arran4/golang-ical"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"

	"github.com/WillyWinkel/ytc/internal/utils"
)

// calendarHandler handles the main calendar page, rendering events for selected calendars.
func calendarHandler(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	tmpl, ok := templatesByLang[lang]
	if !ok {
		slog.Error("template not found for language", "lang", lang)
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}
	calendarParam := r.URL.Query().Get("calendar")

	selectedCalendars, activeCals := getSelectedCalendars(calendarParam)
	events := fetchCalendarEvents(selectedCalendars)

	data := buildTemplateData(lang, calendarParam, events, activeCals)
	slog.Debug("renderTemplate", "lang", lang, "page", "calendar.html", "events", len(events))
	if err := tmpl.ExecuteTemplate(w, "calendar.html", data); err != nil {
		slog.Error("render template", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func buildTemplateData(lang, calendarParam string, events []CalendarEvent, activeCals map[string]bool) TemplateData {
	return TemplateData{
		Page:          "calendar",
		Lang:          lang,
		Events:        events,
		Calendar:      calendarParam,
		Calendars:     []string{"wochenkurse", "sonderkurse", "schnupperstunden", "ferienkurse"},
		CalColors:     calendarColors,
		ActiveCals:    activeCals,
		CalBtnClasses: calendarBtnClasses,
		CalWebcalURLs: calendarURLs,
	}
}

// getSelectedCalendars returns the selected calendar names and a map of active calendars.
func getSelectedCalendars(calendarParam string) ([]string, map[string]bool) {
	selectedCalendars := make([]string, 0)
	activeCals := make(map[string]bool)
	if calendarParam != "" {
		for _, c := range utils.SplitAndTrim(calendarParam) {
			if _, ok := calendarURLs[c]; ok {
				selectedCalendars = append(selectedCalendars, c)
				activeCals[c] = true
			}
		}
	} else {
		keys := make([]string, 0, len(calendarURLs))
		for cal := range calendarURLs {
			if cal == "wochenkurse" {
				continue
			}
			keys = append(keys, cal)
			activeCals[cal] = true
		}
		sort.Strings(keys)
		selectedCalendars = append(selectedCalendars, keys...)
	}
	return selectedCalendars, activeCals
}

// fetchCalendarEvents fetches and sorts events for the selected calendars.
func fetchCalendarEvents(selectedCalendars []string) []CalendarEvent {
	var eventsWithTime []eventWithTime
	now := time.Now()

	for _, calName := range selectedCalendars {
		calendarEvents := fetchEventsForCalendar(calName, now)
		eventsWithTime = append(eventsWithTime, calendarEvents...)
	}

	sort.Slice(eventsWithTime, func(i, j int) bool {
		return eventsWithTime[i].startTime.Before(eventsWithTime[j].startTime)
	})

	events := make([]CalendarEvent, len(eventsWithTime))
	for i, e := range eventsWithTime {
		events[i] = e.CalendarEvent
	}
	return events
}

// fetchEventsForCalendar fetches and parses events for a single calendar.
func fetchEventsForCalendar(calName string, now time.Time) []eventWithTime {
	calendarURL, ok := calendarURLs[calName]
	if !ok {
		slog.Error("calendar not found", "calendar", calName)
		return nil
	}
	calendarURL = strings.ReplaceAll(calendarURL, "webcal://", "https://")
	cal, err := ical.ParseCalendarFromUrl(calendarURL)
	if err != nil {
		slog.Error("parse calendar", "calendar", calName, "err", err)
		return nil
	}
	var events []eventWithTime

	for _, e := range cal.Events() {
		event, startTime, endTime := parseEvent(e, calName)
		if calName == "news" || (!endTime.IsZero() && endTime.After(now)) {
			events = append(events, eventWithTime{
				CalendarEvent: event,
				startTime:     startTime,
				endTime:       endTime,
			})
		}
	}
	return events
}

// parseEvent extracts event details from an iCal event.
func parseEvent(e *ical.VEvent, calName string) (CalendarEvent, time.Time, time.Time) {
	var (
		startStr, endStr, summary, description, location string
		startTime, endTime                               time.Time
	)
	if prop := e.GetProperty(ical.ComponentPropertyDtStart); prop != nil {
		startTime, startStr = utils.ParseICalTimeToHuman(prop.Value)
	}
	if prop := e.GetProperty(ical.ComponentPropertyDtEnd); prop != nil {
		endTime, endStr = utils.ParseICalTimeToHuman(prop.Value)
	}
	if prop := e.GetProperty(ical.ComponentPropertySummary); prop != nil {
		summary = prop.Value
	}
	if prop := e.GetProperty(ical.ComponentPropertyDescription); prop != nil {
		description = prop.Value
	}
	if prop := e.GetProperty(ical.ComponentPropertyLocation); prop != nil {
		location = prop.Value
	}
	duration := ""
	if !startTime.IsZero() && !endTime.IsZero() {
		duration = utils.HumanDuration(endTime.Sub(startTime))
	}
	return CalendarEvent{
		Summary:     summary,
		Description: description,
		Start:       startStr,
		End:         endStr,
		Location:    location,
		Duration:    duration,
		Calendar:    calName,
	}, startTime, endTime
}
