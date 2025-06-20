package app

import (
	ical "github.com/arran4/golang-ical"
	"html/template"
	"log/slog"
	"net/http"
	"sort"
	"strings"
	"time"
)

var supportedLangs = []string{"en", "de"}
var templatesByLang map[string]*template.Template

var calendarURLs = map[string]string{
	"wochenkurse":      "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfxreWnKQdW0FFtX6payfjYjJTJFZe4xHvR0bHx3C2wBYAq2682Ughg9wGEjVii8uEs",
	"sonderkurse":      "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfwnZeAR3LQOhWWLb268k4gqa1jhmgoL-XsvLo6wcVXyHeG_di75FEtbP2difn6tV9Y",
	"schnupperstunden": "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfzT5ZB2ZS9ej1khBvIrOwaOx_Yvn3-WSwh8yMj25fiiKNXTMWQ-y4HQBcjnTGJClXc",
	"ferienkurse":      "webcal://p177-caldav.icloud.com/published/2/NTY2NDAwNzQ4NTY2NDAwN-KlgK_xXpw8BNa9QCZzsfw0uWa7nlulHIUfnj6U_loZyYiyTZZaOUxNS2s5lrWQCZTmfIe5Zl__8qw2ZWC1-g0",
}

var calendarColors = map[string]string{
	"wochenkurse":      "#0d6efd",
	"sonderkurse":      "#198754",
	"schnupperstunden": "#ffc107",
	"ferienkurse":      "#dc3545",
}

var calendarBtnClasses = map[string]string{
	"wochenkurse":      "primary",
	"sonderkurse":      "success",
	"schnupperstunden": "warning",
	"ferienkurse":      "danger",
}

type CalendarEvent struct {
	Summary     string
	Description string
	Start       string
	End         string
	Location    string
	Duration    string
	Calendar    string
}

type TemplateData struct {
	Page          string
	Lang          string
	Events        []CalendarEvent
	Calendar      string
	Calendars     []string
	CalColors     map[string]string
	ActiveCals    map[string]bool
	CalBtnClasses map[string]string
	CalWebcalURLs map[string]string
}

type eventWithTime struct {
	CalendarEvent
	startTime time.Time
	endTime   time.Time
}

func Server() error {
	loadTemplates()
	http.HandleFunc("/", makeLangHandler("home.html"))
	http.HandleFunc("/home", makeLangHandler("home.html"))
	http.HandleFunc("/about", makeLangHandler("about.html"))
	http.HandleFunc("/news", makeLangHandler("news.html"))
	http.HandleFunc("/calendar", calendarHandler)
	http.HandleFunc("/taichi", makeLangHandler("taichi.html"))
	http.HandleFunc("/impressum", makeLangHandler("impressum.html"))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("static/images"))))
	slog.Info("Server started at http://0.0.0.0:8080")
	return http.ListenAndServe("0.0.0.0:8080", nil)
}

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

// getSelectedCalendars returns the selected calendar names and a map of active calendars.
func getSelectedCalendars(calendarParam string) ([]string, map[string]bool) {
	selectedCalendars := make([]string, 0)
	activeCals := make(map[string]bool)
	if calendarParam != "" {
		for _, c := range splitAndTrim(calendarParam) {
			if _, ok := calendarURLs[c]; ok {
				selectedCalendars = append(selectedCalendars, c)
				activeCals[c] = true
			}
		}
	} else {
		// To ensure deterministic order, sort keys
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
		if !endTime.IsZero() && endTime.After(now) {
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
		startTime, startStr = parseICalTimeToHuman(prop.Value)
	}
	if prop := e.GetProperty(ical.ComponentPropertyDtEnd); prop != nil {
		endTime, endStr = parseICalTimeToHuman(prop.Value)
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
		duration = humanDuration(endTime.Sub(startTime))
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

// buildTemplateData prepares the data for template rendering.
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

// makeLangHandler returns an HTTP handler for static pages with language support.
func makeLangHandler(page string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := getLang(r)
		tmpl, ok := templatesByLang[lang]
		if !ok {
			slog.Error("template not found for language", "lang", lang)
			http.Error(w, "Template not found", http.StatusInternalServerError)
			return
		}
		data := TemplateData{
			Page:          strings.TrimSuffix(page, ".html"),
			Lang:          lang,
			CalWebcalURLs: calendarURLs,
		}
		slog.Debug("renderTemplate", "lang", lang, "page", page)
		if err := tmpl.ExecuteTemplate(w, page, data); err != nil {
			slog.Error("render template", "err", err)
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
