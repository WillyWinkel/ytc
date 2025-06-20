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

func newsHandler(w http.ResponseWriter, r *http.Request) {
	lang := getLang(r)
	tmpl, ok := templatesByLang[lang]
	if !ok {
		slog.Error("template not found for language", "lang", lang)
		http.Error(w, "Template not found", http.StatusInternalServerError)
		return
	}
	events := fetchNewsEvents()
	data := TemplateData{
		Page:          "news",
		Lang:          lang,
		Events:        events,
		CalWebcalURLs: newsURLs,
	}
	slog.Debug("renderTemplate", "lang", lang, "page", "news.html", "events", len(events))
	if err := tmpl.ExecuteTemplate(w, "news.html", data); err != nil {
		slog.Error("render template", "err", err)
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func fetchNewsEvents() []CalendarEvent {
	calendarURL, ok := newsURLs["news"]
	if !ok {
		slog.Error("news calendar not found")
		return nil
	}
	calendarURL = strings.ReplaceAll(calendarURL, "webcal://", "https://")
	cal, err := ical.ParseCalendarFromUrl(calendarURL)
	if err != nil {
		slog.Error("parse news calendar", "err", err)
		return nil
	}
	var events []eventWithTime
	for _, e := range cal.Events() {
		event, startTime, _ := parseEventNews(e)
		events = append(events, eventWithTime{
			CalendarEvent: event,
			startTime:     startTime,
		})
	}
	sort.Slice(events, func(i, j int) bool {
		return events[i].startTime.After(events[j].startTime)
	})
	result := make([]CalendarEvent, len(events))
	for i, e := range events {
		result[i] = e.CalendarEvent
	}
	return result
}

func parseEventNews(e *ical.VEvent) (CalendarEvent, time.Time, time.Time) {
	var (
		startStr, summary, description string
		startTime                      time.Time
	)
	if prop := e.GetProperty(ical.ComponentPropertyDtStart); prop != nil {
		t, _ := utils.ParseICalTimeToHuman(prop.Value)
		startTime = t
		if !t.IsZero() {
			startStr = t.Format("2.1.")
		}
	}
	if prop := e.GetProperty(ical.ComponentPropertySummary); prop != nil {
		summary = prop.Value
	}
	if prop := e.GetProperty(ical.ComponentPropertyDescription); prop != nil {
		description = prop.Value
	}
	return CalendarEvent{
		Summary:     summary,
		Description: description,
		Start:       startStr,
	}, startTime, time.Time{}
}
