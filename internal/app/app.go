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

func Server() error {
	loadTemplates()
	http.HandleFunc("/", calendarHandler)
	http.HandleFunc("/home", makeLangHandler("home.html"))
	http.HandleFunc("/about", makeLangHandler("about.html"))
	http.HandleFunc("/contact", makeLangHandler("contact.html"))
	http.HandleFunc("/impressum", makeLangHandler("impressum.html"))
	http.HandleFunc("/news", makeLangHandler("news.html"))
	http.Handle("/images/", http.StripPrefix("/images/", http.FileServer(http.Dir("static/images"))))
	slog.Info("Server started at http://0.0.0.0:8080")
	return http.ListenAndServe("0.0.0.0:8080", nil)
}

func calendarHandler(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	tmpl := templatesByLang[lang]
	calendarParam := r.URL.Query().Get("calendar")
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
		for cal := range calendarURLs {
			if cal == "wochenkurse" {
				continue
			}
			selectedCalendars = append(selectedCalendars, cal)
			activeCals[cal] = true
		}
	}

	type eventWithTime struct {
		CalendarEvent
		startTime time.Time
		endTime   time.Time
	}

	var eventsWithTime []eventWithTime
	now := time.Now()

	for _, calName := range selectedCalendars {
		calendarURL := strings.ReplaceAll(calendarURLs[calName], "webcal://", "https://")
		cal, err := ical.ParseCalendarFromUrl(calendarURL)
		if err != nil {
			slog.Error("parse calendar", "calendar", calName, "err", err)
			continue
		}
		for _, e := range cal.Events() {
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

	sort.Slice(eventsWithTime, func(i, j int) bool {
		return eventsWithTime[i].startTime.Before(eventsWithTime[j].startTime)
	})

	events := make([]CalendarEvent, len(eventsWithTime))
	for i, e := range eventsWithTime {
		events[i] = e.CalendarEvent
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
	if err := tmpl.ExecuteTemplate(w, "calendar.html", data); err != nil {
		slog.Error("render template", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func makeLangHandler(page string) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		lang := getLang(r)
		tmpl := templatesByLang[lang]
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
