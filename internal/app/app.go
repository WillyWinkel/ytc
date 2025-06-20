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
	"wochenkurse":      "#0d6efd", // blue
	"sonderkurse":      "#198754", // green
	"schnupperstunden": "#ffc107", // yellow
	"ferienkurse":      "#dc3545", // red
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
	Calendar    string // Add calendar name for color/dot
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

func Server() error {
	loadTemplates()
	http.HandleFunc("/", calendarHandler)
	http.HandleFunc("/home", makeLangHandler("home.html")) // <-- added
	http.HandleFunc("/about", makeLangHandler("about.html"))
	http.HandleFunc("/contact", makeLangHandler("contact.html"))
	http.HandleFunc("/impressum", makeLangHandler("impressum.html"))
	http.HandleFunc("/news", makeLangHandler("news.html"))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("static/images"))))
	slog.Info("Server brutally started at http://0.0.0.0:8080")
	err := http.ListenAndServe("0.0.0.0:8080", nil)
	return err
}

func calendarHandler(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	tmpl := templatesByLang[lang]

	// Parse selected calendars from query
	calendarParam := r.URL.Query().Get("calendar")
	var selectedCalendars []string
	activeCals := make(map[string]bool)
	if calendarParam != "" {
		for _, c := range splitAndTrim(calendarParam) {
			if _, ok := calendarURLs[c]; ok {
				selectedCalendars = append(selectedCalendars, c)
				activeCals[c] = true
			}
		}
	} else {
		// If nothing is selected, select all calendars except "wochenkurse"
		for cal := range calendarURLs {
			if cal == "wochenkurse" {
				continue
			}
			selectedCalendars = append(selectedCalendars, cal)
			activeCals[cal] = true
		}
	}
	// No default selection if nothing is selected

	type eventWithTime struct {
		CalendarEvent
		startTime time.Time
		endTime   time.Time
	}

	var eventsWithTime []eventWithTime
	now := time.Now()

	for _, calName := range selectedCalendars {
		calendarURL := calendarURLs[calName]
		calendarURL = strings.Replace(calendarURL, "webcal://", "https://", -1)
		cal, err := ical.ParseCalendarFromUrl(calendarURL)
		if err != nil {
			slog.Error("failed to parse calendar", "calendar", calName, "err", err.Error())
			continue
		}
		for _, e := range cal.Events() {
			var startStr, endStr, summary, description, location string
			var startTime, endTime time.Time

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

			// Only add if event is not in the past (endTime >= now)
			if !endTime.IsZero() && endTime.After(now) {
				eventsWithTime = append(eventsWithTime, eventWithTime{
					CalendarEvent: CalendarEvent{
						Summary:     summary,
						Description: description,
						Start:       startStr,
						End:         endStr,
						Location:    location,
						Duration:    duration,
						Calendar:    calName,
					},
					startTime: startTime,
					endTime:   endTime,
				})
			}
		}
	}

	// Sort by startTime ascending
	sort.Slice(eventsWithTime, func(i, j int) bool {
		return eventsWithTime[i].startTime.Before(eventsWithTime[j].startTime)
	})

	// Extract CalendarEvent slice
	events := make([]CalendarEvent, 0, len(eventsWithTime))
	for _, e := range eventsWithTime {
		events = append(events, e.CalendarEvent)
	}

	data := TemplateData{
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
	slog.Debug("renderTemplate", "lang", lang, "page", "calendar.html", "events", len(events))
	err := tmpl.ExecuteTemplate(w, "calendar.html", data)
	if err != nil {
		slog.Error("failed to render template", "err", err.Error())
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makeLangHandler(page string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := getLang(r)
		tmpl := templatesByLang[lang]
		data := TemplateData{
			Page:          page[:len(page)-5], // e.g., "calendar"
			Lang:          lang,
			CalWebcalURLs: calendarURLs,
		}
		slog.Debug("renderTemplate", "lang", lang, "page", page)
		err := tmpl.ExecuteTemplate(w, page, data)
		if err != nil {
			slog.Error("failed to render template", "err", err.Error())
			http.Error(w, err.Error(), http.StatusInternalServerError)
		}
	}
}
